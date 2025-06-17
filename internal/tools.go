package tools

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func SpaHandler(uiDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/ui")
		if path == "" || path == "/" {
			path = "/index.html"
		}

		fullPath := filepath.Join(uiDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			http.ServeFile(w, r, fullPath)
			return
		}
		http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
	}
}

func GenerateSelfSignedCert() (tls.Certificate, error) {
	configFile, err := os.Open("config.yaml")
	defer configFile.Close()
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Unable to generate private key: %v", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "LXC-UI",
			Organization: []string{"LXC-UI"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		DNSNames:              []string{"127.0.0.1", "::1"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Certificate generation failed: %v", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  priv,
	}

	return cert, nil
}

func LoadCert(certPath, keyPath string) (tls.Certificate, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Failed to read certificate file: %v", err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Failed to read private key file: %v", err)
	}

	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("Failed to parse certificate and private key: %v", err)
	}

	return cert, nil
}
