package backend

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/frontend"
)

type SyncTester struct {
	mu sync.RWMutex

	log log.Logger
	m   metrics.Metricer

	id      sttypes.SyncTesterID
	chainID eth.ChainID
	// elClient
	elClient *ethclient.Client

	sessions map[string]*Session

	// true when the sync tester is disabled and may not serve any new sync tester requests
	disabled bool
}

var _ frontend.SyncBackend = (*SyncTester)(nil)

func SyncTesterFromConfig(logger log.Logger, m metrics.Metricer, stID sttypes.SyncTesterID, stCfg *config.SyncTesterEntry) (*SyncTester, error) {
	logger = logger.New("syncTester", stID, "chain", stCfg.ChainID)
	elClient, err := ethclient.Dial(stCfg.ELRPC.Value.RPC())
	if err != nil {
		return nil, fmt.Errorf("failed to dial EL client: %w", err)
	}
	return &SyncTester{
		log:      logger,
		m:        m,
		id:       stID,
		chainID:  stCfg.ChainID,
		elClient: elClient,
		sessions: make(map[string]*Session),
		disabled: false,
	}, nil
}

func (s *SyncTester) Init(ctx context.Context) (string, error) {
	session, ok := ctx.Value(CtxKeySession).(*Session)
	if !ok || session == nil {
		return "", fmt.Errorf("no session found in context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := session.ID()
	if existing, ok := s.sessions[id]; ok {
		s.log.Info("Using existing session", "session", existing)
	} else {
		s.sessions[id] = session
		s.log.Info("Initialized new session", "session", session)
	}
	return id, nil
}

func (s *SyncTester) ChainID(ctx context.Context) (eth.ChainID, error) {
	return s.chainID, nil
}

func (s *SyncTester) ClearSessions() {
	s.log.Info("Clearing sessions")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*Session)
}
