package mysql_repo

import "errors"

var ERROR_USER_NOT_EXISTED = errors.New("User does not exist...")
var ERROR_USER_EXISTS = errors.New("User has already exists...")
var ERROR_WRONG_PASSWORD = errors.New("Wrong password")
