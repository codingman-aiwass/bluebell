package controllers

import (
	"bluebell/dao/mysql_repo"
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
		if errors.Is(err, mysql_repo.ERROR_USER_EXISTS) {
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
	u.Email = user.Email
	if err := logic.SignInWithPassword(u); err != nil {
		ResponseError(context, CODE_PASSWORD_ERROR)
		zap.L().Error("user sign in parameter in controller.SignInWithPassword()...", zap.Error(err))
		return
	}
	// 判断数据库中是否存在上次的登录凭据，如果不存在，需要邮箱验证码登录
	if err := logic.VerifyLoginToken(u.UserId, logic.LoginTokenExpireDuration); err != nil {
		ResponseError(context, CODE_TOO_LONG_NOT_LOGIN)
		zap.L().Error("login token expired,need to login very email verification code...", zap.Error(err))
		return
	}
	// 继续后续步骤
	res, err := logic.SignInPostProcess(u)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		zap.L().Error("user sign in postprocess error in controller.SignInWithPassword()...", zap.Error(err))
		return
	}
	ResponseSuccess(context, res)
}

// SignInViaEmail 处理账户登录
// @Summary 实现用户登录功能
// @Description 接受用户输入的email，验证码，返回refresh-token 和 access-token
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamUserSignInViaEmail true "登录必备信息"
// @Success 200 {object} _ResponseUserSignIn
// @Router /api/v1/login-via-email [post]
func SignInViaEmail(context *gin.Context) {
	// 1. 参数校验
	user := new(models.ParamUserSignInViaEmail)
	if err := context.ShouldBindJSON(user); err != nil {
		ResponseError(context, CODE_PARAM_ERROR)
		zap.L().Error("user sign in parameter bind error...", zap.Error(err))
		return
	}
	// 2.调用业务逻辑层
	u := new(models.User)
	u.Email = user.Email
	u.Password = user.VerificationCode
	if err := logic.SignInWithEmailVerificationCode(u); err != nil {
		ResponseError(context, CODE_VERFICATION_CODE_ERROR)
		zap.L().Error("user sign in verification code error in controller.SignInViaEmail()...", zap.Error(err))
		return
	}
	// 3.继续后续步骤

	res, err := logic.SignInPostProcess(u)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		zap.L().Error("user sign in postprocess error in controller.SignInWithPassword()...", zap.Error(err))
		return
	}
	ResponseSuccess(context, res)
}

// GetVerificationCode 获取登录的邮箱验证码
// @Summary 获取登录的邮箱验证码
// @Description 获取本次登录需要的验证码
// @Tags 用户相关接口
// @Produce application/json
// @Param email query string true "email"
// @Success 200 {object} _ResponseEmailVerificationCode
// @Router /api/v1/get-email-verification-code [get]
func GetVerificationCode(context *gin.Context) {
	email := context.DefaultQuery("email", "")
	if len(email) == 0 {
		ResponseError(context, CODE_PARAM_ERROR)
		return
	}
	// 在logic层产生验证码，存入Redis数据库
	code, err := logic.GenerateCode(email)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		return
	}
	// 发送邮件
	err = logic.SendVerificationCodeEmail(email, code)
	if err != nil {
		ResponseError(context, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(context, nil)
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
		return
	}
	// 调用逻辑层去修改数据库
	if err = logic.EditUserInfo(info, c.GetInt64(ContextUserIdKey)); err != nil {
		zap.L().Error("edit user info error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
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
		return
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
		return
	}
	ResponseSuccess(c, accessToken)
}

// SendEmail 实现获取邮箱验证码的接口
// @Summary 发送验证邮件
// @Description 产生验证链接，存入redis数据库，并发送给用户
// @Tags 用户相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseUserSendEmail
// @Router /api/v1/send-email [post]
func SendEmail(c *gin.Context) {
	// 需要获取当前用户的邮箱，然后生成一个验证链接，发送到用户邮箱
	email, err := logic.GetEmailById(c.GetInt64(ContextUserIdKey))
	if err != nil {
		zap.L().Error("get user editable info error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	// 接下来是logic层的范围了，给一个email,完成后续任务
	err = logic.SendEmailVerification(email)
	if err != nil {
		zap.L().Error("send email verification error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}

// VerifyEmail 实现验证用户邮箱功能
// @Summary 验证用户邮箱
// @Description 查看传入的info和redis数据库中存放的info，判断是否通过验证
// @Tags 用户相关接口
// @Produce application/json
// @Param info query string true "info"
// @Success 200 {object} _ResponseUserVerifyEmail
// @Router /api/v1/verify-email [get]
func VerifyEmail(c *gin.Context) {
	// 传入参数是info
	info := c.Query("info")
	if len(info) == 0 {
		zap.L().Warn("info is empty")
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	err := logic.VerifyEmail(info)
	if err != nil {
		if errors.Is(err, logic.ERROR_EMAIL_VERIFIED_BEFORE) {
			ResponseSuccess(c, nil)
			return
		}
		zap.L().Warn("verify email error", zap.Error(err))
		ResponseError(c, CODE_VERIFY_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}
