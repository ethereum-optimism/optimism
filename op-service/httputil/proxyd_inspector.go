package httputil

import (
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

// ProxydInspector implements a HTTPInterceptor to monitor cache headers in HTTP responses.
type ProxydInspector struct {
	ServedBy          string
	ProxydCacheStatus string
}

var _ HTTPInterceptor = (*ProxydInspector)(nil)

func (chi *ProxydInspector) Intercept(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
	resp, err = inner.RoundTrip(req)
	if resp != nil {
		chi.ServedBy = resp.Header.Get("X-Served-By")
		chi.ProxydCacheStatus = resp.Header.Get("X-Proxyd-Cache-Status")
	}
	return
}

func (chi *ProxydInspector) LogContext(logger log.Logger) log.Logger {
	return logger.New("served_by", chi.ServedBy, "proxyd_cache_status", chi.ProxydCacheStatus)
}
