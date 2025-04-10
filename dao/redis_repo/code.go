package redis_repo

import (
	"errors"
	"time"
)

var ERROR_MORE_THAN_ONE_USER = errors.New("More than one user is active!")
var ERROR_EXPIRED_POST = errors.New("Expired post")
var ERROR_EMAIL_SEND_FAILED = errors.New("Email send failed")
var ERROR_EMAIL_INFO_NOT_EXISTS = errors.New("Email verification info not exists")
var ERROR_GAP_TOO_LONG = errors.New("Last login too long ago")
var ERROR_EMAIL_INVALID_VERIFICATION_CODE = errors.New("Invalid verification code")

const (
	PER_VOTE_VALUE                    = 416
	POST_VALID_TIME                   = 10 * 365 * 24 * time.Hour
	EMAIL_VERFICATION_VALID_TIME      = 15 * time.Hour
	EMAIL_LOGIN_CODE_VALID_TIME       = 10 * time.Minute
	UserLikeOrDislike2PostBloomFilter = "user_like_or_dislike_to_post_filter"
	UserCollection2PostBloomFilter    = "user_collection_to_filter"
)
