package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Healthz struct {
	ctx    context.Context
	server *http.Server
}

func (h *Healthz) Start(ctx context.Context, host string, port int) {
	go func() {
		hdlr := mux.NewRouter()
		hdlr.HandleFunc("/healthz", h.Handle).Methods("GET")
		addr := fmt.Sprintf("%s:%d", host, port)
		c := cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
		})
		server := &http.Server{
			Handler: c.Handler(hdlr),
			Addr:    addr,
		}
		h.server = server
		h.ctx = ctx
		err := h.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Crit("error starting healthz server", "err", err)
		}
	}()
}

func (h *Healthz) Shutdown() error {
	return h.server.Shutdown(h.ctx)
}

func (h *Healthz) Handle(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
