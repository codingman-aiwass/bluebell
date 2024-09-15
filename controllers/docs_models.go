package controllers

import "bluebell/models"

type _ResponseUserSignUp struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}

type _ResponseUserSignIn struct {
	Code  ResponseCode      `json:"code" example:"200"`                                 // 业务状态响应码
	Msg   string            `json:"message" example:"ok"`                               // 提示信息
	Token map[string]string `json:"token" example:"refresh_token:xxx,access_token:xxx"` // refresh-token and access-token
}

type _ResponseUserEditInfo struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}

type _ResponsePostCreate struct {
	Code ResponseCode `json:"code"  example:"200"`   // 业务状态响应码
	Msg  string       `json:"message"  example:"ok"` // 提示信息
}

type _ResponsePostDetail struct {
	Code     ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg      string       `json:"message" example:"ok"` // 提示信息
	PostInfo models.Post  `json:"post_info"`            // post list
}
type _ResponsePostList struct {
	Code    ResponseCode   `json:"code" example:"200"`   // 业务响应状态码
	Message string         `json:"message" example:"ok"` // 提示信息
	Data    []*models.Post `json:"data"`                 // 数据
}
type _ResponseVotePost struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}
type _ResponseCommunities struct {
	Code ResponseCode        `json:"code" example:"200"`   // 业务状态响应码
	Msg  string              `json:"message" example:"ok"` // 提示信息
	Data []*models.Community `json:"data"`                 // community list
}
type _ResponseUserSendEmail struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}

type _ResponseUserVerifyEmail struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}

type _ResponseEmailVerificationCode struct {
	Code             ResponseCode `json:"code" example:"200"`                 // 业务状态响应码
	Msg              string       `json:"message" example:"ok"`               // 提示信息
	VerificationCode string       `json:"verification_code" example:"AbedEf"` // 验证码
}

type _ResponseCaptchaVerification struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
}

type _ResponseCaptchaInfo struct {
	Code ResponseCode `json:"code" example:"200"`   // 业务状态响应码
	Msg  string       `json:"message" example:"ok"` // 提示信息
	Data []string     `json:"data"`                 // catpcha id and url
}
