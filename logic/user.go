package logic

import (
	"bluebell/cache"
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/emails"
	"bluebell/pkg/encrypt"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"bluebell/pkg/validation"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"time"
)

var ctx = context.Background()
var ERROR_EMAIL_VERIFIED_BEFORE = errors.New("email verified before")
var ERROR_DUPLICATED_EMAIL = errors.New("this email has been occupied")
var ERROR_DUPLICATED_USERNAME = errors.New("this username has been occupied")
var ERROR_EMPTY_USERNAME = errors.New("username can not be empty")
var ERROR_WRONG_PASSWORD = errors.New("username or password wrong")
var ERROR_WRONG_USER = errors.New("no such user exists")

const (
	EMAIL_NOT_VERIFIED = false
	EMAIL_VERIFIED     = true
)
const (
	AccessTokenExpireDuration  = time.Hour * 200000
	RefreshTokenExpireDuration = time.Hour * 24 * 7
	//LoginTokenExpireDuration    = time.Hour * 24 * 30 * 12
	EMAIL_VERIFICATION_CODE_LEN = 6
)

// 处理和用户相关的具体业务逻辑

func SignUp(user *models.ParamUserSignUp) (err error) {
	// 获取到用户传进来的结构体
	// 首先需要判断该结构体中包含的用户名是否存在，如果存在则退出
	if u := mysql_repo.UserRepository.GetByUsername(sqls.DB(), user.Username); u != nil {
		zap.L().Error("user has already exists in db", zap.Error(ERROR_DUPLICATED_USERNAME))
		return ERROR_DUPLICATED_USERNAME
	}
	// 需要检查用户名/密码是否合法
	if len(user.Username) == 0 {
		zap.L().Error("username can not be empty", zap.Error(ERROR_EMPTY_USERNAME))
		return ERROR_EMPTY_USERNAME
	}
	if err = validation.IsPassword(user.Password); err != nil {
		zap.L().Error("password error", zap.Error(err))
		return err
	}

	if len(user.Email) > 0 {
		if err = validation.IsEmail(user.Email); err != nil {
			zap.L().Error("invalid emails...", zap.Error(err))
			return err
		}
		// 判断数据库中是否已经有该email
		u := mysql_repo.UserRepository.GetByEmail(sqls.DB(), user.Email)
		if u != nil {
			zap.L().Error("email has existed...", zap.Error(ERROR_DUPLICATED_EMAIL))
			return ERROR_DUPLICATED_EMAIL
		}
	}

	// 然后需要构建一个新的user结构体，存入数据库
	u := &models.User{
		UserId:   snowflake.GenID(),
		Username: user.Username,
		Password: encrypt.Encrypt(user.Password),
		Email:    user.Email,
	}
	// 存入数据库
	if err = mysql_repo.UserRepository.Create(sqls.DB(), u); err != nil {
		zap.L().Error("save user failed", zap.Error(err))
		return
	}
	return nil
}

