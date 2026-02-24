package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

type TokenCipher struct {
	aead cipher.AEAD
}

func NewTokenCipher(secret string) (*TokenCipher, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("missing USER_KEY_ENCRYPTION_KEY")
	}

	key := []byte(secret)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil {
			key = decoded
		} else if decoded, err := base64.RawStdEncoding.DecodeString(secret); err == nil {
			key = decoded
		} else if decoded, err := base64.RawURLEncoding.DecodeString(secret); err == nil {
			key = decoded
		}
	}

	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("invalid USER_KEY_ENCRYPTION_KEY length: got %d, need 16/24/32 bytes (or base64 thereof)", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &TokenCipher{aead: aead}, nil
}

func (c *TokenCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("empty plaintext")
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	out := c.aead.Seal(nil, nonce, []byte(plaintext), nil)
	joined := append(nonce, out...)
	return base64.RawURLEncoding.EncodeToString(joined), nil
}

func (c *TokenCipher) Decrypt(ciphertext string) (string, error) {
	encoded := strings.TrimSpace(ciphertext)
	if encoded == "" {
		return "", errors.New("empty ciphertext")
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	nonceSize := c.aead.NonceSize()
	if len(raw) <= nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, payload := raw[:nonceSize], raw[nonceSize:]
	plain, err := c.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
