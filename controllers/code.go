package controllers

import (
	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

type ResponseCode int

const (
	NORMAL_STATUS = iota
	NOT_ALLOW_CREATE_POST
)
const (
	EMAIL_NOT_VERIFIED = false
	EMAIL_VERFIED      = true
)

const ContextUserIdKey = "user_id"
const ContextUserNameKey = "username"
const (
	CODE_SUCCESS = 100 * iota
	CODE_USER_EXISTS
	CODE_EMAIL_EXSITS
	CODE_USER_NOT_EXSITS
	CODE_PARAM_ERROR
	CODE_PASSWORD_ERROR
	CODE_INTERNAL_ERROR
	CODE_INVALID_TOKEN
	CODE_NOT_LOGIN
	CODE_MORE_THAN_ONE_USER
	CODE_NO_ROW_IN_DB
	CODE_VERIFY_ERROR
	CODE_TOO_LONG_NOT_LOGIN
	CODE_VERFICATION_CODE_ERROR
	CODE_NOT_ALLOW_PUBLISH_POST
	CODE_NOT_ALLOW_PUBLISH_COMMENT
)

var code_to_msg = map[ResponseCode]string{
	CODE_SUCCESS:                   "success",
	CODE_USER_EXISTS:               "user has already existed",
	CODE_EMAIL_EXSITS:              "email has already existed",
	CODE_USER_NOT_EXSITS:           "user does not exist",
	CODE_PARAM_ERROR:               "parameter error",
	CODE_PASSWORD_ERROR:            "username or password error",
	CODE_INTERNAL_ERROR:            "internal server error",
	CODE_INVALID_TOKEN:             "invalid token",
	CODE_NOT_LOGIN:                 "not login",
	CODE_MORE_THAN_ONE_USER:        "more than one user",
	CODE_NO_ROW_IN_DB:              "no data",
	CODE_VERIFY_ERROR:              "verification error",
	CODE_TOO_LONG_NOT_LOGIN:        "too long since last login",
	CODE_VERFICATION_CODE_ERROR:    "verification code error",
	CODE_NOT_ALLOW_PUBLISH_POST:    "not allow publish post",
	CODE_NOT_ALLOW_PUBLISH_COMMENT: "not allow publish comment",
}

func getMsg(code ResponseCode) string {
	if _, ok := code_to_msg[code]; !ok {
		return code_to_msg[CODE_INTERNAL_ERROR]
	} else {
		return code_to_msg[code]
	}
}

type Response struct {
	Code ResponseCode `json:"code"`
	Msg  string       `json:"message"`
	Data interface{}  `json:"data,omitempty"`
}

func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CODE_SUCCESS,
		Msg:  getMsg(CODE_SUCCESS),
		Data: data,
	})
}

func ResponseCaptcha(c *gin.Context, captchaId string) {
	c.Writer.Header().Set("Content-Type", "image/png")
	err := captcha.WriteImage(c.Writer, captchaId, captcha.StdWidth, captcha.StdHeight)
	if err != nil {
		zap.L().Error("generate captcha picture error in captcha.WriteImage()...", zap.Error(err))
		c.JSON(http.StatusOK, Response{
			Code: CODE_INTERNAL_ERROR,
			Msg:  getMsg(CODE_INTERNAL_ERROR),
		})
		return
	}
}

func ResponseError(c *gin.Context, code ResponseCode) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  getMsg(code),
	})
}
