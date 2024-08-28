package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ResponseCode int

const ContextUserIdKey = "user_id"
const ContextUserNameKey = "username"
const (
	CODE_SUCCESS = 100 * iota
	CODE_USER_EXISTS
	CODE_USER_NOT_EXSITS
	CODE_PARAM_ERROR
	CODE_PASSWORD_ERROR
	CODE_INTERNAL_ERROR
	CODE_INVALID_TOKEN
	CODE_NOT_LOGIN
	CODE_MORE_THAN_ONE_USER
	CODE_NO_ROW_IN_DB
)

var code_to_msg = map[ResponseCode]string{
	CODE_SUCCESS:            "success",
	CODE_USER_EXISTS:        "user has already existed",
	CODE_USER_NOT_EXSITS:    "user does not exist",
	CODE_PARAM_ERROR:        "parameter error",
	CODE_PASSWORD_ERROR:     "username or password error",
	CODE_INTERNAL_ERROR:     "internal server error",
	CODE_INVALID_TOKEN:      "invalid token",
	CODE_NOT_LOGIN:          "not login",
	CODE_MORE_THAN_ONE_USER: "more than one user",
	CODE_NO_ROW_IN_DB:       "no data",
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
	Msg  string       `json:"msg"`
	Data interface{}  `json:"data,omitempty"`
}

func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CODE_SUCCESS,
		Msg:  getMsg(CODE_SUCCESS),
		Data: data,
	})
}

func ResponseError(c *gin.Context, code ResponseCode) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  getMsg(code),
	})
}
