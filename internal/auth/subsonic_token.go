package auth

import (
	"crypto/hmac"
	"crypto/md5"
	"fmt"
)

func VerifySubsonicToken(plainPassword, salt, token string) bool {
	expected := fmt.Sprintf("%x", md5.Sum([]byte(plainPassword+salt)))
	return hmac.Equal([]byte(expected), []byte(token))
}
