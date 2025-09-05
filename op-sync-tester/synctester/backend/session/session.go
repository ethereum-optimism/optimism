package session

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
	deletedSessionIDs map[string]struct{}

	log log.Logger

	sessions sync.Map // map[string]*eth.SyncTesterSession
}

type sessionKeyType struct{}

var ctxKeySession = sessionKeyType{}

// WithSyncTesterSession returns a new context with the given Session.
func WithSyncTesterSession(ctx context.Context, s *eth.SyncTesterSession) context.Context {
	return context.WithValue(ctx, ctxKeySession, s)
}

// SyncTesterSessionFromContext retrieves the Session from the context, if present.
func SyncTesterSessionFromContext(ctx context.Context) (*eth.SyncTesterSession, bool) {
	s, ok := ctx.Value(ctxKeySession).(*eth.SyncTesterSession)
	return s, ok
}

func NewSessionManager(logger log.Logger) *SessionManager {
	return &SessionManager{log: logger, deletedSessionIDs: make(map[string]struct{})}
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

func (s *SessionManager) isSessionDeleted(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, deleted := s.deletedSessionIDs[sessionID]
	return deleted
}

func (s *SessionManager) DeleteSession(sessionID string) {
	s.mu.Lock()
	s.deletedSessionIDs[sessionID] = struct{}{}
	s.mu.Unlock()
	if v, ok := s.sessions.Load(sessionID); ok {
		if sess, ok := v.(*eth.SyncTesterSession); ok {
			sess.Close()
		}
	}
	s.sessions.Delete(sessionID)
	s.log.Info("Deleted session", "sessionID", sessionID)
}

func (s *SessionManager) get(given *eth.SyncTesterSession) (*eth.SyncTesterSession, error) {
	if given == nil {
		s.log.Warn("No initial session value provided")
		return nil, fmt.Errorf("no initial session value")
	}
	id := given.SessionID
	if s.isSessionDeleted(id) {
		s.log.Warn("Attempted to use deleted session", "sessionID", id)
		return nil, fmt.Errorf("session already deleted: %s", id)
	}
	actual, loaded := s.sessions.LoadOrStore(id, given)
	if loaded {
		s.log.Debug("Using existing session", "sessionID", id)
	} else {
		s.log.Info("Initialized new session", "sessionID", id)
	}
	// close the race window
	if s.isSessionDeleted(id) {
		s.sessions.Delete(id)
		s.log.Warn("Session was deleted concurrently", "sessionID", id)
		return nil, fmt.Errorf("session already deleted: %s", id)
	}
	sess, ok := actual.(*eth.SyncTesterSession)
	if !ok {
		return nil, fmt.Errorf("invalid session type for %s", id)
	}
	return sess, nil
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
	session, err := mgr.get(given)
	if err != nil {
		return zero, err
	}
	// blocking
	session.Lock()
	defer session.Unlock()
	if session.IsClosed() {
		return zero, fmt.Errorf("session already deleted: %s", session.SessionID)
	}
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
	session, err := mgr.get(given)
	if err != nil {
		return zero, err
	}
	// blocking
	session.RLock()
	defer session.RUnlock()
	if session.IsClosed() {
		return zero, fmt.Errorf("session already deleted: %s", session.SessionID)
	}
	return fn(session)
}
