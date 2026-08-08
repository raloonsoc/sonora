package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// deriveKey hashes an arbitrary-length secret into a 32-byte AES-256 key.
func deriveKey(secret string) [32]byte {
	return sha256.Sum256([]byte(secret))
}

// EncryptReversible encrypts plaintext with AES-256-GCM, using secret to
// derive the key. The nonce is generated randomly and prepended to the
// ciphertext, since GCM needs it again to decrypt.
func EncryptReversible(secret, plaintext string) ([]byte, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("auth: creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth: creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("auth: generating nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// DecryptReversible reverses EncryptReversible.
func DecryptReversible(secret string, ciphertext []byte) (string, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("auth: creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("auth: creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("auth: ciphertext too short")
	}
	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("auth: decrypting: %w", err)
	}
	return string(plaintext), nil
}
