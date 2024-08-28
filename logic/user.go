package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"bluebell/modules"
	"bluebell/pkg/snowflake"
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"time"
)

var ctx = context.Background()
var ERROR_EMAIL_VERIFIED_BEFORE = errors.New("email verified before")
var ERROR_DUPLICATED_EMAIL = errors.New("this email has been occupied")

const (
	EMAIL_NOT_VERIFIED = iota
	EMAIL_VERIFIED
)
const (
	AccessTokenExpireDuration  = time.Hour * 2
	RefreshTokenExpireDuration = time.Hour * 24 * 7
)

// 处理和用户相关的具体业务逻辑

func SignUp(user *models.ParamUserSignUp) (err error) {
	// 获取到用户传进来的结构体
	// 首先需要判断该结构体中包含的用户名是否存在，如果存在则退出
	if err = mysql.CheckUserExist(user.Username); err != nil {
		zap.L().Error("user has already exists in db", zap.Error(err))
		return
	}

	// 然后需要构建一个新的user结构体，存入数据库
	u := &models.User{
		UserId:   snowflake.GenID(),
		Username: user.Username,
		Password: modules.Encrypt(user.Password),
	}
	// 存入数据库
	if err = mysql.SaveUser(u); err != nil {
		zap.L().Error("save user failed", zap.Error(err))
		return
	}
	return nil
}

func SignIn(user *models.User) (err error) {
	if err = mysql.CheckValidUser(user); err != nil {
		zap.L().Error("check user failed", zap.Error(err))
		return err
	}
	return nil
}

type MyClaims struct {
	UserId   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

var MySecret = []byte("Hello Bluebell!")
var INVALID_TOKEN = errors.New("invalid token")

func GenAccessToken(user *models.User) (string, error) {
	c := MyClaims{
		user.UserId,
		user.Username,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(AccessTokenExpireDuration).Unix(),
			Issuer:    "bluebell-project",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(MySecret)
}

func GenRefreshToken(user *models.User) (string, error) {
	c := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(RefreshTokenExpireDuration).Unix(),
		Issuer:    "bluebell-project",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(MySecret)
}

func ParseToken(tokenString string) (*MyClaims, error) {
	c := new(MyClaims)
	token, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
		return MySecret, nil
	})
	if err != nil {
		zap.L().Error("parse token failed", zap.Error(err))
		return nil, err
	}
	if !token.Valid {
		zap.L().Error("invalid token")
		return nil, INVALID_TOKEN
	}
	return c, nil
}

func RefreshToken(accessToken string, refreshToken string) (newAccessToken string, err error) {
	// 如果refresh-token无效，直接返回错误
	_, err = jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return MySecret, nil
	})
	if err != nil {
		return "", err
	}
	// 从旧access token解析claim数据
	claim := new(MyClaims)
	_, err = jwt.ParseWithClaims(accessToken, claim, func(token *jwt.Token) (interface{}, error) {
		return MySecret, nil
	})
	// 如果access-token有效，不做处理，此时返回一个空
	if err != nil {
		v, _ := err.(*jwt.ValidationError)

		// 当错误类型是过期错误，并且refresh token没有过期，创建一个新的access token
		if v.Errors == jwt.ValidationErrorExpired {
			u := models.User{UserId: claim.UserId, Username: claim.Username}
			if newAccessToken, err = GenAccessToken(&u); err != nil {
				zap.L().Error("gen access token failed", zap.Error(err))
				return "", err
			}
		}
	}
	// 将有效的access token记录到redis数据库中
	err = redis.SaveUserId2AccessToken(newAccessToken, claim.UserId)
	if err != nil {
		zap.L().Error("save new access token to redis failed", zap.Error(err))
		return newAccessToken, err
	}

	return newAccessToken, err
}

func CheckMoreThanOneUser(userId int64, accessToken string) (bool, error) {
	err := redis.CheckUserId2AccessToken(userId, accessToken)
	if err != nil {
		return true, err
	}
	return false, nil
}

func GetUsernameById(userId int64) (username string, err error) {
	return mysql.GetUsernameById(userId)
}

func EditUserInfo(info *models.ParamUserEditInfo, userId int64) (err error) {
	// 先查看一下原本的值，如果用户传来的字段有的部分没有填,或者没有变动，就不做改动
	oInfo, err := mysql.GetUserEditableInfoById(userId)
	if err != nil {
		zap.L().Error("get user editable info by id error", zap.Error(err))
	}
	// 判断用户传来的值是否都为默认值或者原值
	if (info.Email == oInfo.Email && info.Gender == oInfo.Gender) ||
		(info.Email == "" && info.Gender == 0) {
		return nil
	}
	// 判断用户传来的值是否和数据库中已有的email重复
	user, _ := mysql.GetUserByEmail(info.Email)
	if user != nil {
		return ERROR_DUPLICATED_EMAIL
	}
	return mysql.SaveUserEditableInfo(userId, info)

}

// 生成code，存入redis，发送验证邮件
func SendEmailVerification(email string) error {
	// 首先需要生成由email|code组成的经过base64编码过的字符串
	info := modules.GenEmailVerificationInfo(email)
	// 将信息设置一个有效期然后存入redis数据库
	err := redis.SetEmailVerificationInfo(ctx, info, time.Minute*15)
	if err != nil {
		zap.L().Error("set email verification info in logic failed", zap.Error(err))
		return err
	}
	// 生成email信息
	emailData := GenEmailData(email, info)
	// 准备发送邮件
	err = SendEmail(email, emailData)
	if err != nil {
		zap.L().Error("send email failed", zap.Error(err))
		return redis.ERROR_EMAIL_SEND_FAILED
	}

	return nil

}

func VerifyEmail(info string) error {
	// 从redis中查看info是否存在以及是否过期
	exists, err := redis.GetEmailVerificationCode(ctx, info)
	// check if exists and expired
	if err != nil || !exists {
		zap.L().Warn("get email verification code from redis failed", zap.Error(err))
		return redis.ERROR_EMAIL_INFO_NOT_EXISTS
	}

	// 确认存在以后，删掉这条记录
	err = redis.DeleteEmailVerificationInfo(ctx, info)
	if err != nil {
		zap.L().Error("delete email verification info from redis failed", zap.Error(err))
	}

	// 提取出email信息，通过该信息找到用户，查看是否已经验证过，没有验证过就修改mysql状态
	email, _, err := modules.ParseEmailVerificationInfo(info)
	if err != nil {
		zap.L().Error("parse email verification info from redis failed", zap.Error(err))
		return err
	}
	user, err := mysql.GetUserByEmail(email)
	if err != nil {
		zap.L().Error("get user by email failed", zap.Error(err))
		return err
	}
	if user.Verified == EMAIL_VERIFIED {
		zap.L().Warn("email has been verified...", zap.String("email", email))
		return ERROR_EMAIL_VERIFIED_BEFORE
	}
	err = mysql.UpdateUserFieldByEmail(email, "verified", EMAIL_VERIFIED)
	return err
}
