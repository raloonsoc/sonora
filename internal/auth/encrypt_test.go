package auth

import "testing"

func TestEncryptReversible_DecryptReversible(t *testing.T) {
	ciphertext, err := EncryptReversible("test-secret", "hunter2")
	if err != nil {
		t.Fatalf("EncryptReversible: %v", err)
	}

	plaintext, err := DecryptReversible("test-secret", ciphertext)
	if err != nil {
		t.Fatalf("DecryptReversible: %v", err)
	}
	if plaintext != "hunter2" {
		t.Errorf("plaintext = %q, want %q", plaintext, "hunter2")
	}
}

func TestDecryptReversible_WrongSecret(t *testing.T) {
	ciphertext, err := EncryptReversible("test-secret", "hunter2")
	if err != nil {
		t.Fatalf("EncryptReversible: %v", err)
	}

	if _, err := DecryptReversible("wrong-secret", ciphertext); err == nil {
		t.Error("expected an error when decrypting with the wrong secret, got nil")
	}
}

func TestEncryptReversible_ProducesDifferentCiphertextEachTime(t *testing.T) {
	ciphertext1, err := EncryptReversible("test-secret", "hunter2")
	if err != nil {
		t.Fatalf("EncryptReversible: %v", err)
	}
	ciphertext2, err := EncryptReversible("test-secret", "hunter2")
	if err != nil {
		t.Fatalf("EncryptReversible: %v", err)
	}

	if string(ciphertext1) == string(ciphertext2) {
		t.Error("two encryptions of the same plaintext should differ (random nonce)")
	}
}
