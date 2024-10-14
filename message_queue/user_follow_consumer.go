package message_queue

import (
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var REDIS_WRITE_ERROR = errors.New("redis write error")
var MYSQL_WRITE_ERROR = errors.New("mysql write error")

type UserFollowProcessor struct {
	kafkaReader             *kafka.Reader
	messages                chan kafka.Message
	deadLetterQueueForMySQL chan []UserFollowEvent // 用于存储失败的事件
	deadLetterQueueForRedis chan []UserFollowEvent // 用于存储失败的事件
	maxRetries              int                    // 最大重试次数
}

func NewUserFollowProcessor(brokers []string, topic string, maxRetries int) *UserFollowProcessor {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     "user_follow_event_consumer_group",
		StartOffset: kafka.FirstOffset,
		Partition:   0,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})

	return &UserFollowProcessor{
		kafkaReader:             reader,
		messages:                make(chan kafka.Message),
		deadLetterQueueForMySQL: make(chan []UserFollowEvent, 100), // 设定一个缓冲区
		deadLetterQueueForRedis: make(chan []UserFollowEvent, 100),
		maxRetries:              maxRetries,
	}
}
func (lp *UserFollowProcessor) Start(ctx context.Context) {
	go lp.consumeMessages(ctx)
	go lp.processFollows(ctx)
	go lp.handleDeadLetters(ctx) // 处理死信队列

	// Wait for termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	lp.kafkaReader.Close()
}

func (lp *UserFollowProcessor) consumeMessages(ctx context.Context) {
	for {
		msg, err := lp.kafkaReader.ReadMessage(ctx)
		if err != nil {
			zap.L().Info(fmt.Sprintf("Failed to read message:%v", err))
			continue
		}
		lp.messages <- msg // Send the message to the processing channel
	}
}

func (lp *UserFollowProcessor) processFollows(ctx context.Context) {
	var eventList []UserFollowEvent
	var msgs []kafka.Message
	ticker := time.NewTicker(10 * time.Second) // 定时器，每10秒触发一次
	defer ticker.Stop()                        // 确保退出时停止定时器
	for {
		select {
		case msg := <-lp.messages:
			msgs = append(msgs, msg)
			var userFollowEvent UserFollowEvent
			if err := json.Unmarshal(msg.Value, &userFollowEvent); err != nil {
				zap.L().Info(fmt.Sprintf("Failed to unmarshal message:%v", err))
				continue
			}
			eventList = append(eventList, userFollowEvent) // 收集消息

			// 收集了10条消息，触发
			if len(eventList) >= 10 {
				err := lp.handleFollowRedis(eventList)
				if err != nil {
					zap.L().Info(fmt.Sprintf("Failed to process UserFollowEvent event in redis: %v, moving to dead letter queue\n", err))
					lp.deadLetterQueueForMySQL <- eventList // 添加到死信队列
				}

				err = lp.handleFollowMySQL(eventList)
				if err != nil {
					zap.L().Info(fmt.Sprintf("Failed to process UserFollowEvent event in mysql: %v, moving to dead letter queue\n", err))
					lp.deadLetterQueueForMySQL <- eventList // 添加到死信队列
				}
				commitMessages(lp.kafkaReader, msgs)
				eventList = []UserFollowEvent{}
				msgs = []kafka.Message{}
				// 已经移动到死信队列里了，不需要在留在正常的消息队列里

			}

		case <-ticker.C: // 每隔 10 秒触发
			if len(eventList) > 0 {
				err := lp.handleFollowRedis(eventList)
				if err != nil {
					zap.L().Info(fmt.Sprintf("Failed to process UserFollowEvent event in redis: %v, moving to dead letter queue\n", err))
					lp.deadLetterQueueForMySQL <- eventList // 添加到死信队列
				}

				err = lp.handleFollowMySQL(eventList)
				if err != nil {
					zap.L().Info(fmt.Sprintf("Failed to process UserFollowEvent event in mysql: %v, moving to dead letter queue\n", err))
					lp.deadLetterQueueForMySQL <- eventList // 添加到死信队列
				}
				commitMessages(lp.kafkaReader, msgs)
				eventList = []UserFollowEvent{} // 清空eventList
				msgs = []kafka.Message{}
				// 已经移动到死信队列里了，不需要在留在正常的消息队列里
			}

		case <-ctx.Done():
			return
		}
	}
}

