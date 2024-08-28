package redis

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// 将userId-access token存入数据库

func SaveUserId2AccessToken(accessToken string, userId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := rdb.HSet(ctx, fmt.Sprintf("%s%s", KeyPrefix, KeyUserTokenHash), strconv.FormatInt(userId, 10), accessToken).Err()
	if err != nil {
		zap.L().Error("save user id to access token failed", zap.Error(err))
		return err
	}
	return nil
}

func CheckUserId2AccessToken(userId int64, access_token string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	accessToken, err := rdb.HGet(ctx, fmt.Sprintf("%s%s", KeyPrefix, KeyUserTokenHash), strconv.FormatInt(userId, 10)).Result()
	if err != nil {
		zap.L().Error("get access token from redis userid2access_token failed", zap.Error(err))
		return
	}
	if accessToken != access_token {
		zap.L().Error("current access token is different from one in redis...", zap.Error(ERROR_MORE_THAN_ONE_USER))
		return
	}
	return nil
}

// 将邮箱验证码存入redis
func SetEmailVerificationInfo(ctx context.Context, info string, duration time.Duration) error {
	err := rdb.Set(ctx, getKey(KeyMailVerification+":"+info), true, duration).Err()
	if err != nil {
		zap.L().Error("save email verification info to redis failed", zap.Error(err))
		return err
	}
	return nil
}

// 查看验证码是否存在且未过期
func GetEmailVerificationCode(ctx context.Context, info string) (bool, error) {
	key := getKey(KeyMailVerification + ":" + info)
	return rdb.Get(ctx, key).Bool()
}

// 删除验证码
func DeleteEmailVerificationInfo(ctx context.Context, info string) error {
	return rdb.Del(ctx, getKey(KeyMailVerification+":"+info)).Err()
}
