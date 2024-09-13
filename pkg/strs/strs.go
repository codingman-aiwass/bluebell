package strs

import "unicode"

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
