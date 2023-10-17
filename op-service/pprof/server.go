package pprof

import (
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
)

func StartServer(hostname string, port int) (*httputil.HTTPServer, error) {
	mux := http.NewServeMux()

	// have to do below to support multiple servers, since the
	// pprof import only uses DefaultServeMux
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	addr := net.JoinHostPort(hostname, strconv.Itoa(port))
	return httputil.StartHTTPServer(addr, mux)
}
