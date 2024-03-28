package plasma

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type KVStore interface {
	// Get retrieves the given key if it's present in the key-value data store.
	Get(ctx context.Context, key []byte) ([]byte, error)
	// Put inserts the given value into the key-value data store.
	Put(ctx context.Context, key []byte, value []byte) error
}

type MemStore struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		db: make(map[string][]byte),
	}
}

// Get retrieves the given key if it's present in the key-value store.
func (s *MemStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if entry, ok := s.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, ErrNotFound
}

// Put inserts the given value into the key-value store.
func (s *MemStore) Put(ctx context.Context, key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.db[string(key)] = common.CopyBytes(value)
	return nil
}

type DAServer struct {
	log        log.Logger
	endpoint   string
	store      KVStore
	tls        *rpc.ServerTLSConfig
	httpServer *http.Server
	listener   net.Listener
}

type DAServerOption func(b *DAServer)

func WithLogger(log log.Logger) DAServerOption {
	return func(b *DAServer) {
		b.log = log
	}
}

func NewDAServer(host string, port int, store KVStore, opts ...DAServerOption) *DAServer {
	endpoint := net.JoinHostPort(host, strconv.Itoa(port))
	return &DAServer{
		log:      log.Root(),
		endpoint: endpoint,
		store:    store,
		httpServer: &http.Server{
			Addr: endpoint,
		},
	}
}

func (d *DAServer) Start() error {
	mux := http.NewServeMux()

	mux.Handle("/get/", http.StripPrefix("/get/", http.HandlerFunc(d.HandleGet)))
	mux.Handle("/put/", http.StripPrefix("/put/", http.HandlerFunc(d.HandlePut)))

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

	comm, err := hexutil.Decode(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	input, err := d.store.Get(r.Context(), comm)
	// cannot use errors.Is because different implementations may use different ErrNotFound types
	if err != nil && errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(input); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (d *DAServer) HandlePut(w http.ResponseWriter, r *http.Request) {
	d.log.Debug("PUT", "url", r.URL)

	input, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	comm, err := hexutil.Decode(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := d.store.Put(r.Context(), comm, input); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(comm); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
