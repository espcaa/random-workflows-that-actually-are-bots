package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GenerateCodeVerifier(length int) (string, error) {
	if length < 43 || length > 128 {
		return "", fmt.Errorf("length must be between 43 and 128")
	}

	numBytes := (length * 3) / 4
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	verifier := base64.RawURLEncoding.EncodeToString(randomBytes)
	if len(verifier) > length {
		verifier = verifier[:length]
	}
	return verifier, nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
