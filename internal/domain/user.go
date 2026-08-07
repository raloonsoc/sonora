package domain

import "time"

type User struct {
	ID                string
	Username          string
	PasswordEncrypted []byte
	IsAdmin           bool
	CreatedAt         time.Time
}
