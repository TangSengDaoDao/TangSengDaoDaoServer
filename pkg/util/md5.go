package util

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

//MD5 加密
func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str)) // 需要加密的字符串
	passwordmdsBys := h.Sum(nil)
	return hex.EncodeToString(passwordmdsBys)
}

// SHA1加密
func SHA1(str string) string {
	fmt.Println("str:", str)
	h := sha1.New()
	h.Write([]byte(str))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

func HMACSHA1(keyStr string, data string) string {
	//hmac ,use sha1
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(data))
	srcBytes := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(srcBytes)
}
