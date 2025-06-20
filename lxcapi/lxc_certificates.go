package lxcapi

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
)

var IsTrustedToken bool
var ClientToken string
var config Config

type TokenPayload struct {
	Type     string `json:"type"`
	Password string `json:"password"`
}

type Token struct {
	Token string `yaml:"token"`
}

type Config struct {
	Client struct {
		Tokens []Token `yaml:"tokens"`
	} `yaml:"client"`
}

// CertificatesHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
func CertificatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := io.ReadAll(r.Body)
	var payload TokenPayload
	json.Unmarshal(body, &payload)
	defer r.Body.Close()

	fmt.Println("Request Method:", r.Method, "|", "Request API:", r.URL.Path)

	var response map[string]any

	if len(r.TLS.PeerCertificates) > 0 {
		clientCert := r.TLS.PeerCertificates[0]

		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: clientCert.Raw,
		})
		certFingerprint := fmt.Sprintf("%x", sha256.Sum256(clientCert.Raw))

		response = map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"operation":   "",
			"error_code":  0,
			"error":       "",
			"metadata": []map[string]any{
				{
					"name":        "stdin",
					"type":        "client",
					"restricted":  false,
					"projects":    []string{},
					"certificate": string(certPEM),
					"description": nil,
					"fingerprint": certFingerprint,
				},
			},
		}
	} else if r.Method == http.MethodPost && payload.Type == "client" {
		config := ReadClientConfig("config.yaml")
		for _, clientToken := range config.Client.Tokens {
			if payload.Password == clientToken.Token {
				session, expires := Base64Token2Json(payload.Password)
				if session == "" {
					break
				}
				_, err := r.Cookie("client_id")
				if err != nil {
					http.SetCookie(w, &http.Cookie{
						Name:     "client_id",
						Value:    session,
						Expires:  expires,
						Secure:   true,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
				}
				// Token is just a token and does not verify fingerprint,
				// expires_at, and other information.
				// Under normal circumstances, token should be converted to
				// base64 with content similar to the following:
				// From json: {"client_name":"incus-ui","fingerprint":"0ba029714a9e1e93dcc8a0f960125c2ed82c05c19906ff7e254577e2361274cc","addresses":["127.0.0.1:8443","[::1]:8443"],"secret":"8ee82edf87034f4c24fb0f2472bb4ee742cbb0822c57b8ef92b63719ad3f705e","expires_at":"0001-01-01T00:00:00Z"}
				// To base64: eyJjbGllbnRfbmFtZSI6ImluY3VzLXVpIiwiZmluZ2VycHJpbnQiOiIwYmEwMjk3MTRhOWUxZTkzZGNjOGEwZjk2MDEyNWMyZWQ4MmMwNWMxOTkwNmZmN2UyNTQ1NzdlMjM2MTI3NGNjIiwiYWRkcmVzc2VzIjpbIjEyNy4wLjAuMTo4NDQzIiwiWzo6MV06ODQ0MyJdLCJzZWNyZXQiOiI4ZWU4MmVkZjg3MDM0ZjRjMjRmYjBmMjQ3MmJiNGVlNzQyY2JiMDgyMmM1N2I4ZWY5MmI2MzcxOWFkM2Y3MDVlIiwiZXhwaXJlc19hdCI6IjAwMDEtMDEtMDFUMDA6MDA6MDBaIn0=
				response = map[string]any{
					"type":        "sync",
					"status":      "Success",
					"status_code": 200,
				}
				IsTrustedToken = true
				ClientToken = clientToken.Token
			}
		}
	} else {
		response = map[string]any{
			"type":        "error",
			"status":      "error",
			"status_code": 403,
			"error_code":  403,
			"error":       "not authorized",
		}
	}

	json.NewEncoder(w).Encode(response)
}

func IsTrusted(r *http.Request) bool {
	if len(r.TLS.PeerCertificates) > 0 {
		return true
	} else if IsTrustedToken {
		return true
	}
	return false
}
