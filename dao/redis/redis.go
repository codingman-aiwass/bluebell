package redis

import (
	"bluebell/settings"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

func Init(cfg *settings.RedisConfig) (err error) {
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d",
			cfg.Host, cfg.Port),
		Password: cfg.Password, // 密码
		DB:       cfg.Database, // 数据库
		PoolSize: cfg.PoolSize, // 连接池大小
	})
	_, err = rdb.Ping(ctx).Result()
	return
}

func CLose() {
	_ = rdb.Close()
}
