package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type HealthzServer struct {
	ctx    context.Context
	server *http.Server
}

func (h *HealthzServer) Start(ctx context.Context, addr string) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/healthz", h.Handle).Methods("GET")
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	server := &http.Server{
		Handler: c.Handler(hdlr),
		Addr:    addr,
	}
	h.server = server
	h.ctx = ctx
	return h.server.ListenAndServe()
}

func (h *HealthzServer) Shutdown() error {
	return h.server.Shutdown(h.ctx)
}

func (h *HealthzServer) Handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
