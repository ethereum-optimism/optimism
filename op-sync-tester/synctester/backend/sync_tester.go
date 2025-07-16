package backend

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/frontend"
)

var (
	ErrNoSession  = errors.New("no session")
	ErrNoReceipts = errors.New("no receipts")
)

type SyncTester struct {
	mu sync.RWMutex

	log log.Logger
	m   metrics.Metricer

	id       sttypes.SyncTesterID
	chainID  eth.ChainID
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

func (s *SyncTester) fetchSession(ctx context.Context) (*Session, error) {
	session, ok := ctx.Value(CtxKeySession).(*Session)
	if !ok || session == nil {
		return nil, fmt.Errorf("no session found in context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.sessions[session.SessionID]; ok {
		s.log.Info("Using existing session", "session", existing)
	} else {
		s.sessions[session.SessionID] = session
		s.log.Info("Initialized new session", "session", session)
	}
	return session, nil
}

func (s *SyncTester) Init(ctx context.Context) (string, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return "", err
	}
	return session.SessionID, nil
}

func (s *SyncTester) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, ErrNoSession
	}
	res, err := s.elClient.BlockReceipts(ctx, blockNrOrHash)
	if err != nil {
		return nil, ethereum.NotFound
	}
	if len(res) == 0 {
		return nil, ErrNoReceipts // should never happen because of deposit tx
	}
	if res[0].BlockNumber.Uint64() >= session.Head {
		return nil, ethereum.NotFound
	}
	return res, nil
}

func (s *SyncTester) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, ErrNoSession
	}
	res, err := s.elClient.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, ethereum.NotFound
	}
	if res.Number.Uint64() >= session.Head {
		return nil, ethereum.NotFound
	}
	return res, nil
}

func (s *SyncTester) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, ErrNoSession
	}
	if number.Uint64() >= session.Head {
		return nil, ethereum.NotFound
	}
	return s.elClient.HeaderByNumber(ctx, number)
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

func (s *SyncTester) GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayload, error) {
	return nil, nil
}

func (s *SyncTester) GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (s *SyncTester) GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (s *SyncTester) GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (s *SyncTester) ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (s *SyncTester) ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (s *SyncTester) ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (s *SyncTester) NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

func (s *SyncTester) NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

func (s *SyncTester) NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

func (s *SyncTester) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error) {
	return nil, nil
}
