package modules

import (
	"encoding/base64"
	"errors"
	"github.com/thanhpk/randstr"
	"strings"
)

var ERROR_INVLID_VERIFICATION_INFO = errors.New("Invalid Verification Information")

// GenCode 返回指定位数的随机字符串
func GenCode(len int) string {
	return randstr.String(len)
}

// Encode 将给定字符串编码,采用base64编码
func Encode(s string) string {
	data := base64.StdEncoding.EncodeToString([]byte(s))
	return data
}
func Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// 存放和email相关的函数
func GenEmailVerificationInfo(email string) string {
	code := GenCode(24)
	info := Encode(email + "|" + code)
	return info

}

// 从info中提取出email code
func ParseEmailVerificationInfo(info string) (email, code string, err error) {
	data, err := Decode(info)
	if err != nil {
		return "", "", err
	}
	s := strings.Split(data, "|")
	if len(s) != 2 {
		return "", "", ERROR_INVLID_VERIFICATION_INFO
	}
	return s[0], s[1], nil
}
