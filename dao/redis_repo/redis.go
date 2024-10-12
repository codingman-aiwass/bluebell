package redis_repo

import (
	"bluebell/settings"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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

func Exists(ctx context.Context, key string) (bool, error) {
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func AddToZset(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return rdb.ZAdd(ctx, getKey(key), members...)
}

func CreateBloomFilter(ctx context.Context, filterName string, errorRate float64, capacity int64) (err error) {
	_, err = rdb.Do(ctx, "BF.RESERVE", filterName, errorRate, capacity).Result()
	if err != nil {
		zap.L().Error("Could not create bloom filter", zap.Error(err))
		return err
	}
	return nil
}

func CheckInBloomFilter(filterName string, item string) (exists bool, err error) {
	exist, err := rdb.Do(ctx, "BF.EXISTS", filterName, item).Result()
	if err != nil {
		zap.L().Error("Fail to check bloom filter", zap.Error(err))
		return false, err
	}
	return exist.(int64) == 1, nil

}

func AddToBloomFilter(filterName string, item string) (err error) {
	added, err := rdb.Do(ctx, "BF.ADD", filterName, item).Result()
	if err != nil {
		zap.L().Error("Could not add to bloom filter", zap.Error(err))
		return err
	}
	if added == 1 {
		zap.L().Info(fmt.Sprintf("Item %s added to the bloom filter", item))
	} else {
		zap.L().Info(fmt.Sprintf("Item %s already exists in the bloom filter", item))
	}
	return nil
}
