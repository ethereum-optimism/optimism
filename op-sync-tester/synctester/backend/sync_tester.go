package backend

import (
	"bytes"
	"context"
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
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

// GetPayloadV3 is functionally identical to GetPayloadV4.
func (s *SyncTester) GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	if !payloadID.Is(engine.PayloadV3) {
		return nil, engine.UnsupportedFork
	}
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	return s.getPayload(session, payloadID)
}

// GetPayloadV4 retrieves an execution payload previously initialized by
// ForkchoiceUpdated engine APIs when valid payload attributes were provided.
// Retrieved payloads are deleted from the session after being served to
// emulate one-time consumption by the consensus layer.
func (s *SyncTester) GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	if !payloadID.Is(engine.PayloadV3) {
		return nil, engine.UnsupportedFork
	}
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	return s.getPayload(session, payloadID)
}

func (s *SyncTester) getPayload(session *Session, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
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

// ForkchoiceUpdatedV3 processes a forkchoice state update from the consensus
// layer, validates the request against the current execution layer state, and
// optionally initializes a new payload build process if payload attributes are
// provided. When payload attributes are not nil and validation succeeds, the
// derived payload is stored for later retrieval via GetPayload.
//
// Return values:
//   - {status: VALID, latestValidHash: headBlockHash, payloadId: id} when the
//     forkchoice state is applied successfully and payload attributes were
//     provided and validated.
//   - {status: VALID, latestValidHash: headBlockHash, payloadId: null} when the
//     forkchoice state is applied successfully but no payload build was started
//     (attr was not provided).
//   - {status: INVALID, latestValidHash: null, validationError: err} when payload
//     attributes are malformed or finalized/safe blocks are not canonical.
//   - {status: SYNCING} when the head block is unknown or not yet validated, or
//     when block data cannot be retrieved from the execution layer.
func (s *SyncTester) ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	// Validate attributes shape
	if attr != nil {
		// https://github.com/ethereum/execution-apis/blob/bc5a37ee69a64769bd8d0a2056672361ef5f3839/src/engine/cancun.md#engine_forkchoiceupdatedv3
		// Spec: payloadAttributes matches the PayloadAttributesV3 structure, return -38003: Invalid payload attributes on failure.
		if attr.Withdrawals == nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
		}
		if attr.ParentBeaconBlockRoot == nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing beacon root"))
		}
	}
	// Simulate head block hash check
	candLatest, err := s.elReader.GetBlockByHash(ctx, state.HeadBlockHash)
	// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
	// Spec: {payloadStatus: {status: SYNCING, latestValidHash: null, validationError: null}, payloadId: null} if forkchoiceState.headBlockHash references an unknown payload or a payload that can't be validated because requisite data for the validation is missing
	if err != nil {
		// Consider as sync error if read only EL interaction fails because we cannot validate
		return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
	}
	if candLatest.NumberU64() > session.Validated {
		// Let CL backfill via newPayload
		return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
	}
	// Simulate db check for finalized head
	var finalizedNum uint64
	if state.FinalizedBlockHash != (common.Hash{}) {
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
		// Spec: MUST return -38002: Invalid forkchoice state error if the payload referenced by forkchoiceState.headBlockHash is VALID and a payload referenced by either forkchoiceState.finalizedBlockHash or forkchoiceState.safeBlockHash does not belong to the chain defined by forkchoiceState.headBlockHash.
		candFinalized, err := s.elReader.GetBlockByHash(ctx, state.FinalizedBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not available"))
		}
		finalizedNum = candFinalized.NumberU64()
		if session.CurrentState.Latest < finalizedNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not canonical"))
		}
	}
	// Simulate db check for safe head
	var safeNum uint64
	if state.SafeBlockHash != (common.Hash{}) {
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
		// Spec: MUST return -38002: Invalid forkchoice state error if the payload referenced by forkchoiceState.headBlockHash is VALID and a payload referenced by either forkchoiceState.finalizedBlockHash or forkchoiceState.safeBlockHash does not belong to the chain defined by forkchoiceState.headBlockHash.
		candSafe, err := s.elReader.GetBlockByHash(ctx, state.SafeBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not available"))
		}
		safeNum = candSafe.NumberU64()
		if session.CurrentState.Latest < safeNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not canonical"))
		}
	}
	var id *engine.PayloadID
	if attr != nil {
		// attr is the ingredient for the block built after the head block
		candNum := int64(candLatest.NumberU64())
		// Query read only EL to fetch block which is desired to be produced from attr
		newBlock, err := s.elReader.GetBlockByNumber(ctx, rpc.BlockNumber(candNum+1))
		if err != nil {
			// Consider as sync error if read only EL interaction fails because we cannot validate
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		}
		// Implictly determine whether holocene is enabled by inspecting extraData from read only EL data
		isHolocene := eip1559.ValidateHoloceneExtraData(newBlock.Header().Extra) == nil
		// Sanity check attr comparing with newBlock
		if err := s.validateAttributesForBlock(attr, newBlock, isHolocene); err != nil {
			// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
			// Client software MUST respond to this method call in the following way: {error: {code: -38003, message: "Invalid payload attributes"}} if the payload is deemed VALID and forkchoiceState has been applied successfully, but no build process has been started due to invalid payloadAttributes.
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(err)
		}
		// Initialize payload args for sane payload ID
		// All attr fields already sanity checked
		args := miner.BuildPayloadArgs{
			Parent:       state.HeadBlockHash,
			Timestamp:    uint64(attr.Timestamp),
			FeeRecipient: attr.SuggestedFeeRecipient,
			Random:       common.Hash(attr.PrevRandao),
			Withdrawals:  *attr.Withdrawals,
			BeaconRoot:   attr.ParentBeaconBlockRoot,
			NoTxPool:     attr.NoTxPool,
			Transactions: newBlock.Transactions(),
			GasLimit:     &newBlock.Header().GasLimit,
			Version:      engine.PayloadV3,
		}
		if isHolocene {
			args.EIP1559Params = (*attr.EIP1559Params)[:]
		}
		payloadID := args.Id()
		id = &payloadID
		// Activate Canyon and Isthmus
		config := &params.ChainConfig{CanyonTime: new(uint64), IsthmusTime: new(uint64)}
		payloadEnv, err := eth.BlockAsPayloadEnv(newBlock, config)
		if err != nil {
			// The failure is from the EL processing so consider as a server error and make CL retry
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.GenericServerError.With(err)
		}
		// Store payload and payloadID. This will be processed using GetPayload engine API
		session.Payloads[payloadID] = payloadEnv
	}
	session.UpdateFCUState(candLatest.NumberU64(), safeNum, finalizedNum)
	s.storeSession(session)
	// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
	// Spec: Client software MUST respond to this method call in the following way: {payloadStatus: {status: VALID, latestValidHash: forkchoiceState.headBlockHash, validationError: null}, payloadId: buildProcessId} if the payload is deemed VALID and the build process has begun
	return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &state.HeadBlockHash}, PayloadID: id}, nil
}

