package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"bluebell/modules"
	"bluebell/pkg/snowflake"
	"errors"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"time"
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

const AccessTokenExpireDuration = time.Hour * 2
const RefreshTokenExpireDuration = time.Hour * 24 * 7

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
	return mysql.SaveUserEditableInfo(userId, info)

}
