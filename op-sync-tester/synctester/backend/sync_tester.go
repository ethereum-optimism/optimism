package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-sync-tester/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
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
	if !payloadID.Is(engine.PayloadV3) {
		return nil, engine.UnsupportedFork
	}
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	payloadEnv, ok := session.Payloads[payloadID]
	if !ok {
		return nil, engine.UnknownPayload
	}
	// Clean up payload
	delete(session.Payloads, payloadID)
	s.storeSession(session)
	return payloadEnv, nil
}

func (s *SyncTester) ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (s *SyncTester) ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return nil, nil
}

func (s *SyncTester) ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	if attr != nil {
		if attr.Withdrawals == nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
		}
		if attr.ParentBeaconBlockRoot == nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing beacon root"))
		}
	}
	candLatest, err := s.elReader.GetBlockByHash(ctx, state.HeadBlockHash)
	if err != nil {
		return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
	}
	if candLatest.NumberU64() > session.Validated {
		// Let CL backfill via newPayload
		return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
	}
	var safeNum uint64
	if state.SafeBlockHash != (common.Hash{}) {
		candSafe, err := s.elReader.GetBlockByHash(ctx, state.SafeBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not available"))
		}
		safeNum = candSafe.NumberU64()
		if session.CurrentState.Latest < safeNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not canonical"))
		}
	}
	var finalizedNum uint64
	if state.FinalizedBlockHash != (common.Hash{}) {
		candFinalized, err := s.elReader.GetBlockByHash(ctx, state.FinalizedBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not available"))
		}
		finalizedNum = candFinalized.NumberU64()
		if session.CurrentState.Latest < finalizedNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not canonical"))
		}
	}
	var id *engine.PayloadID
	if attr != nil {
		// attr is the ingredient for the block built after the head block
		candNum := int64(candLatest.NumberU64())
		newBlock, err := s.elReader.GetBlockByNumber(ctx, rpc.BlockNumber(candNum+1))
		if err != nil {
			// Wait for backend EL to catch up
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		}
		// sanity check attr comparing with targetBlock
		if err := s.attrCheck(attr, newBlock, true); err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(err)
		}
		args := miner.BuildPayloadArgs{
			Parent:        state.HeadBlockHash,
			Timestamp:     uint64(attr.Timestamp),
			FeeRecipient:  attr.SuggestedFeeRecipient,
			Random:        common.Hash(attr.PrevRandao),
			Withdrawals:   *attr.Withdrawals,
			BeaconRoot:    attr.ParentBeaconBlockRoot,
			NoTxPool:      attr.NoTxPool,
			Transactions:  newBlock.Transactions(),
			GasLimit:      &newBlock.Header().GasLimit,
			Version:       engine.PayloadV3,
			EIP1559Params: (*attr.EIP1559Params)[:],
		}
		payloadID := args.Id()
		id = &payloadID
		config := &params.ChainConfig{ShanghaiTime: new(uint64), IsthmusTime: new(uint64)}
		payloadEnv, err := eth.BlockAsPayloadEnv(newBlock, config)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &state.HeadBlockHash}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(err)
		}
		session.Payloads[payloadID] = payloadEnv
	}
	// Update FCU State
	session.CurrentState.Latest = candLatest.NumberU64()
	session.CurrentState.Safe = safeNum
	session.CurrentState.Finalized = finalizedNum
	s.storeSession(session)
	return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &state.HeadBlockHash}, PayloadID: id}, nil
}

func (s *SyncTester) attrCheck(attr *eth.PayloadAttributes, block *types.Block, isHolocene bool) error {
	h := block.Header()
	if h.Time != uint64(attr.Timestamp) {
		return fmt.Errorf("timestamp mismatch: header=%d, attr=%d", h.Time, attr.Timestamp)
	}
	if h.MixDigest != common.Hash(attr.PrevRandao) {
		return fmt.Errorf("prevRandao mismatch: header=%s, attr=%s", h.MixDigest, attr.PrevRandao)
	}
	if h.Coinbase != attr.SuggestedFeeRecipient {
		return fmt.Errorf("coinbase mismatch: header=%s, attr=%s", h.Coinbase, attr.SuggestedFeeRecipient)
	}
	if attr.Withdrawals != nil && len(*attr.Withdrawals) != 0 {
		return errors.New("withdrawals must be nil or empty")
	}
	if attr.ParentBeaconBlockRoot == nil || h.ParentBeaconRoot == nil {
		return fmt.Errorf("parentBeaconBlockRoot must be provided")
	}
	if (*attr.ParentBeaconBlockRoot).Cmp(*h.ParentBeaconRoot) != 0 {
		return fmt.Errorf("parentBeaconBlockRoot mismatch: attr=%s, header=%s", *attr.ParentBeaconBlockRoot, *h.ParentBeaconRoot)
	}
	// Optimism additions
	if len(attr.Transactions) != len(block.Transactions()) {
		return fmt.Errorf("tx count mismatch: attr=%d, header=%d", len(attr.Transactions), len(block.Transactions()))
	}
	for idx := range len(attr.Transactions) {
		blockTx := block.Transactions()[idx]
		blockTxRaw, err := blockTx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal block tx: %w", err)
		}
		if !bytes.Equal([]byte(attr.Transactions[idx]), blockTxRaw) {
			return fmt.Errorf("tx mismatch: tx=%s, idx=%d", attr.Transactions[idx], idx)
		}
	}
	if !attr.NoTxPool {
		// Verifier is only supported
		return errors.New("txpool cannot be enabled yet")
	}
	if *attr.GasLimit != eth.Uint64Quantity(h.GasLimit) {
		return fmt.Errorf("gaslimit mismatch: attr=%d, header=%d", *attr.GasLimit, h.GasLimit)
	}
	if isHolocene && !bytes.Equal(block.Extra()[1:], (*attr.EIP1559Params)[:]) {
		// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/holocene/exec-engine.md
		return fmt.Errorf("invalid eip1559Params params: %s", *attr.EIP1559Params)
	}
	return nil
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