// validateAttributesForBlock verifies that a given block matches the expected
// execution payload attributes. It ensures consistency between the provided
// PayloadAttributes and the block header and body.
//
// OP Stack additions:
//   - Transaction count and raw transaction bytes must match exactly.
//   - NoTxPool must be always true, since sync tester only runs in verifier mode.
//   - Gas limit must match.
//   - If Holocene is active: Extra data must be exactly 9 bytes, the version byte must equal to 0,
//     the remaining 8 bytes must match the EIP-1559 parameters.
//
// Returns an error if any mismatch or invalid condition is found, otherwise nil.
func (s *SyncTester) validateAttributesForBlock(attr *eth.PayloadAttributes, block *types.Block, isHolocene bool) error {
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
	// OP Stack additions
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
		// Sync Tester only supports verifier sync
		return errors.New("txpool cannot be enabled yet")
	}
	if *attr.GasLimit != eth.Uint64Quantity(h.GasLimit) {
		return fmt.Errorf("gaslimit mismatch: attr=%d, header=%d", *attr.GasLimit, h.GasLimit)
	}
	if isHolocene {
		if err := eip1559.ValidateHolocene1559Params((*attr.EIP1559Params)[:]); err != nil {
			return fmt.Errorf("invalid eip1559Params: %w", err)
		}
		denominator, elasticity := eip1559.DecodeHolocene1559Params((*attr.EIP1559Params)[:])
		if denominator == 0 && elasticity == 0 {
			// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/holocene/exec-engine.md#payload-attributes-processing
			// Spec: The denominator and elasticity values within this extraData must correspond to those in eip1559Parameters, unless both are 0. When both are 0, the prior EIP-1559 constants must be used to populate extraData instead.
			// Cannot validate since EL will fall back to prior eip1559 constants
			return nil
		}
		if !bytes.Equal(block.Extra()[1:], (*attr.EIP1559Params)[:]) {
			return fmt.Errorf("eip1559Params mismatch: %s != 0x%s", *attr.EIP1559Params, hex.EncodeToString(block.Extra()[1:]))
		}
	} else {
		// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/holocene/exec-engine.md#payload-attributes-processing
		// Spec: Prior to Holocene activation, eip1559Parameters in PayloadAttributesV3 must be null and is otherwise considered invalid.
		if attr.EIP1559Params != nil {
			return fmt.Errorf("holocene disabled but EIP1559Params not nil. eip1559Params: %s", attr.EIP1559Params)
		}
	}
	return nil
}

func (s *SyncTester) NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

func (s *SyncTester) NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return nil, nil
}

// NewPayloadV3 must be only called with Ecotone Payload
func (s *SyncTester) NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	return s.newPayload(ctx, session, payload, versionedHashes, beaconRoot, nil, true, false)
}

// NewPayloadV4 must be only called with Isthmus payload
func (s *SyncTester) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error) {
	session, err := s.fetchSession(ctx)
	if err != nil {
		return nil, err
	}
	return s.newPayload(ctx, session, payload, versionedHashes, beaconRoot, executionRequests, true, true)
}

