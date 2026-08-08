package auth

import "testing"

func TestHashPassword_VerifyPassword(t *testing.T) {
	hashed, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !VerifyPassword(hashed, "correct horse battery staple") {
		t.Error("VerifyPassword returned false for the correct password")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hashed, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if VerifyPassword(hashed, "wrong password") {
		t.Error("VerifyPassword returned true for an incorrect password")
	}
}

func TestHashPassword_ProducesDifferentHashesEachTime(t *testing.T) {
	hashed1, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	hashed2, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if string(hashed1) == string(hashed2) {
		t.Error("two hashes of the same password should differ (bcrypt uses a random salt)")
	}
}
