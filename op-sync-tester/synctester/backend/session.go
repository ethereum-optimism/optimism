package backend

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type SessionManager struct {
	mu                sync.RWMutex
	deletedSessionIDs map[string]interface{}

	log log.Logger

	sessions sync.Map // map[string]*eth.SyncTesterSession
}

func NewSessionManager(logger log.Logger) *SessionManager {
	return &SessionManager{log: logger, deletedSessionIDs: make(map[string]any)}
}

func (s *SessionManager) SessionIDs() []string {
	keys := make([]string, 0)
	s.sessions.Range(func(k, v any) bool {
		if key, ok := k.(string); ok {
			keys = append(keys, key)
		}
		return true
	})
	sort.Strings(keys)
	return keys
}

func (s *SessionManager) IsSessionDeleted(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, deleted := s.deletedSessionIDs[sessionID]
	return deleted
}

func (s *SessionManager) DeleteSession(sessionID string) {
	s.mu.RLock()
	s.deletedSessionIDs[sessionID] = struct{}{}
	s.mu.RUnlock()
	s.sessions.Delete(sessionID)
	s.log.Info("Deleted session", "sessionID", sessionID)
}

func (s *SessionManager) Get(given *eth.SyncTesterSession) (*eth.SyncTesterSession, error) {
	id := given.SessionID
	if s.IsSessionDeleted(id) {
		s.log.Warn("Attempted to use deleted session", "sessionID", id)
		return nil, fmt.Errorf("session already deleted: %s", id)
	}
	if given == nil {
		s.log.Warn("No initial session value provided")
		return nil, fmt.Errorf("no initial session value")
	}
	actual, loaded := s.sessions.LoadOrStore(id, given)
	if loaded {
		s.log.Debug("Using existing session", "sessionID", id)
	} else {
		s.log.Info("Initialized new session", "sessionID", id)
	}
	return actual.(*eth.SyncTesterSession), nil
}

func WithSessionWrite[T any](
	mgr *SessionManager,
	ctx context.Context,
	fn func(*eth.SyncTesterSession) (T, error),
) (T, error) {
	var zero T
	given, ok := SyncTesterSessionFromContext(ctx)
	if !ok || given == nil {
		return zero, fmt.Errorf("no session found in context")
	}
	session, err := mgr.Get(given)
	if err != nil {
		return zero, err
	}
	// blocking
	session.Lock()
	defer session.Unlock()
	return fn(session)
}

func WithSessionRead[T any](
	mgr *SessionManager,
	ctx context.Context,
	fn func(*eth.SyncTesterSession) (T, error),
) (T, error) {
	var zero T
	given, ok := SyncTesterSessionFromContext(ctx)
	if !ok || given == nil {
		return zero, fmt.Errorf("no session found in context")
	}
	session, err := mgr.Get(given)
	if err != nil {
		return zero, err
	}
	// blocking
	session.RLock()
	defer session.RUnlock()
	return fn(session)
}
