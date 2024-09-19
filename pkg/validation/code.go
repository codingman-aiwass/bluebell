package validation

import "errors"

var (
	ERROR_TOO_MANY_PUBLISH = errors.New("too many publish within a short period")
)