func (s *SyncTester) newPayloadInvalid(err error, latestValid *types.Header) *eth.PayloadStatusV1 {
	var currentHash *common.Hash
	if latestValid != nil {
		if latestValid.Difficulty.BitLen() != 0 {
			// Set latest valid hash to 0x0 if parent is PoW block
			currentHash = &common.Hash{}
		} else {
			// Otherwise set latest valid hash to parent hash
			h := latestValid.Hash()
			currentHash = &h
		}
	}
	errorMsg := err.Error()
	return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid, LatestValidHash: currentHash, ValidationError: &errorMsg}
}

func (s *SyncTester) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	// NewPayloadV4 is used starting from Isthmus HF. Activate necessary configs in field to populate execution payload fields
	// https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/isthmus/exec-engine.md#engine_newpayloadv4-api
	// Validate request shape, fork required fields
	// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_newpayloadv2
	// Spec: Client software MUST return -32602: Invalid params error if the wrong version of the structure is used in the method call.
	if payload.Withdrawals == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
	}
	if payload.ExcessBlobGas == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil excessBlobGas post-cancun"))
	}
	if payload.BlobGasUsed == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil blobGasUsed post-cancun"))
	}
	if payload.WithdrawalsRoot == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil withdrawalsRoot post-isthmus"))
	}
	if versionedHashes == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil versionedHashes post-cancun"))
	}
	if len(versionedHashes) != 0 {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("op stack does not use blob txs"))
	}
	if beaconRoot == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil beaconRoot post-cancun"))
	}
	if executionRequests == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("executionRequests must be an empty array for isthmus/prague"))
	}
	if len(executionRequests) != 0 {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("executionRequests should be empty for isthmus/prague"))
	}
	// Look up canonical block for relay comparison
	block, err := s.elReader.GetBlockByHash(ctx, payload.BlockHash)
	if err != nil {
		// Do not know block hash included in payload is correct or not. Consider as a server error and make CL retry
		if errors.Is(err, ethereum.NotFound) {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(fmt.Errorf("sync tester: block not found: %w", err))
		}
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(fmt.Errorf("sync tester: failed to fetch block: %w", err))
	}
	hash := block.Hash()
	// We only attempt to advance non-canonical view of the chain, following the read only EL
	if block.NumberU64() <= session.Validated+1 {
		// Already have the block locally or advance single block without setting the head
		// Spec: Client software MUST return {status: INVALID, latestValidHash: null, validationError: errorMessage | null} if the blockHash validation has failed.
		// Validate beacon root by recomputing hash
		execEnvelope := eth.ExecutionPayloadEnvelope{ParentBeaconBlockRoot: beaconRoot, ExecutionPayload: payload}
		_, ok := execEnvelope.CheckBlockHash()
		if block.Hash() != payload.BlockHash || !ok {
			return s.newPayloadInvalid(errors.New("sync tester: hash mismatch"), nil), nil
		}
		// Activate Shanghai and Isthmus
		config := &params.ChainConfig{ShanghaiTime: new(uint64), IsthmusTime: new(uint64)}
		correctPayload, err := eth.BlockAsPayload(block, config)
		if err != nil {
			// The failure is from the EL processing so consider as a server error
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(fmt.Errorf("sync tester: failed convert block to payload: %w", err))
		}
		// Check that the payload matches the real one, and if not consider as blockHash validation failure
		if err := correctPayload.Compare(payload); err != nil {
			return s.newPayloadInvalid(fmt.Errorf("sync tester: payload mismatch: %w", err), nil), nil
		}
		if block.NumberU64() == session.Validated+1 {
			// Advance single block without setting the head
			session.Validated += 1
			s.storeSession(session)
		}
		// If validation succeeds, the response MUST contain {status: VALID, latestValidHash: payload.blockHash}
		return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &hash}, nil
	}
	// Block not available so mark as syncing
	return &eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, nil
}
