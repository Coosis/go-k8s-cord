package server

import (
	// "crypto/x509"
	// "net/http"
)

const (
)

/// Wrapper to require a certificate for the request
// func(s *CentralServer) NeedCrt(
// 	f func(w http.ResponseWriter, r *http.Request),
// ) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
// 			http.Error(w, "Client certificate required", http.StatusForbidden)
// 			return
// 		}
//
// 		cert := r.TLS.PeerCertificates[0]
// 		opts := x509.VerifyOptions{
// 			Roots: s.cp,
// 		}
// 		if _, err := cert.Verify(opts); err != nil {
// 			http.Error(w, "Invalid client certificate", http.StatusForbidden)
// 			return
// 		}
//
// 		f(w, r)
// 	}
// }
//
