package redis

import (
	"bluebell/models"
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
	return err
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
func SetEmailVerificationInfo(ctx context.Context, info string) error {
	err := rdb.Set(ctx, getKey(KeyMailVerification+":"+info), true, EMAIL_VERFICATION_VALID_TIME).Err()
	return err
}

// 查看验证码是否存在且未过期
func GetEmailVerificationCode(ctx context.Context, info string) (bool, error) {
	key := getKey(KeyMailVerification + ":" + info)
	return rdb.Get(ctx, key).Bool()
}

// 删除邮箱绑定验证信息
func DeleteEmailVerificationInfo(ctx context.Context, info string) error {
	return rdb.Del(ctx, getKey(KeyMailVerification+":"+info)).Err()
}

// 删除邮箱验证码
func DeleteEmailVerificationCode(ctx context.Context, email string) error {
	return rdb.Del(ctx, getKey(KeyMailLoginCode+":"+email)).Err()
}

// 查看用户上次登录凭据是否依然存在
func GetUserLastLoginToken(ctx context.Context, userId int64) (bool, error) {
	key := getKey(KeyPrefix + ":" + KeyUserLastLoginToken + ":" + strconv.FormatInt(userId, 10))
	return rdb.Get(ctx, key).Bool()
}

// 存入用户登录凭据
func SetUserLastLoginToken(ctx context.Context, userId int64, duration time.Duration) error {
	err := rdb.Set(ctx, getKey(KeyMailVerification+":"+strconv.FormatInt(userId, 10)), true, duration).Err()
	return err
}

// 设置上次登录凭据的过期时间
func SetExpiredTime(ctx context.Context, userId int64, duration time.Duration) error {
	err := rdb.Expire(ctx, getKey(KeyPrefix+":"+KeyUserLastLoginToken+":"+strconv.FormatInt(userId, 10)), duration).Err()
	return err
}

// 查看传入的验证码是否在redis中
func CheckValidEmailVerificationCode(ctx context.Context, user *models.User) (string, error) {
	key := getKey(KeyMailLoginCode + ":" + user.Email)
	return rdb.Get(ctx, key).Result()
}

// 设置邮箱登录验证码
func SetEmailVerificationCode(ctx context.Context, email, code string) error {
	key := getKey(KeyMailLoginCode + ":" + email)
	return rdb.Set(ctx, key, code, EMAIL_LOGIN_CODE_VALID_TIME).Err()
}
