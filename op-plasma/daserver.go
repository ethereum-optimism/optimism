package plasma

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type KVStore interface {
	// Get retrieves the given key if it's present in the key-value data store.
	Get(ctx context.Context, key []byte) ([]byte, error)
	// Put inserts the given value into the key-value data store.
	Put(ctx context.Context, key []byte, value []byte) error
}

type DAServer struct {
	log        log.Logger
	endpoint   string
	store      KVStore
	tls        *rpc.ServerTLSConfig
	httpServer *http.Server
	listener   net.Listener
}

func NewDAServer(host string, port int, store KVStore, log log.Logger) *DAServer {
	endpoint := net.JoinHostPort(host, strconv.Itoa(port))
	return &DAServer{
		log:      log,
		endpoint: endpoint,
		store:    store,
		httpServer: &http.Server{
			Addr: endpoint,
		},
	}
}

func (d *DAServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/get/", d.HandleGet)
	mux.HandleFunc("/put/", d.HandlePut)

	d.httpServer.Handler = mux

	listener, err := net.Listen("tcp", d.endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	d.listener = listener

	d.endpoint = listener.Addr().String()
	errCh := make(chan error, 1)
	go func() {
		if d.tls != nil {
			if err := d.httpServer.ServeTLS(d.listener, "", ""); err != nil {
				errCh <- err
			}
		} else {
			if err := d.httpServer.Serve(d.listener); err != nil {
				errCh <- err
			}
		}
	}()

	// verify that the server comes up
	tick := time.NewTimer(10 * time.Millisecond)
	defer tick.Stop()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server failed: %w", err)
	case <-tick.C:
		return nil
	}
}

func (d *DAServer) HandleGet(w http.ResponseWriter, r *http.Request) {
	d.log.Debug("GET", "url", r.URL)

	route := path.Dir(r.URL.Path)
	if route != "/get" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := path.Base(r.URL.Path)
	comm, err := hexutil.Decode(key)
	if err != nil {
		d.log.Error("Failed to decode commitment", "err", err, "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	input, err := d.store.Get(r.Context(), comm)
	if err != nil && errors.Is(err, ErrNotFound) {
		d.log.Error("Commitment not found", "key", key, "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		d.log.Error("Failed to read commitment", "err", err, "key", key)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(input); err != nil {
		d.log.Error("Failed to write pre-image", "err", err, "key", key)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (d *DAServer) HandlePut(w http.ResponseWriter, r *http.Request) {
	d.log.Info("PUT", "url", r.URL)

	route := path.Dir(r.URL.Path)
	if route != "/put" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	input, err := io.ReadAll(r.Body)
	if err != nil {
		d.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.URL.Path == "/put" || r.URL.Path == "/put/" { // without commitment

		comm := GenericCommitment(crypto.Keccak256Hash(input).Bytes())
		if err = d.store.Put(r.Context(), comm.Encode(), input); err != nil {
			d.log.Error("Failed to store commitment to the DA server", "err", err, "comm", comm)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(comm); err != nil {
			d.log.Error("Failed to write commitment request body", "err", err, "comm", comm)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	} else {
		key := path.Base(r.URL.Path)
		comm, err := hexutil.Decode(key)
		if err != nil {
			d.log.Error("Failed to decode commitment", "err", err, "key", key)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := d.store.Put(r.Context(), comm, input); err != nil {
			d.log.Error("Failed to store commitment to the DA server", "err", err, "key", key)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)

}

func (b *DAServer) Endpoint() string {
	return b.listener.Addr().String()
}

func (b *DAServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = b.httpServer.Shutdown(ctx)
	return nil
}