// newPayload validates and processes a new execution payload according to the
// Engine API rules to simulate consensus-layer to execution-layer interactions
// without advancing canonical chain state.
//
// The method enforces mandatory post-fork fields, including withdrawals, excessBlobGas,
// blobGasUsed, versionedHashes, beaconRoot, executionRequests, and withdrawalsRoot,
// returning an InvalidParams error if any are missing or improperly shaped.
//
// Return values:
//   - {status: VALID, latestValidHash: payload.blockHash} if validation succeeds.
//   - {status: INVALID, latestValidHash: null, validationError: err} on mismatch
//     or malformed payloads.
//   - {status: SYNCING} when the block cannot be executed because its parent is missing.
//   - Errors surfaced as engine.InvalidParams or engine.GenericServerError to
//     trigger appropriate consensus-layer retries.
func (s *SyncTester) newPayload(ctx context.Context, session *Session, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes,
	isEcotone, isIsthmus bool,
) (*eth.PayloadStatusV1, error) {
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
	if versionedHashes == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil versionedHashes post-cancun"))
	}
	if beaconRoot == nil {
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil beaconRoot post-cancun"))
	}
	if isIsthmus {
		if executionRequests == nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil executionRequests post-prague"))
		}
	}
	// OP Stack specific request shape validation
	if isEcotone {
		if payload.WithdrawalsRoot == nil {
			// https://github.com/ethereum-optimism/specs/blob/a773587fca6756f8468164613daa79fcee7bbbe4/specs/protocol/exec-engine.md#engine_newpayloadv3
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil withdrawalsRoot post-isthmus"))
		}
		if len(versionedHashes) != 0 {
			// https://github.com/ethereum-optimism/specs/blob/a773587fca6756f8468164613daa79fcee7bbbe4/specs/protocol/exec-engine.md#engine_newpayloadv3
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(fmt.Errorf("versionedHashes length non-zero: %d", len(versionedHashes)))
		}
	}
	if isIsthmus {
		if len(executionRequests) != 0 {
			// https://github.com/ethereum-optimism/specs/blob/a773587fca6756f8468164613daa79fcee7bbbe4/specs/protocol/exec-engine.md#engine_newpayloadv4
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(fmt.Errorf("executionRequests must be empty array but got %d", len(executionRequests)))
		}
	}
	// Look up canonical block for relay comparison
	block, err := s.elReader.GetBlockByHash(ctx, payload.BlockHash)
	if err != nil {
		// Do not know block hash included in payload is correct or not. Consider as a server error and make CL retry
		if errors.Is(err, ethereum.NotFound) {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("block not found", err))
		}
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("failed to fetch block", err))
	}
	blockHash := block.Hash()
	// We only attempt to advance non-canonical view of the chain, following the read only EL
	if block.NumberU64() <= session.Validated+1 {
		// Already have the block locally or advance single block without setting the head
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#specification
		// Spec: MUST return {status: INVALID, latestValidHash: null, validationError: errorMessage | null} if the blockHash validation has failed.
		config := &params.ChainConfig{CanyonTime: new(uint64)}
		if isIsthmus {
			config.IsthmusTime = new(uint64)
		}
		correctPayload, err := eth.BlockAsPayload(block, config)
		if err != nil {
			// The failure is from the EL processing so consider as a server error and make CL retry
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("failed convert block to payload: %w", err))
		}
		// Sanity check parent beacon block root and block hash by recomputation
		if !isIsthmus {
			// Depopulate withdrawal root field for block hash recomputation
			if payload.WithdrawalsRoot != nil {
				s.log.Warn("Isthmus disabled but withdrawal roots included in payload not nil", "root", payload.WithdrawalsRoot)
			}
			payload.WithdrawalsRoot = nil
		}
		// Check given payload matches the payload derived using the read only EL block
		if err := correctPayload.CheckEqual(payload); err != nil {
			// Consider as block hash validation error when payload mismatch
			return s.newPayloadInvalid(fmt.Errorf("payload check mismatch: %w", err), nil), nil
		}
		execEnvelope := eth.ExecutionPayloadEnvelope{ParentBeaconBlockRoot: beaconRoot, ExecutionPayload: payload}
		actual, ok := execEnvelope.CheckBlockHash()
		if blockHash != payload.BlockHash || !ok {
			return s.newPayloadInvalid(fmt.Errorf("block hash check from execution envelope failed. %s != %s", blockHash, actual), nil), nil
		}
		if block.NumberU64() == session.Validated+1 {
			// Advance single block without setting the head, equivalent to geth InsertBlockWithoutSetHead
			session.Validated += 1
			s.storeSession(session)
		}
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#payload-validation
		// Spec: If validation succeeds, the response MUST contain {status: VALID, latestValidHash: payload.blockHash}
		return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &blockHash}, nil
	}
	// Block not available so mark as syncing
	return &eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, nil
}

func wrapSyncTesterError(msg string, err error) error {
	if err == nil {
		return fmt.Errorf("sync tester: %s", msg)
	}
	return fmt.Errorf("sync tester: %s: %w", msg, err)
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
