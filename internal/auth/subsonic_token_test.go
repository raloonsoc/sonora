package auth

import (
	"crypto/md5"
	"fmt"
	"testing"
)

func TestVerifySubsonicToken_Correct(t *testing.T) {
	password := "sesame"
	salt := "c19b2d"
	token := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))

	if !VerifySubsonicToken(password, salt, token) {
		t.Error("VerifySubsonicToken returned false for a correctly computed token")
	}
}

func TestVerifySubsonicToken_WrongPassword(t *testing.T) {
	salt := "c19b2d"
	token := fmt.Sprintf("%x", md5.Sum([]byte("sesame"+salt)))

	if VerifySubsonicToken("wrong-password", salt, token) {
		t.Error("VerifySubsonicToken returned true for the wrong password")
	}
}

func TestVerifySubsonicToken_WrongSalt(t *testing.T) {
	token := fmt.Sprintf("%x", md5.Sum([]byte("sesame"+"c19b2d")))

	if VerifySubsonicToken("sesame", "different-salt", token) {
		t.Error("VerifySubsonicToken returned true for the wrong salt")
	}
}