func (lp *UserFollowProcessor) handleFollowRedis(events []UserFollowEvent) error {
	var err error
	// 处理Redis
	for i := 0; i <= lp.maxRetries; i++ {
		err = batchUserFollowRedisOperation(events)
		if err != nil {
			zap.L().Info("Error batch processing user follow events")
		}
		if err == nil {
			return nil // 成功处理
		}
		zap.L().Info(fmt.Sprintf("Error processing batch operate redis events, retrying... (%d/%d): %v\n", i+1, lp.maxRetries, err))
		time.Sleep(100 * time.Millisecond) // 等待后重试
	}
	zap.L().Error(fmt.Sprintf("max retries reached for event: %v", events), zap.Error(REDIS_WRITE_ERROR))
	return REDIS_WRITE_ERROR
}

// 处理关注事件，带有不同的处理策略
func (lp *UserFollowProcessor) handleFollowMySQL(events []UserFollowEvent) error {
	var err error
	// 处理MySQL
	for i := 0; i <= lp.maxRetries; i++ {
		err = batchUserFollowDatabaseOperation(events)
		if err != nil {
			zap.L().Info("Error batch processing user follow events")
		}
		if err == nil {
			return nil // 成功处理
		}
		zap.L().Info(fmt.Sprintf("Error processing batch operate mysql database events, retrying... (%d/%d): %v\n", i+1, lp.maxRetries, err))
		time.Sleep(100 * time.Millisecond) // 等待后重试
	}
	zap.L().Error(fmt.Sprintf("max retries reached for event: %v", events), zap.Error(MYSQL_WRITE_ERROR))
	return MYSQL_WRITE_ERROR
}

// 处理死信队列中的事件
func (lp *UserFollowProcessor) handleDeadLetters(ctx context.Context) {
	for {
		select {
		case events := <-lp.deadLetterQueueForMySQL:
			zap.L().Info(fmt.Sprintf("Handling dead letter event: %+v\n", events))
			err := lp.handleFollowMySQL(events)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Final attempt to process user follow in redis event failed: %v\n", err), zap.Error(err))
				// 这里可以记录到持久化存储，或发送告警通知
				// 在此场景下，需要将这些事件记录，并交给一个线程专门继续尝试将这些事件写入数据库
				zap.L().Warn("Need to add this events to a special thread for future process")
				return
			}
		case events := <-lp.deadLetterQueueForRedis:
			zap.L().Info(fmt.Sprintf("Handling dead letter event: %+v\n", events))
			err := lp.handleFollowRedis(events)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Final attempt to process user follow in mysql failed: %v\n", err), zap.Error(err))
				// 这里可以记录到持久化存储，或发送告警通知
				// 在此场景下，需要将这些事件记录，并交给一个线程专门继续尝试将这些事件写入数据库
				zap.L().Warn("Need to add this events to a special thread for future process")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// 用户关注数据库操作
func userFollowDatabaseOperation(event UserFollowEvent) error {
	oValue := mysql_repo.UserFollowRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("follower_id = ?", event.UserId).Where("following_id = ?", event.TargetUserId))
	if oValue == nil {
		follow := &models.Follow{FollowId: snowflake.GenID(), FollowerId: event.UserId, FollowingId: event.TargetUserId, Val: 1}
		err := mysql_repo.UserFollowRepository.Create(sqls.DB(), follow)
		if err != nil {
			return err
		}
	} else {
		oValue.Val = 1
		err := mysql_repo.UserFollowRepository.UpdateColumn(sqls.DB(), oValue.FollowId, "val", oValue.Val)
		if err != nil {
			return err
		}
	}

	return nil
}

// 取消关注数据库操作
func cancelUserFollowPostDatabaseOperation(event UserFollowEvent) error {
	oValue := mysql_repo.UserFollowRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("follower_id = ?", event.UserId).Where("following_id = ?", event.TargetUserId))
	if oValue == nil {
		// 说明没有关注过，没必要执行操作
		return nil
	} else {
		oValue.Val = 0
		err := mysql_repo.UserFollowRepository.UpdateColumn(sqls.DB(), oValue.FollowId, "val", oValue.Val)
		if err != nil {
			return err
		}
	}
	return nil
}

