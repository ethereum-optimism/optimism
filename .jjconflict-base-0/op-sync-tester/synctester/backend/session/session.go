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
	sync.Mutex
	sessions          map[string]*eth.SyncTesterSession
	deletedSessionIDs map[string]struct{}

	log log.Logger
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
	return &SessionManager{log: logger,
		sessions:          make(map[string]*eth.SyncTesterSession),
		deletedSessionIDs: make(map[string]struct{}),
	}
}

func (s *SessionManager) SessionIDs() []string {
	s.Lock()
	defer s.Unlock()
	keys := make([]string, 0)
	for key := range s.sessions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *SessionManager) DeleteSession(sessionID string) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.sessions[sessionID]; !ok {
		return fmt.Errorf("attempted to delete non-existent session: %s", sessionID)
	}
	s.deletedSessionIDs[sessionID] = struct{}{}
	delete(s.sessions, sessionID)
	s.log.Info("Deleted session", "sessionID", sessionID)
	return nil
}

func (s *SessionManager) get(given *eth.SyncTesterSession) (*eth.SyncTesterSession, error) {
	if given == nil {
		s.log.Warn("No initial session value provided")
		return nil, fmt.Errorf("no initial session value")
	}
	id := given.SessionID
	s.Lock()
	defer s.Unlock()
	if _, ok := s.deletedSessionIDs[id]; ok {
		s.log.Warn("Attempted to use deleted session", "sessionID", id)
		return nil, fmt.Errorf("session already deleted: %s", id)
	}
	var sess *eth.SyncTesterSession
	sess, ok := s.sessions[id]
	if ok {
		s.log.Trace("Using existing session", "sessionID", id)
	} else {
		s.sessions[id] = given
		sess = given
		s.log.Info("Initialized new session", "sessionID", id)
	}
	return sess, nil
}

func WithSession[T any](
	mgr *SessionManager,
	ctx context.Context,
	logger log.Logger,
	fn func(*eth.SyncTesterSession, log.Logger) (T, error),
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
	// Bind session ID and starting fcu state
	logger = logger.With("id", session.SessionID, "start_fcu", session.CurrentState)
	return fn(session, logger)
}
