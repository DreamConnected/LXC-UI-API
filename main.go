package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	tools "github/dreamconnected/lxc-ui-api/internal"
	"github/dreamconnected/lxc-ui-api/lxcapi"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

// Config
var config Config

type Config struct {
	Server struct {
		IP            string `yaml:"ip"`
		Port          int    `yaml:"port"`
		Cert          string `yaml:"cert"`
		ServerCert    string `yaml:"server-cert"`
		ServerCertKey string `yaml:"server-cert-key"`
	} `yaml:"server"`
}

func main() {
	var cert tls.Certificate
	configFile, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("Unable to open config file: %v\n", err)
	}
	defer configFile.Close()

	decoder := yaml.NewDecoder(configFile)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Bad config file: %v\n", err)
	}

	address := fmt.Sprintf("%s:%d", config.Server.IP, config.Server.Port)
	fmt.Printf("Start LXC-API service: %s\n", address)

	if config.Server.ServerCert == "" && config.Server.ServerCertKey == "" {
		cert, err = tools.GenerateSelfSignedCert()
	} else {
		cert, err = tools.LoadCert(config.Server.ServerCert, config.Server.ServerCertKey)
	}

	caCert, err := os.ReadFile(config.Server.Cert)
	if err != nil {
		log.Fatalf("Unable to read client CA certificate: %v\n", err)
	}
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(caCert)

	// TLS Config
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS13,
		ClientCAs:          clientCAs,
		ClientAuth:         tls.VerifyClientCertIfGiven,
		InsecureSkipVerify: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/1.0/events", lxcapi.HandleOperationsWebSocket)
	mux.HandleFunc("/1.0/operations/", lxcapi.HandleOperationsWebSocketTerminal)
	lxc_ui_path, exists := os.LookupEnv("LXC_UI")
	if exists {
		mux.HandleFunc("/ui/", tools.SpaHandler(lxc_ui_path))
	} else {
		mux.HandleFunc("/ui/", tools.SpaHandler("./ui"))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	mux.HandleFunc("/1.0", lxcapi.SyncHandler)
	mux.HandleFunc("/1.0/projects", lxcapi.ProjectHandler)
	mux.HandleFunc("/1.0/profiles", lxcapi.ProfilesHandler)
	mux.HandleFunc("/1.0/projects/default", lxcapi.ProjectDefaultHandler)
	mux.HandleFunc("/1.0/operations", lxcapi.OperationsHandler)
	mux.HandleFunc("/1.0/instances", lxcapi.InstancesHandler)
	mux.HandleFunc("/1.0/instances/", lxcapi.InstancesHandler)
	mux.HandleFunc("/1.0/certificates", lxcapi.CertificatesHandler)
	mux.HandleFunc("/1.0/networks", lxcapi.NetworksHandler)
	mux.HandleFunc("/1.0/networks/", lxcapi.NetworksHandler)

	server := &http.Server{
		Addr:      address,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Service startup failed: %v\n", err)
	}
}
