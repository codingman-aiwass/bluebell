package message_queue

import (
	"bluebell/dao/mysql_repo"
	"bluebell/pkg/sqls"
	"bluebell/settings"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type PostLikeEvent struct {
	Action    string `json:"action"`
	UserId    int64  `json:"user_id"`
	PostId    int64  `json:"post_id"`
	Timestamp string `json:"timestamp"`
}

type PostClickEvent struct {
	UserId int64
	PostId int64
}

type PostCollectionEvent struct {
	Action    string `json:"action"`
	UserId    int64  `json:"user_id"`
	PostId    int64  `json:"post_id"`
	Timestamp string `json:"timestamp"`
}

// 初始化需要的消费者和生产者，以及对应的topic
var (
	LikeTopic              = "post-like-events"
	LikeTopicMaxRetries    = 1
	DislikeTopic           = "post-dislike-events"
	DislikeTopicMaxRetries = 1
	PostClickTopic         = "post-click-events"
	PostClickMaxRetries    = 1
	ctx                    = context.Background()
)

func SendPostLikeEvent(ctx context.Context, topic string, message PostLikeEvent) (err error) {
	writer := kafka.Writer{
		Addr:                   kafka.TCP(settings.GlobalSettings.MQCfg.Brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           1 * time.Second,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()
	// try to send to mq for 3 times, if error, break
	send_msg, _ := json.Marshal(message)
	for i := 0; i < 3; i++ {
		if err = writer.WriteMessages(
			ctx, kafka.Message{Key: []byte(strconv.FormatInt(message.UserId, 10)), Value: send_msg}); err != nil {
			zap.L().Info("write kafka error,try...", zap.Error(err))
		} else {
			zap.L().Info(fmt.Sprintf("send like event msg to mq successfully,action = %s,user id = %d,post id = %d",
				message.Action, message.UserId, message.PostId))
			break
		}
	}
	return err
}

func SendPostClickEvent(ctx context.Context, message PostClickEvent) (err error) {
	writer := kafka.Writer{
		Addr:                   kafka.TCP(settings.GlobalSettings.MQCfg.Brokers...),
		Topic:                  PostClickTopic,
		Balancer:               &kafka.Hash{},
		WriteTimeout:           1 * time.Second,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}
	defer writer.Close()
	// try to send to mq for 2 times, if error, break
	send_msg, _ := json.Marshal(message)
	for i := 0; i < 2; i++ {
		zap.L().Info("UserId type and value", zap.Int64("UserId", message.UserId))

		if err = writer.WriteMessages(
			ctx, kafka.Message{Key: []byte(strconv.FormatInt(message.UserId, 10)), Value: send_msg}); err != nil {
			zap.L().Info("write kafka error,try again...", zap.Error(err))
		} else {
			zap.L().Info(fmt.Sprintf("send event msg to mq successfully,action = add click num,user id = %d,post id = %d",
				message.UserId, message.PostId))
			break
		}
	}
	return err
}

func InitMQ(cfg *settings.MessageQueueConfig) {
	// 需要启动多个监听消息队列的消费者
	likeProcessor := NewLikeProcessor(cfg.Brokers, LikeTopic, LikeTopicMaxRetries)
	go likeProcessor.Start(ctx)
	disLikeProcessor := NewDisLikeProcessor(cfg.Brokers, DislikeTopic, DislikeTopicMaxRetries)
	go disLikeProcessor.Start(ctx)
	postClickProcessor := NewPostClickProcessor(cfg.Brokers, PostClickTopic, PostClickMaxRetries)
	go postClickProcessor.Start(ctx)
}

// 帖子是否已被删除
func postDeleted(postID int64) bool {
	return mysql_repo.PostRepository.Get(sqls.DB(), postID) == nil
}

// 提交消息的 offset
func commitMessages(reader *kafka.Reader, messages []kafka.Message) {
	// 提交缓存中的所有消息
	err := reader.CommitMessages(context.Background(), messages...)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to commit messages: %v", err))
	}
	zap.L().Info(fmt.Sprintf("成功提交 %d 条消息\n", len(messages)))
}

func commitMessage(reader *kafka.Reader, message kafka.Message) {
	// 提交缓存中的一条消息
	err := reader.CommitMessages(context.Background(), message)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to commit messages: %v", err))
	}
	zap.L().Info("成功提交 1 条消息\n")
}
