package lxcapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func charCases(char string) string {
	caser := cases.Title(language.English)
	return caser.String(strings.ToLower(char))
}

func convertMaskToCIDR(mask net.IPMask) string {
	ones, _ := mask.Size()
	return fmt.Sprintf("%d", ones)
}

func generateFds(length int) (string, error) {
	byteLength := length / 2
	randomBytes := make([]byte, byteLength)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}
