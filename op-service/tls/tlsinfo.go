package tls

import (
	"context"
	"crypto/x509"
	"net/http"
)

// PeerTLSInfo contains request-scoped peer certificate data
// It can be used by downstream http.Handlers to authorize access for TLS-authenticated clients
type PeerTLSInfo struct {
	LeafCertificate *x509.Certificate
}

type peerTLSInfoContextKey struct{}

// NewPeerTLSMiddleware returns an http.Handler that extracts the peer's certificate data into PeerTLSInfo and attaches it to the request-scoped context.
// PeerTLSInfo will only be populated if the http.Server is listening with ListenAndServeTLS
// This is useful for ethereum-go/rpc endpoints because the http.Request object isn't accessible in the registered service.
func NewPeerTLSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerTlsInfo := PeerTLSInfo{}
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			peerTlsInfo.LeafCertificate = r.TLS.PeerCertificates[0]
		}
		ctx := context.WithValue(r.Context(), peerTLSInfoContextKey{}, peerTlsInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PeerTLSInfoFromContext extracts PeerTLSInfo from the context
// Result will only be populated if NewPeerTLSMiddleware has been added to the handler stack.
func PeerTLSInfoFromContext(ctx context.Context) PeerTLSInfo {
	info, _ := ctx.Value(peerTLSInfoContextKey{}).(PeerTLSInfo)
	return info
}
