package httputil

import (
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

func NewLoggingMiddleware(lgr log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := NewWrappedResponseWriter(w)
		start := time.Now()
		next.ServeHTTP(ww, r)
		lgr.Debug(
			"served HTTP request",
			"status", ww.StatusCode,
			"response_len", ww.ResponseLen,
			"path", r.URL.EscapedPath(),
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
			"upgrade_attempt", ww.UpgradeAttempt,
		)
	})
}
