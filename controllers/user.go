package controllers

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/logic"
	"bluebell/models"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"strings"
)

// 处理和用户相关的请求

// SignUp 处理账户注册
// @Summary 实现用户注册功能
// @Description 接受用户输入的用户名，密码，确认密码，邮箱（可选）
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamUserSignUp true "注册信息"
// @Success 200 {object} _ResponseUserSignUp
// @Router /api/v1/signup [post]
func SignUp(context *gin.Context) {
	// 1. 参数校验
	user := new(models.ParamUserSignUp)
	if err := context.ShouldBindJSON(user); err != nil {
		ResponseError(context, CODE_PARAM_ERROR)
		zap.L().Error("user sign up parameter bind error...", zap.Error(err))
		return
	}
	// 2.调用业务逻辑层
	if err := logic.SignUp(user); err != nil {
		if errors.Is(err, mysql.ERROR_USER_EXISTS) {
			ResponseError(context, CODE_USER_EXISTS)
		} else {
			ResponseError(context, CODE_INTERNAL_ERROR)
		}
		zap.L().Error("user sign up error in logic.SignUp()...", zap.Error(err))
		return
	}
	// 3. 返回
	ResponseSuccess(context, nil)
}

// SignIn 处理账户登录
// @Summary 实现用户登录功能
// @Description 接受用户输入的用户名，密码，返回refresh-token 和 access-token
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamUserSignIn true "登录必备信息"
// @Success 200 {object} _ResponseUserSignIn
// @Router /api/v1/login [post]
func SignIn(context *gin.Context) {
	// 1. 参数校验
	user := new(models.ParamUserSignIn)
	if err := context.ShouldBindJSON(user); err != nil {
		ResponseError(context, CODE_PARAM_ERROR)
		zap.L().Error("user sign in parameter bind error...", zap.Error(err))
		return
	}
	// 2.调用业务逻辑层
	u := new(models.User)
	u.Username = user.Username
	u.Password = user.Password
	if err := logic.SignIn(u); err != nil {
		ResponseError(context, CODE_PASSWORD_ERROR)
		zap.L().Error("user sign in parameter in controller.SignIn()...", zap.Error(err))
		return
	}
	// 3. 生成有效的token并返回给客户端
	access_token, err := logic.GenAccessToken(u)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		zap.L().Error("user sign in jwt gen access token error in controller.SignIn()...", zap.Error(err))
	}
	refresh_token, err := logic.GenRefreshToken(u)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		zap.L().Error("user sign in jwt gen refresh token error in controller.SignIn()...", zap.Error(err))
	}
	// 4. 将access token存入redis数据库，实现每次只能有一个用户访问特定资源的目的
	err = redis.SaveUserId2AccessToken(access_token, u.UserId)
	if err != nil {
		zap.L().Error("user sign in jwt save access token to redis error in controller.SignIn()...", zap.Error(err))
		ResponseError(context, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(context, gin.H{
		"access_token":  access_token,
		"refresh_token": refresh_token,
	})
}

// EditUserInfo 更新用户信息
// @Summary 更新用户信息
// @Description 用户可更新用户名/性别/email
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamUserEditInfo true "用户修改信息"
// @Param Authorization header string false "Bearer 用户令牌"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseUserEditInfo
// @Router /api/v1/edit-info [post]
func EditUserInfo(c *gin.Context) {
	info := new(models.ParamUserEditInfo)
	err := c.BindJSON(&info)
	if err != nil {
		zap.L().Error("bind user editable info error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}
	// 调用逻辑层去修改数据库
	if err = logic.EditUserInfo(info, c.GetInt64(ContextUserIdKey)); err != nil {
		zap.L().Error("edit user info error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
	}
	ResponseSuccess(c, nil)
}

// RefreshAccessToken 刷新AccessToken的接口
// @Summary 实现刷新token功能
// @Description 判断当前refresh-token是否过期，没有的话返回新的access-token（access token过期的话）
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string false "用户的refresh-token和access-token"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseUserSignIn
// @Router /api/v1/refresh-access-token [get]
func RefreshAccessToken(c *gin.Context) {
	// 1.检验参数，判断是否携带access token 和 refresh token
	// 假设token是存放在Head的Authorization字段中，access token和refresh token用｜隔开
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		zap.L().Error("no auth header")
		ResponseError(c, CODE_NOT_LOGIN)
		return
	}
	// 分割token
	tokens := strings.Split(authHeader, "|")
	if len(tokens) != 2 {
		zap.L().Error("invalid auth header")
		ResponseError(c, CODE_INVALID_TOKEN)
	}
	accessToken, err := logic.RefreshToken(tokens[0], tokens[1])
	if err != nil {
		if errors.Is(err, err.(jwt.ValidationError)) {
			zap.L().Error("invalid refresh token", zap.Error(err))
			ResponseError(c, CODE_INVALID_TOKEN)
		} else {
			zap.L().Error("refresh token error", zap.Error(err))
			ResponseError(c, CODE_INTERNAL_ERROR)
		}
	}
	ResponseSuccess(c, accessToken)
}
