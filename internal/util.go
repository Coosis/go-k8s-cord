package internal

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

/// Helper function to restrict access to localhost
func LocalOnly(f func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Error("Failed to parse remote address:", err)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		rip := net.ParseIP(host)
		if rip == nil || !(rip.IsLoopback()) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		f(w, r)
	}
}

/// Helper function to get the CA certificate pool
func LoadCAPool(certname string) (*x509.CertPool, error) {
	cas := x509.NewCertPool()
	caCert, err := os.ReadFile(certname)
	if err != nil {
		log.Error("Failed to read CA certificate:", err)
		return nil, err
	}
	cas.AppendCertsFromPEM(caCert)
	return cas, nil
}

func AgentClientWithCert(ca *x509.CertPool) (*http.Client, error) {
	clientCert, err := tls.LoadX509KeyPair("agent.crt", "agent.key")
	if err != nil {
		log.Error("Failed to load client certificate:", err)
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      ca,
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

