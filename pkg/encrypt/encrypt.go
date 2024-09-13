package encrypt

import (
	"crypto/md5"
	"encoding/hex"
)

// 进行加密操作
var salt = "www.chan.com"

func Encrypt(pwd string) string {
	h := md5.New()
	h.Write([]byte(salt))
	h.Write([]byte(pwd))
	return hex.EncodeToString(h.Sum(nil))
}
