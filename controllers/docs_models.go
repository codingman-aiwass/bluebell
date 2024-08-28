package controllers

import "bluebell/models"

type _ResponseUserSignUp struct {
	Code ResponseCode `json:"code"`    // 业务状态响应码
	Msg  string       `json:"message"` // 提示信息
}

type _ResponseUserSignIn struct {
	Code  ResponseCode      `json:"code"`    // 业务状态响应码
	Msg   string            `json:"message"` // 提示信息
	Token map[string]string `json:"token"`   // refresh-token and access-token
}

type _ResponseUserEditInfo struct {
	Code ResponseCode `json:"code"`    // 业务状态响应码
	Msg  string       `json:"message"` // 提示信息
}

type _ResponsePostCreate struct {
	Code ResponseCode `json:"code"`    // 业务状态响应码
	Msg  string       `json:"message"` // 提示信息
}

type _ResponsePostDetail struct {
	Code     ResponseCode `json:"code"`      // 业务状态响应码
	Msg      string       `json:"message"`   // 提示信息
	PostInfo models.Post  `json:"post_info"` // post list
}
type _ResponsePostList struct {
	Code    ResponseCode   `json:"code"`    // 业务响应状态码
	Message string         `json:"message"` // 提示信息
	Data    []*models.Post `json:"data"`    // 数据
}
type _ResponseVotePost struct {
	Code ResponseCode `json:"code"`    // 业务状态响应码
	Msg  string       `json:"message"` // 提示信息
}
type _ResponseCommunities struct {
	Code ResponseCode        `json:"code"`    // 业务状态响应码
	Msg  string              `json:"message"` // 提示信息
	Data []*models.Community `json:"data"`    // community list
}
