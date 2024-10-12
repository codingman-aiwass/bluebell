package message_queue

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type CollectPostProcessor struct {
	kafkaReader     *kafka.Reader
	messages        chan kafka.Message
	deadLetterQueue chan PostCollectionEvent // 用于存储失败的事件
	maxRetries      int                      // 最大重试次数
}

func NewCollectPostProcessor(brokers []string, topic string, maxRetries int) *CollectPostProcessor {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     "post_collection_event_consumer_group",
		StartOffset: kafka.FirstOffset,
		Partition:   0,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})

	return &CollectPostProcessor{
		kafkaReader:     reader,
		messages:        make(chan kafka.Message),
		deadLetterQueue: make(chan PostCollectionEvent, 100), // 设定一个缓冲区
		maxRetries:      maxRetries,
	}
}
func (lp *CollectPostProcessor) Start(ctx context.Context) {
	go lp.consumeMessages(ctx)
	go lp.processCollectPost(ctx)
	go lp.handleDeadLetters(ctx) // 处理死信队列

	// Wait for termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	lp.kafkaReader.Close()
}

func (lp *CollectPostProcessor) consumeMessages(ctx context.Context) {
	for {
		msg, err := lp.kafkaReader.ReadMessage(ctx)
		if err != nil {
			zap.L().Info(fmt.Sprintf("Failed to read message:%v", err))
			continue
		}
		lp.messages <- msg // Send the message to the processing channel
	}
}

func (lp *CollectPostProcessor) processCollectPost(ctx context.Context) {
	for {
		select {
		case msg := <-lp.messages:
			var collectionEvent PostCollectionEvent
			if err := json.Unmarshal(msg.Value, &collectionEvent); err != nil {
				zap.L().Info(fmt.Sprintf("Failed to unmarshal message:%v", err))
				continue
			}
			if err := lp.handleCollectPost(collectionEvent); err != nil {
				zap.L().Info(fmt.Sprintf("Failed to process like event: %v, moving to dead letter queue\n", err))
				lp.deadLetterQueue <- collectionEvent // 添加到死信队列
			} else {
				commitMessage(lp.kafkaReader, msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

// 处理收藏事件，带有不同的处理策略
func (lp *CollectPostProcessor) handleCollectPost(event PostCollectionEvent) error {
	var err error
	for i := 0; i <= lp.maxRetries; i++ {
		if postDeleted(event.PostId) {
			zap.L().Info(fmt.Sprintf("Post %d is deleted, cannot like, skipping...\n", event.PostId))
			return nil // 帖子已删除，直接放弃
		}
		switch event.Action {
		case "collect":
			zap.L().Info(fmt.Sprintf("User %d liked post %d at %s\n", event.UserId, event.PostId, event.Timestamp))
			// 处理收藏逻辑
			err = collectPostDatabaseOperation(event) // 执行点赞操作
		case "none":
			zap.L().Info(fmt.Sprintf("User %d cancel liked post %d at %s\n", event.UserId, event.PostId, event.Timestamp))
			// 处理取消收藏逻辑
			err = cancelCollectPostDatabaseOperation(event)
		default:
			return errors.New(fmt.Sprintf("Unknown like event,%s\n", event.Action))
		}

		if err == nil {
			return nil // 成功处理
		}
		zap.L().Info(fmt.Sprintf("Error processing event, retrying... (%d/%d): %v\n", i+1, lp.maxRetries, err))
		time.Sleep(100 * time.Millisecond) // 等待后重试
	}
	return errors.New(fmt.Sprintf("max retries reached for event: %v", event))
}

// 处理死信队列中的事件
func (lp *CollectPostProcessor) handleDeadLetters(ctx context.Context) {
	for {
		select {
		case event := <-lp.deadLetterQueue:
			zap.L().Info(fmt.Sprintf("Handling dead letter event: %+v\n", event))
			// 对于死信事件的策略：再尝试一次，若失败则记录日志
			if postDeleted(event.PostId) {
				zap.L().Info(fmt.Sprintf("Post %d is deleted, skipping retry...\n", event.PostId))
				continue // 放弃重试
			}
			err := lp.handleCollectPost(event)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Final attempt to process like event failed: %v\n", err), zap.Error(err))
				// 这里可以记录到持久化存储，或发送告警通知
			}
		case <-ctx.Done():
			return
		}
	}
}

// 收藏数据库操作
func collectPostDatabaseOperation(event PostCollectionEvent) error {
	oValue := mysql_repo.LikeRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("user_id = ?", event.UserId).Where("post_id = ?", event.PostId))
	if oValue == nil {
		like := &models.Like{LikeId: snowflake.GenID(), UserId: event.UserId, PostId: event.PostId, Val: 1}
		err := mysql_repo.LikeRepository.Create(sqls.DB(), like)
		if err != nil {
			return err
		}
	} else {
		oValue.Val = 1
		err := mysql_repo.LikeRepository.UpdateColumn(sqls.DB(), oValue.LikeId, "val", oValue.Val)
		if err != nil {
			return err
		}
	}

	return nil
}

// 取消收藏数据库操作
func cancelCollectPostDatabaseOperation(event PostCollectionEvent) error {
	oValue := mysql_repo.LikeRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("user_id = ?", event.UserId).Where("post_id = ?", event.PostId))
	if oValue == nil {
		// 说明没有收藏过，没必要执行操作
		return nil
	} else {
		// 需要将like记录里的val设置为0
		oValue.Val = 0
		err := mysql_repo.LikeRepository.UpdateColumn(sqls.DB(), oValue.LikeId, "val", oValue.Val)
		if err != nil {
			return err
		}
	}
	return nil
}
