package message_queue

import (
	"bluebell/dao/mysql_repo"
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

var (
	MaxWaitTime  = 30 * time.Second
	MaxBatchSize = 5
)

type PostClickProcessor struct {
	kafkaReader     *kafka.Reader
	messages        chan kafka.Message
	deadLetterQueue chan PostClickEvent // 用于存储失败的事件
	maxRetries      int                 // 最大重试次数

}

func NewPostClickProcessor(brokers []string, topic string, maxRetries int) *PostClickProcessor {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     "post_click_event_consumer_group",
		StartOffset: kafka.FirstOffset,
		Partition:   0,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})

	return &PostClickProcessor{
		kafkaReader:     reader,
		messages:        make(chan kafka.Message),
		deadLetterQueue: make(chan PostClickEvent, 100), // 设定一个缓冲区
		maxRetries:      maxRetries,
	}
}
func (lp *PostClickProcessor) Start(ctx context.Context) {
	go lp.consumeMessages(ctx)
	go lp.process(ctx)
	go lp.handleDeadLetters(ctx) // 处理死信队列

	// Wait for termination signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	lp.kafkaReader.Close()
}

func (lp *PostClickProcessor) consumeMessages(ctx context.Context) {
	for {
		msg, err := lp.kafkaReader.ReadMessage(ctx)
		if err != nil {
			zap.L().Info(fmt.Sprintf("Failed to read message:%v", err))
			continue
		}
		lp.messages <- msg // Send the message to the processing channel
	}
}

func (lp *PostClickProcessor) process(ctx context.Context) {
	//var messages []kafka.Message
	//// 启动定时器
	//var resetTimer func()
	//resetTimer = func() {
	//	time.AfterFunc(MaxWaitTime, func() {
	//		if len(messages) > 0 {
	//			zap.L().Info("时间到，提交消息")
	//
	//			commitMessages(lp.kafkaReader, messages)
	//			messages = nil
	//		}
	//		resetTimer() // 重置定时器
	//	})
	//}
	//
	//resetTimer() // 初始化定时器
	for {
		select {
		case msg := <-lp.messages:
			var event PostClickEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				zap.L().Info(fmt.Sprintf("Failed to unmarshal message:%v", err))
				continue
			}
			if err := lp.handle(event); err != nil {
				zap.L().Info(fmt.Sprintf("Failed to process like event: %v, moving to dead letter queue\n", err))
				lp.deadLetterQueue <- event // 添加到死信队列
			} else {
				// 处理成功，提交消息
				commitMessage(lp.kafkaReader, msg)
			}

		case <-ctx.Done():
			return
		}
	}
}

// 点击事件
func (lp *PostClickProcessor) handle(event PostClickEvent) error {
	var err error

	for i := 0; i <= lp.maxRetries; i++ {
		if postDeleted(event.PostId) {
			zap.L().Info(fmt.Sprintf("Post %d is deleted, cannot like, skipping...\n", event.PostId))
			return nil // 帖子已删除，直接放弃
		}

		err = lp.addPostClickNums(event.PostId)

		if err == nil {
			return nil // 成功处理
		}
		zap.L().Info(fmt.Sprintf("Error processing event, retrying... (%d/%d): %v\n", i+1, lp.maxRetries, err))
		time.Sleep(100 * time.Millisecond) // 等待后重试
	}
	return errors.New(fmt.Sprintf("max retries reached for event: %v", event))
}

// 处理死信队列中的事件
func (lp *PostClickProcessor) handleDeadLetters(ctx context.Context) {
	for {
		select {
		case event := <-lp.deadLetterQueue:
			zap.L().Info(fmt.Sprintf("Handling dead letter event: %+v\n", event))
			// 对于死信事件的策略：再尝试一次，若失败则记录日志
			if postDeleted(event.PostId) {
				zap.L().Info(fmt.Sprintf("Post %d is deleted, skipping retry...\n", event.PostId))
				continue // 放弃重试
			}
			err := lp.handle(event)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Final attempt to process like event failed: %v\n", err), zap.Error(err))
				// 这里可以记录到持久化存储，或发送告警通知
			}
		case <-ctx.Done():
			return
		}
	}
}

func (lp *PostClickProcessor) addPostClickNums(postId int64) (err error) {
	err = mysql_repo.PostRepository.IncreaseClickNum(sqls.DB(), postId)
	return err
}