func SignInWithPassword(user *models.User) (err error) {
	// 可能传进来的是email，或者是username
	var u *models.User = nil
	// 先检查是否为email
	if err = validation.IsEmail(user.Username); err == nil {
		u = mysql_repo.UserRepository.GetByEmail(sqls.DB(), user.Username)
	}
	// 按照email找不到的话再根据用户名
	if u == nil {
		if err = validation.IsUsername(user.Username); err == nil {
			u = mysql_repo.UserRepository.GetByUsername(sqls.DB(), user.Username)
		}
	}

	if !validation.CheckPassword(user.Password, u.Password) {
		zap.L().Error("check user failed", zap.Error(err))
		return ERROR_WRONG_PASSWORD
	}
	user.UserId = u.UserId
	user.Username = u.Username
	user.Email = u.Email
	return nil
}
func SignInWithEmailVerificationCode(user *models.User) (err error) {
	// 需要去redis中查询验证码是否存在
	code, err := redis_repo.CheckValidEmailVerificationCode(ctx, user)
	// check if exists and expired
	if err != nil || user.Password != code {
		zap.L().Error("check emails verification code failed", zap.Error(err))
		return redis_repo.ERROR_EMAIL_INVALID_VERIFICATION_CODE
	}
	// 将这个验证码删除
	err = redis_repo.DeleteEmailVerificationCode(ctx, user.Email)
	if err != nil {
		zap.L().Warn("delete emails verification code failed", zap.Error(err))
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

func GenRefreshToken() (string, error) {
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
	err = redis_repo.SaveUserId2AccessToken(newAccessToken, claim.UserId)
	if err != nil {
		zap.L().Error("save new access token to redis_repo failed", zap.Error(err))
		return newAccessToken, err
	}

	return newAccessToken, err
}

func CheckMoreThanOneUser(userId int64, accessToken string) (bool, error) {
	err := redis_repo.CheckUserId2AccessToken(userId, accessToken)
	if err != nil {
		return true, err
	}
	return false, nil
}

func GetUsernameById(userId int64) (username string, err error) {
	u := cache.UserCache.Get(userId)
	if u == nil {
		return "", ERROR_WRONG_USER
	}
	return u.Username, nil
}

func GetEmailById(userId int64) (email string, err error) {
	u := cache.UserCache.Get(userId)
	if u == nil {
		return "", ERROR_WRONG_USER
	}
	return u.Email, nil
}

func EditUserInfo(info *models.ParamUserEditInfo, userId int64) (err error) {
	// 先查看一下原本的值，如果用户传来的字段有的部分没有填,或者没有变动，就不做改动
	oInfo := cache.UserCache.Get(userId)
	if oInfo == nil {
		zap.L().Error("get user editable info by id error", zap.Error(ERROR_WRONG_USER))
	}
	// 判断用户传来的值是否都为默认值或者原值
	if (info.Email == oInfo.Email && info.Gender == oInfo.Gender) ||
		(info.Email == "" && info.Gender == 0) {
		return nil
	}
	// 判断用户传来的值是否和数据库中已有的email重复
	user := mysql_repo.UserRepository.GetByEmail(sqls.DB(), info.Email)
	if user != nil {
		return ERROR_DUPLICATED_EMAIL
	}
	columns := map[string]interface{}{
		"gender":   info.Gender,
		"email":    info.Email,
		"verified": 0,
	}

	err = mysql_repo.UserRepository.Updates(sqls.DB(), userId, columns)
	cache.UserCache.Invalidate(userId)
	return err
}

// 生成code，存入redis，发送验证邮件
func SendEmailVerification(email1 string) error {
	// 首先需要生成由email|code组成的经过base64编码过的字符串
	info := emails.GenEmailVerificationInfo(email1)
	// 将信息设置一个有效期然后存入redis数据库
	err := redis_repo.SetEmailVerificationInfo(ctx, info)
	if err != nil {
		zap.L().Error("set emails verification info in logic failed", zap.Error(err))
		return err
	}
	// 生成email信息
	emailData := GenEmailVerificationData(email1, info)
	// 准备发送邮件
	err = SendVerificationInfoEmail(email1, emailData)
	if err != nil {
		zap.L().Error("send emails failed", zap.Error(err))
		return redis_repo.ERROR_EMAIL_SEND_FAILED
	}

	return nil

}

func VerifyEmail(info string) error {
	// 从redis中查看info是否存在以及是否过期
	exists, err := redis_repo.GetEmailVerificationCode(ctx, info)
	// check if exists and expired
	if err != nil || !exists {
		zap.L().Warn("get emails verification code from redis_repo failed", zap.Error(err))
		return redis_repo.ERROR_EMAIL_INFO_NOT_EXISTS
	}

	// 确认存在以后，删掉这条记录
	err = redis_repo.DeleteEmailVerificationInfo(ctx, info)
	if err != nil {
		zap.L().Error("delete emails verification info from redis_repo failed", zap.Error(err))
	}

	// 提取出email信息，通过该信息找到用户，查看是否已经验证过，没有验证过就修改mysql状态
	email, _, err := emails.ParseEmailVerificationInfo(info)
	if err != nil {
		zap.L().Error("parse emails verification info from redis_repo failed", zap.Error(err))
		return err
	}
	user := mysql_repo.UserRepository.GetByEmail(sqls.DB(), email)
	if user == nil {
		zap.L().Error("get user by emails failed", zap.Error(ERROR_WRONG_USER))
		return ERROR_WRONG_USER
	}
	if user.Verified == EMAIL_VERIFIED {
		zap.L().Warn("emails has been verified...", zap.String("emails", email))
		return ERROR_EMAIL_VERIFIED_BEFORE
	}
	err = mysql_repo.UserRepository.UpdateColumn(sqls.DB(), user.UserId, "verified", EMAIL_VERIFIED)
	return err
}

//func VerifyLoginToken(userId int64, duration time.Duration) (err error) {
//	// 从redis中查看Login是否存在以及是否过期
//	exists, err := redis_repo.GetUserLastLoginToken(ctx, userId)
//	// check if exists and expired
//	if err != nil || !exists {
//		zap.L().Warn("last login is too long ago", zap.Error(err))
//		return redis_repo.ERROR_GAP_TOO_LONG
//	}
//
//	// 确认存在以后，更新记录的过期时间
//	err = redis_repo.SetExpiredTime(ctx, userId, duration)
//	if err != nil {
//		zap.L().Error("reset Login Token in redis_repo failed", zap.Error(err))
//	}
//	return err
//}

func SignInPostProcess(u *models.User) (res gin.H, err error) {
	if u.UserId == 0 {
		user := mysql_repo.UserRepository.GetByEmail(sqls.DB(), u.Email)
		if user == nil {
			zap.L().Error("get user by emails failed", zap.Error(ERROR_WRONG_USER))
			return res, ERROR_WRONG_USER
		}
		u.UserId = user.UserId
	}

	// 生成有效的token并返回给客户端
	access_token, err := GenAccessToken(u)
	if err != nil {
		zap.L().Error("user sign in jwt gen access token error in controller.SignInWithPassword()...", zap.Error(err))
	}
	refresh_token, err := GenRefreshToken()
	if err != nil {
		zap.L().Error("user sign in jwt gen refresh token error in controller.SignInWithPassword()...", zap.Error(err))
		return
	}
	// 将access token存入redis数据库，实现每次只能有一个用户访问特定资源的目的
	err = redis_repo.SaveUserId2AccessToken(access_token, u.UserId)
	if err != nil {
		zap.L().Error("user sign in jwt save access token to redis_repo error in controller.SignInWithPassword()...", zap.Error(err))
		return
	}
	res = gin.H{
		"access_token":  access_token,
		"refresh_token": refresh_token,
	}

	// 存入用户登录凭据，在一定时间内，下次就不用再用验证码登录
	//err = redis_repo.SetUserLastLoginToken(ctx, u.UserId, LoginTokenExpireDuration)
	//if err != nil {
	//	zap.L().Error("set user last login token in redis_repo failed", zap.Error(err))
	//	return nil, err
	//}

	return res, nil
}
func GenerateCode(email1 string) (string, error) {
	code := emails.GenCode(EMAIL_VERIFICATION_CODE_LEN)
	// 将code存入redis
	err := redis_repo.SetEmailVerificationCode(ctx, email1, code)
	if err != nil {
		return "", err
	}
	// 发送邮件

	return code, nil

}
