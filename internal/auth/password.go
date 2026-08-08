package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func VerifyPassword(hashed []byte, password string) bool {
	err := bcrypt.CompareHashAndPassword(hashed, []byte(password))
	return err == nil
}
