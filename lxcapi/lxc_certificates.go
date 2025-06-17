package lxcapi

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
)

// CertificatesHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
func CertificatesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
	} else {
		response = map[string]any{
			"type":        "sync",
			"status":      "error",
			"status_code": 400,
			"operation":   "",
			"error_code":  1,
			"error":       "Untrusted client certificate",
			"metadata":    nil,
		}
	}

	json.NewEncoder(w).Encode(response)
}
