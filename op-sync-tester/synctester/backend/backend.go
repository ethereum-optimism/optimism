package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
)

type sessionKeyType struct{}

var CtxKeySession = sessionKeyType{}

type Session struct {
	Head      uint64
	Safe      uint64
	Finalized uint64
}

func (s *Session) ID() string {
	key := fmt.Sprintf("head=%d&safe=%d&finalized=%d", s.Head, s.Safe, s.Finalized)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:8])
}

type APIRouter interface {
	AddRPC(route string) error
	AddAPIToRPC(route string, api rpc.API) error
}

type Backend struct {
	log      log.Logger
	m        metrics.Metricer
	mu       sync.Mutex
	sessions map[string]*Session
}

func (b *Backend) Stop(ctx context.Context) error {
	// We have support for ctx/error here,
	// for future improvements like awaiting txs to complete and/or storing rate-limit data to disk.
	return nil
}

func FromConfig(log log.Logger, m metrics.Metricer, cfg *config.Config) (*Backend, error) {
	b := &Backend{
		log:      log,
		m:        m,
		sessions: make(map[string]*Session),
	}
	return b, nil
}

func (b *Backend) Init(ctx context.Context) error {
	session, ok := ctx.Value(CtxKeySession).(*Session)
	if !ok || session == nil {
		return fmt.Errorf("no session found in context")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if existing, ok := b.sessions[session.ID()]; ok {
		b.log.Info("Using existing session", "session", existing)
	} else {
		b.sessions[session.ID()] = session
		b.log.Info("Initialized new session", "session", session)
	}
	return nil
}

func (b *Backend) ClearSessions(ctx context.Context) {
	b.log.Info("Clearing sessions")
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions = make(map[string]*Session)
}
