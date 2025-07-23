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
	"github.com/holiman/uint256"

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
}

var _ frontend.SyncBackend = (*SyncTester)(nil)
var _ frontend.EngineBackend = (*SyncTester)(nil)
var _ frontend.EthBackend = (*SyncTester)(nil)

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
	}, nil
}

func (s *SyncTester) fetchSession(ctx context.Context) (*Session, error) {
	session, ok := SessionFromContext(ctx)
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

func (s *SyncTester) GetSession(ctx context.Context) error {
	_, err := s.fetchSession(ctx)
	if err != nil {
		return ErrNoSession
	}
	return nil
}

func (s *SyncTester) DeleteSession(ctx context.Context) error {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return ErrNoSession
	}
	delete(s.sessions, session.SessionID)
	return nil
}

func (s *SyncTester) ListSessions(ctx context.Context) ([]string, error) {
	panic("not implemented")
}

func (s *SyncTester) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}

	receipts, err := s.elClient.BlockReceipts(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	if len(receipts) == 0 {
		return nil, ErrNoReceipts
	}

	if receipts[0].BlockNumber.Uint64() > session.Latest {
		return nil, ethereum.NotFound
	}

	return receipts, nil
}

func (s *SyncTester) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}

	block, err := s.elClient.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if block.NumberU64() > session.Latest {
		return nil, ethereum.NotFound
	}

	return block, nil
}

func (s *SyncTester) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}

	if number.Uint64() > session.Latest {
		return nil, ethereum.NotFound
	}

	return s.elClient.BlockByNumber(ctx, number)
}

func (s *SyncTester) ChainId(ctx context.Context) (eth.ChainID, error) {
	_, err := s.fetchSession(ctx)
	if err != nil {
		return eth.ChainID(uint256.Int{}), err
	}

	return s.chainID, nil
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
