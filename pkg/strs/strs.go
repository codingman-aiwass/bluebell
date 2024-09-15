package strs

import (
	uuid "github.com/iris-contrib/go.uuid"
	"strings"
	"unicode"
)

func IsBlank(str string) bool {
	strLen := len(str)
	if strLen == 0 {
		return true
	}
	for i := 0; i < len(str); i++ {
		if unicode.IsSpace(rune(str[i])) == false {
			return false
		}
	}
	return true
}

// RuneLen 字符串长度
func RuneLen(s string) int {
	bt := []rune(s)
	return len(bt)
}

func UUID() string {
	u, _ := uuid.NewV4()
	return strings.ReplaceAll(u.String(), "-", "")
}
