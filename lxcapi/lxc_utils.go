package lxcapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
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

func ReadClientConfig(confFile string) *Config {
	configFile, err := os.Open(confFile)
	if err != nil {
		log.Fatalf("Unable to open config file: %v\n", err)
	}
	defer configFile.Close()

	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Bad config file: %v\n", err)
	}
	configFile.Close()
	return &config
}
