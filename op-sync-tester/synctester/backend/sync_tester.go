package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	id      sttypes.SyncTesterID
	chainID eth.ChainID

	elReader ReadOnlyELBackend

	sessions map[string]*Session
}

// HeaderNumberOnly is a lightweight header type that only contains the
// block number field. It is useful in contexts where the full Ethereum
// block header is not needed, and only the block number is required.
type HeaderNumberOnly struct {
	Number *hexutil.Big `json:"number"  gencodec:"required"`
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
	elReader := NewELReader(elClient)
	return NewSyncTester(logger, m, stID, stCfg.ChainID, elReader), nil
}

func NewSyncTester(logger log.Logger, m metrics.Metricer, stID sttypes.SyncTesterID, chainID eth.ChainID, elReader ReadOnlyELBackend) *SyncTester {
	return &SyncTester{
		log:      logger,
		m:        m,
		id:       stID,
		chainID:  chainID,
		elReader: elReader,
		sessions: make(map[string]*Session),
	}
}

func (s *SyncTester) storeSession(session *Session) {
	s.sessions[session.SessionID] = session
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
		return existing, nil
	} else {
		s.storeSession(session)
		s.log.Info("Initialized new session", "session", session)
		return session, nil
	}
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
	number, isNumber := blockNrOrHash.Number()
	var receipts []*types.Receipt
	if !isNumber {
		// hash
		receipts, err = s.elReader.GetBlockReceipts(ctx, blockNrOrHash)
		if err != nil {
			return nil, err
		}
	} else {
		var target uint64
		if target, err = s.checkBlockNumber(number, session); err != nil {
			return nil, err
		}
		receipts, err = s.elReader.GetBlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(target)))
		if err != nil {
			return nil, err
		}
	}
	if len(receipts) == 0 {
		// Should never happen since every block except genesis has at least one deposit tx
		return nil, ErrNoReceipts
	}
	if receipts[0].BlockNumber.Uint64() > session.CurrentState.Latest {
		return nil, ethereum.NotFound
	}
	return receipts, nil
}

func (s *SyncTester) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if raw, err = s.elReader.GetBlockByHashJSON(ctx, hash, fullTx); err != nil {
		return nil, err
	}
	var header HeaderNumberOnly
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	if header.Number.ToInt().Uint64() > session.CurrentState.Latest {
		return nil, ethereum.NotFound
	}
	return raw, nil
}

func (s *SyncTester) checkBlockNumber(number rpc.BlockNumber, session *Session) (uint64, error) {
	var target uint64
	switch number {
	case rpc.LatestBlockNumber:
		target = session.CurrentState.Latest
	case rpc.SafeBlockNumber:
		target = session.CurrentState.Safe
	case rpc.FinalizedBlockNumber:
		target = session.CurrentState.Finalized
	case rpc.PendingBlockNumber, rpc.EarliestBlockNumber:
		// pending, earliest block label not supported
		return 0, ethereum.NotFound
	default:
		if number.Int64() < 0 {
			// safety guard for overflow
			return 0, ethereum.NotFound
		}
		target = uint64(number.Int64())
		// Short circuit for numeric request beyond sync tester canonical head
		if target > session.CurrentState.Latest {
			return 0, ethereum.NotFound
		}
	}
	return target, nil
}

func (s *SyncTester) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	var target uint64
	if target, err = s.checkBlockNumber(number, session); err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if raw, err = s.elReader.GetBlockByNumberJSON(ctx, rpc.BlockNumber(target), fullTx); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *SyncTester) ChainId(ctx context.Context) (hexutil.Big, error) {
	if _, err := s.fetchSession(ctx); err != nil {
		return hexutil.Big{}, err
	}
	chainID, err := s.elReader.ChainId(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	if chainID.ToInt().Cmp(s.chainID.ToBig()) != 0 {
		return hexutil.Big{}, fmt.Errorf("chainID mismatch: config: %s, backend: %s", s.chainID, chainID.ToInt())
	}
	return hexutil.Big(*s.chainID.ToBig()), nil
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
