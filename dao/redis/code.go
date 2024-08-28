package redis

import "errors"

var ERROR_MORE_THAN_ONE_USER = errors.New("More than one user is active!")
var ERROR_EXPIRED_POST = errors.New("Expired post")
var ERROR_EMAIL_SEND_FAILED = errors.New("Email send failed")
var ERROR_EMAIL_INFO_NOT_EXISTS = errors.New("Email verification info not exists")

const (
	PerVoteValue  = 416
	PostValidTime = 7 * 24 * 60 * 60
)