// 批量提交Redis修改
func batchUserFollowRedisOperation(events []UserFollowEvent) error {
	// 往Redis hash set 中写入数据
	ops := make([]models.FollowOperation, len(events))
	for i, event := range events {
		ops[i].UserId = event.UserId
		ops[i].TargetUserId = event.TargetUserId
		if event.Action == "follow" {
			ops[i].Action = 1
		}
	}
	err := redis_repo.ExecuteBatchFollowOperation(ops)
	if err != nil {
		return err
	}
	return nil
}

// 批量用户关注与取关数据库操作 (GORM 版本)
func batchUserFollowDatabaseOperation(events []UserFollowEvent) error {
	followerIds := make([]interface{}, len(events))
	followingIds := make([]interface{}, len(events))

	for i, event := range events {
		followerIds[i] = event.UserId
		followingIds[i] = event.TargetUserId
	}

	// 批量查询所有已存在的记录
	var existingFollows []models.Follow
	existingFollows = mysql_repo.UserFollowRepository.Find(sqls.DB(), sqls.NewCnd().In("follower_id", followerIds).In("following_id", followingIds))

	// 构建已存在记录的map
	existingFollowMap := make(map[string]*models.Follow)
	for _, follow := range existingFollows {
		key := fmt.Sprintf("%d_%d", follow.FollowerId, follow.FollowingId)
		existingFollowMap[key] = &follow
	}

	var newFollows []*models.Follow
	var updates []struct {
		FollowId int64
		Val      int8
	}

	for _, event := range events {
		var newVal int8
		if event.Action == "follow" {
			newVal = 1
		} else if event.Action == "none" {
			newVal = 0
		}
		key := fmt.Sprintf("%d_%d", event.UserId, event.TargetUserId)
		if oValue, exists := existingFollowMap[key]; exists {
			oValue.Val = newVal
			updates = append(updates, struct {
				FollowId int64
				Val      int8
			}{FollowId: oValue.FollowId, Val: oValue.Val})
		} else {
			newFollows = append(newFollows, &models.Follow{
				FollowId:    snowflake.GenID(),
				FollowerId:  event.UserId,
				FollowingId: event.TargetUserId,
				Val:         newVal,
			})
		}
	}
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		// 批量插入新记录
		if len(newFollows) > 0 {
			if err := tx.Create(&newFollows).Error; err != nil {
				return err
			}
		}

		// 批量更新已存在记录
		if len(updates) > 0 {
			query := "UPDATE t_follow SET val = CASE follow_id "
			ids := make([]interface{}, len(updates))
			params := make([]interface{}, 0, len(updates)*2+len(updates))

			// 构建 SQL 和 参数
			for i, update := range updates {
				query += "WHEN ? THEN ? "
				ids[i] = update.FollowId
				params = append(params, update.FollowId, update.Val)
			}
			query += "END WHERE follow_id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"

			// 将 ids 添加到参数中
			params = append(params, ids...)

			// 执行原生 SQL 批量更新
			if err := tx.Exec(query, params...).Error; err != nil {
				return err
			}
		}

		// 如果所有操作成功，提交事务
		return nil
	})
}
