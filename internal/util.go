package internal

import (
	"crypto/x509"
	"os"

	log "github.com/sirupsen/logrus"
)

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
