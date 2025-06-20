package lxcapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var AccessTokens = make(map[string]*Base64Token)

type Base64Token struct {
	Secret    string    `json:"secret"`
	ExpiresAt time.Time `json:"expires_at"`
}

func Base64Token2Json(token string) (string, time.Time) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		fmt.Println("Not a base64 token:", err)
		return "", time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	var tokenJson Base64Token
	err = json.Unmarshal([]byte(decoded), &tokenJson)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
		return "", time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return tokenJson.Secret, tokenJson.ExpiresAt
}

func SaveToken(session string) error {
	mu.Lock()
	defer mu.Unlock()

	config := ReadClientConfig("config.yaml")
	for _, clientToken := range config.Client.Tokens {
		secret, expires := Base64Token2Json(clientToken.Token)
		if session == secret {
			token := &Base64Token{
				Secret:    session,
				ExpiresAt: expires,
			}
			AccessTokens[session] = token
		}
	}
	return nil
}

func DeleteToken(session string) error {
	if _, exists := AccessTokens[session]; exists {
		delete(AccessTokens, session)
		return nil
	}
	return fmt.Errorf("token %s not found", session)
}

func GetTokenIsAvailable(session string) bool {
	mu.Lock()
	defer mu.Unlock()

	if base64Token, exists := AccessTokens[session]; exists {
		if base64Token.ExpiresAt.Before(time.Now()) && !base64Token.ExpiresAt.Equal(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)) {
			DeleteToken(session)
			return false
		} else if base64Token.ExpiresAt.Equal(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)) {
			return true
		}
	}
	DeleteToken(session)
	return false
}
