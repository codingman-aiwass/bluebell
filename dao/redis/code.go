package redis

import "errors"

var ERROR_MORE_THAN_ONE_USER = errors.New("More than one user is active!")
var ERROR_EXPIRED_POST = errors.New("Expired post")

const (
	PerVoteValue  = 416
	PostValidTime = 7 * 24 * 60 * 60
)
