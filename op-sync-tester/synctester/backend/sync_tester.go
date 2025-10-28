package backend

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/session"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/frontend"
)

type SyncTester struct {
	log log.Logger
	m   metrics.Metricer

	id      sttypes.SyncTesterID
	chainID eth.ChainID

	elReader ReadOnlyELBackend

	sessMgr *session.SessionManager
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
	logger.Info("Initialized sync tester from config", "syncTester", stID)
	return NewSyncTester(logger, m, stID, stCfg.ChainID, elReader), nil
}

func NewSyncTester(logger log.Logger, m metrics.Metricer, stID sttypes.SyncTesterID, chainID eth.ChainID, elReader ReadOnlyELBackend) *SyncTester {
	return &SyncTester{
		log:      logger,
		m:        m,
		id:       stID,
		chainID:  chainID,
		elReader: elReader,
		sessMgr:  session.NewSessionManager(logger),
	}
}

func (s *SyncTester) GetSession(ctx context.Context) (*eth.SyncTesterSession, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.SyncTesterSession, error) {
		logger.Debug("GetSession")
		return session, nil
	})
}

func (s *SyncTester) DeleteSession(ctx context.Context) error {
	_, err := session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (any, error) {
		logger.Debug("DeleteSession")
		return struct{}{}, s.sessMgr.DeleteSession(session.SessionID)
	})
	return err
}

func (s *SyncTester) ResetSession(ctx context.Context) error {
	_, err := session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (any, error) {
		logger.Debug("ResetSession")
		session.ResetSession()
		return struct{}{}, nil
	})
	return err
}

func (s *SyncTester) ListSessions(ctx context.Context) ([]string, error) {
	ids := s.sessMgr.SessionIDs()
	s.log.Debug("ListSessions", "count", len(ids))
	return ids, nil
}

func (s *SyncTester) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) ([]*types.Receipt, error) {
		logger.Debug("GetBlockReceipts", "blockNrOrHash", blockNrOrHash)
		number, isNumber := blockNrOrHash.Number()
		var err error
		var receipts []*types.Receipt
		if !isNumber {
			// hash
			receipts, err = s.elReader.GetBlockReceipts(ctx, blockNrOrHash)
			if err != nil {
				return nil, err
			}
		} else {
			var target uint64
			if target, err = s.checkBlockNumber(number, session, logger); err != nil {
				return nil, err
			}
			receipts, err = s.elReader.GetBlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(target)))
			if err != nil {
				return nil, err
			}
		}
		if len(receipts) == 0 {
			// Should never happen since every block except genesis has at least one deposit tx
			logger.Warn("L2 Block has zero receipts", "blockNrHash", blockNrOrHash)
			return nil, errors.New("no receipts")
		}
		target := receipts[0].BlockNumber.Uint64()
		if target > session.CurrentState.Latest {
			logger.Warn("Requested block is ahead of sync tester state", "requested", target)
			return nil, ethereum.NotFound
		}
		return receipts, nil
	})
}

func (s *SyncTester) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (json.RawMessage, error) {
		logger.Debug("GetBlockByHash", "hash", hash, "fullTx", fullTx)
		var err error
		var raw json.RawMessage
		if raw, err = s.elReader.GetBlockByHashJSON(ctx, hash, fullTx); err != nil {
			return nil, err
		}
		var header HeaderNumberOnly
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, err
		}
		target := header.Number.ToInt().Uint64()
		if target > session.CurrentState.Latest {
			logger.Warn("Requested block is ahead of sync tester state", "requested", target)
			return nil, ethereum.NotFound
		}
		return raw, nil
	})
}

func (s *SyncTester) checkBlockNumber(number rpc.BlockNumber, session *eth.SyncTesterSession, logger log.Logger) (uint64, error) {
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
			logger.Warn("Requested block is ahead of sync tester state", "requested", target)
			return 0, ethereum.NotFound
		}
	}
	return target, nil
}

func (s *SyncTester) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (json.RawMessage, error) {
		logger.Debug("GetBlockByNumber", "number", number, "fullTx", fullTx)
		var err error
		var target uint64
		if target, err = s.checkBlockNumber(number, session, logger); err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if raw, err = s.elReader.GetBlockByNumberJSON(ctx, rpc.BlockNumber(target), fullTx); err != nil {
			return nil, err
		}
		return raw, nil
	})
}

func (s *SyncTester) ChainId(ctx context.Context) (hexutil.Big, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (hexutil.Big, error) {
		logger.Debug("ChainId")
		chainID, err := s.elReader.ChainId(ctx)
		if err != nil {
			return hexutil.Big{}, err
		}
		if chainID.ToInt().Cmp(s.chainID.ToBig()) != 0 {
			logger.Error("ChainId mismatch", "config", s.chainID, "backend", chainID.ToInt())
			return hexutil.Big{}, fmt.Errorf("chainID mismatch: config: %s, backend: %s", s.chainID, chainID.ToInt())
		}
		return hexutil.Big(*s.chainID.ToBig()), nil
	})
}

// GetPayloadV1 only supports V1 payloads.
func (s *SyncTester) GetPayloadV1(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
		logger.Debug("GetPayloadV1", "payloadID", payloadID)
		if !payloadID.Is(engine.PayloadV1) {
			return nil, engine.UnsupportedFork
		}
		return s.getPayload(session, logger, payloadID)
	})
}

// GetPayloadV2 supports V1, V2 payloads.
func (s *SyncTester) GetPayloadV2(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
		logger.Debug("GetPayloadV2", "payloadID", payloadID)
		if !payloadID.Is(engine.PayloadV1, engine.PayloadV2) {
			return nil, engine.UnsupportedFork
		}
		return s.getPayload(session, logger, payloadID)
	})
}

// GetPayloadV3 must be only called when Ecotone activated.
func (s *SyncTester) GetPayloadV3(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
		logger.Debug("GetPayloadV3", "payloadID", payloadID)
		if !payloadID.Is(engine.PayloadV3) {
			return nil, engine.UnsupportedFork
		}
		return s.getPayload(session, logger, payloadID)
	})
}

// GetPayloadV4 must be only called when Isthmus activated.
func (s *SyncTester) GetPayloadV4(ctx context.Context, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ExecutionPayloadEnvelope, error) {
		logger.Debug("GetPayloadV4", "payloadID", payloadID)
		if !payloadID.Is(engine.PayloadV3) {
			return nil, engine.UnsupportedFork
		}
		return s.getPayload(session, logger, payloadID)
	})
}

// getPayload retrieves an execution payload previously initialized by
// ForkchoiceUpdated engine APIs when valid payload attributes were provided.
// Retrieved payloads are deleted from the session after being served to
// emulate one-time consumption by the consensus layer.
func (s *SyncTester) getPayload(session *eth.SyncTesterSession, logger log.Logger, payloadID eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	payloadEnv, ok := session.Payloads[payloadID]
	if !ok {
		return nil, engine.UnknownPayload
	}
	// Clean up payload
	delete(session.Payloads, payloadID)
	logger.Trace("Deleted payload", "payloadID", payloadID)
	return payloadEnv, nil
}

// ForkchoiceUpdatedV1 is called for processing V1 attributes
func (s *SyncTester) ForkchoiceUpdatedV1(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ForkchoiceUpdatedResult, error) {
		logger.Debug("ForkchoiceUpdatedV1", "state", state, "attr", attr)
		return s.forkchoiceUpdated(ctx, session, logger, state, attr, engine.PayloadV1, false, false)
	})
}

// ForkchoiceUpdatedV2 is called for processing V2 attributes
func (s *SyncTester) ForkchoiceUpdatedV2(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ForkchoiceUpdatedResult, error) {
		logger.Debug("ForkchoiceUpdatedV2", "state", state, "attr", attr)
		return s.forkchoiceUpdated(ctx, session, logger, state, attr, engine.PayloadV2, true, false)
	})
}

// ForkchoiceUpdatedV3 must be only called with Ecotone attributes
func (s *SyncTester) ForkchoiceUpdatedV3(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.ForkchoiceUpdatedResult, error) {
		logger.Debug("ForkchoiceUpdatedV3", "state", state, "attr", attr)
		return s.forkchoiceUpdated(ctx, session, logger, state, attr, engine.PayloadV3, true, true)
	})
}

// forkchoiceUpdated processes a forkchoice state update from the consensus
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
func (s *SyncTester) forkchoiceUpdated(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger, state *eth.ForkchoiceState, attr *eth.PayloadAttributes, payloadVersion engine.PayloadVersion,
	isCanyon, isEcotone bool,
) (*eth.ForkchoiceUpdatedResult, error) {
	// Validate attributes shape
	if attr != nil {
		if isEcotone {
			// https://github.com/ethereum/execution-apis/blob/bc5a37ee69a64769bd8d0a2056672361ef5f3839/src/engine/cancun.md#engine_forkchoiceupdatedv3
			// Spec: payloadAttributes matches the PayloadAttributesV3 structure, return -38003: Invalid payload attributes on failure.
			// Ecotone activated Cancun
			if attr.ParentBeaconBlockRoot == nil {
				return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing beacon root"))
			}
			if attr.Withdrawals == nil {
				return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
			}
		} else if isCanyon {
			if attr.ParentBeaconBlockRoot != nil {
				return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("unexpected beacon root"))
			}
			// Canyon activated Shanghai
			if attr.Withdrawals == nil {
				return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
			}
		} else {
			// Bedrock
			if attr.Withdrawals != nil || attr.ParentBeaconBlockRoot != nil {
				return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidParams.With(errors.New("withdrawals and beacon root not supported"))
			}
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
	candLatestNum := candLatest.NumberU64()
	if session.Validated < candLatestNum {
		if !session.IsELSyncActive() {
			// Let CL backfill via newPayload
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		}
		switch session.ELSyncPolicy.ELSyncStatus(candLatestNum) {
		case eth.ExecutionValid:
			// EL Sync complete so advance non canonical chain first
			session.Validated = candLatestNum
			logger.Info("Non canonical chain advanced because of EL Sync", "validated", session.Validated)
			// Equivalent to SetCanonical
			session.UpdateFCULatest(session.Validated)
			logger.Info("Canonical chain advanced because of EL Sync", "latest", session.CurrentState.Latest)
			// Still return SYNCING to mimic the asynchronous EL behavior
			// The EL will eventually return VALID with the identical unsafe target with the next FCU call
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		case eth.ExecutionSyncing:
			logger.Trace("EL Sync on progress", "target", candLatestNum)
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		default:
			logger.Warn("EL Sync failure", "target", candLatestNum)
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, fmt.Errorf("EL Sync failure with target block %d:%s", candLatest.NumberU64(), candLatest.Hash())
		}
	}
	// Equivalent to SetCanonical
	session.UpdateFCULatest(candLatestNum)
	logger.Debug("Updated FCU State", "latest", session.CurrentState.Latest)
	// Simulate db check for finalized head
	if state.FinalizedBlockHash != (common.Hash{}) {
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
		// Spec: MUST return -38002: Invalid forkchoice state error if the payload referenced by forkchoiceState.headBlockHash is VALID and a payload referenced by either forkchoiceState.finalizedBlockHash or forkchoiceState.safeBlockHash does not belong to the chain defined by forkchoiceState.headBlockHash.
		candFinalized, err := s.elReader.GetBlockByHash(ctx, state.FinalizedBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not available"))
		}
		finalizedNum := candFinalized.NumberU64()
		if session.CurrentState.Latest < finalizedNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("finalized block not canonical"))
		}
		// Equivalent to SetFinalized
		session.UpdateFCUFinalized(finalizedNum)
		logger.Debug("Updated FCU State", "finalized", session.CurrentState.Finalized)
	}
	// Simulate db check for safe head
	if state.SafeBlockHash != (common.Hash{}) {
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
		// Spec: MUST return -38002: Invalid forkchoice state error if the payload referenced by forkchoiceState.headBlockHash is VALID and a payload referenced by either forkchoiceState.finalizedBlockHash or forkchoiceState.safeBlockHash does not belong to the chain defined by forkchoiceState.headBlockHash.
		candSafe, err := s.elReader.GetBlockByHash(ctx, state.SafeBlockHash)
		if err != nil {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not available"))
		}
		safeNum := candSafe.NumberU64()
		if session.CurrentState.Latest < safeNum {
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidForkChoiceState.With(errors.New("safe block not canonical"))
		}
		// Equivalent to SetSafe
		session.UpdateFCUSafe(safeNum)
		logger.Debug("Updated FCU State", "safe", session.CurrentState.Safe)
	}
	var id *engine.PayloadID
	if attr != nil {
		// attr is the ingredient for the block built after the head block
		// Query read only EL to fetch block which is desired to be produced from attr
		newBlock, err := s.elReader.GetBlockByNumber(ctx, rpc.BlockNumber(int64(candLatestNum)+1))
		if err != nil {
			// Consider as sync error if read only EL interaction fails because we cannot validate
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionSyncing}, PayloadID: nil}, nil
		}
		// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/holocene/exec-engine.md#eip-1559-parameters-in-block-header
		// Implicitly determine whether holocene is enabled by inspecting extraData from read only EL data
		isHolocene := eip1559.ValidateHoloceneExtraData(newBlock.Header().Extra) == nil
		// Sanity check attr comparing with newBlock
		if err := s.validateAttributesForBlock(attr, newBlock, isHolocene); err != nil {
			// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#specification-1
			// Client software MUST respond to this method call in the following way: {error: {code: -38003, message: "Invalid payload attributes"}} if the payload is deemed VALID and forkchoiceState has been applied successfully, but no build process has been started due to invalid payloadAttributes.
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.InvalidPayloadAttributes.With(err)
		}
		// https://github.com/ethereum-optimism/specs/blob/7b39adb0bea3b0a56d6d3a7d61feef5c33e49b73/specs/protocol/isthmus/exec-engine.md#header-validity-rules
		// Implicitly determine whether isthmus is enabled by inspecting withdrawalsRoot from read only EL data
		isIsthmus := newBlock.WithdrawalsRoot() != nil && len(*newBlock.WithdrawalsRoot()) == 32
		// Initialize payload args for sane payload ID
		// All attr fields already sanity checked
		args := miner.BuildPayloadArgs{
			Parent:       state.HeadBlockHash,
			Timestamp:    uint64(attr.Timestamp),
			FeeRecipient: attr.SuggestedFeeRecipient,
			Random:       common.Hash(attr.PrevRandao),
			BeaconRoot:   attr.ParentBeaconBlockRoot,
			NoTxPool:     attr.NoTxPool,
			Transactions: newBlock.Transactions(),
			GasLimit:     &newBlock.Header().GasLimit,
			Version:      payloadVersion,
		}
		config := &params.ChainConfig{}
		if isCanyon {
			args.Withdrawals = *attr.Withdrawals
			config.CanyonTime = new(uint64)
		}
		if isHolocene {
			args.EIP1559Params = (*attr.EIP1559Params)[:]
		}
		if isIsthmus {
			config.IsthmusTime = new(uint64)
		}
		payloadID := args.Id()
		id = &payloadID
		payloadEnv, err := eth.BlockAsPayloadEnv(newBlock, config)
		if err != nil {
			// The failure is from the EL processing so consider as a server error and make CL retry
			return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, PayloadID: nil}, engine.GenericServerError.With(err)
		}
		// Store payload and payloadID. This will be processed using GetPayload engine API
		logger.Debug("Store payload", "payloadID", payloadID)
		session.Payloads[payloadID] = payloadEnv
	}
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
	if (attr.ParentBeaconBlockRoot == nil) != (h.ParentBeaconRoot == nil) {
		return fmt.Errorf("parentBeaconBlockRoot mismatch: attr=%v, header=%v", attr.ParentBeaconBlockRoot, h.ParentBeaconRoot)
	}
	if h.ParentBeaconRoot != nil && (*attr.ParentBeaconBlockRoot).Cmp(*h.ParentBeaconRoot) != 0 {
		return fmt.Errorf("parentBeaconBlockRoot mismatch: attr=%s, header=%s", *attr.ParentBeaconBlockRoot, *h.ParentBeaconRoot)
	}
	// OP Stack additions
	if len(attr.Transactions) != len(block.Transactions()) {
		return fmt.Errorf("tx count mismatch: attr=%d, block=%d", len(attr.Transactions), len(block.Transactions()))
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
		// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/holocene/exec-engine.md#encoding
		// Spec: At and after Holocene activation, eip1559Parameters in PayloadAttributeV3 must be exactly 8 bytes with the following format
		if attr.EIP1559Params == nil {
			return errors.New("holocene enabled but EIP1559Params nil")
		}
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

// NewPayloadV1 must be only called with Bedrock Payload
func (s *SyncTester) NewPayloadV1(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.PayloadStatusV1, error) {
		logger.Debug("NewPayloadV1", "payload", payload)
		return s.newPayload(ctx, session, logger, payload, nil, nil, nil, false, false)
	})
}

// NewPayloadV2 must be only called with Bedrock, Canyon, Delta Payload
func (s *SyncTester) NewPayloadV2(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.PayloadStatusV1, error) {
		logger.Debug("NewPayloadV2", "payload", payload)
		return s.newPayload(ctx, session, logger, payload, nil, nil, nil, false, false)
	})
}

// NewPayloadV3 must be only called with Ecotone Payload
func (s *SyncTester) NewPayloadV3(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.PayloadStatusV1, error) {
		logger.Debug("NewPayloadV3", "payload", payload, "versionedHashes", versionedHashes, "beaconRoot", beaconRoot)
		return s.newPayload(ctx, session, logger, payload, versionedHashes, beaconRoot, nil, true, false)
	})
}

// NewPayloadV4 must be only called with Isthmus payload
func (s *SyncTester) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (*eth.PayloadStatusV1, error) {
	return session.WithSession(s.sessMgr, ctx, s.log, func(session *eth.SyncTesterSession, logger log.Logger) (*eth.PayloadStatusV1, error) {
		logger.Debug("NewPayloadV4", "payload", payload, "versionedHashes", versionedHashes, "beaconRoot", beaconRoot, "executionRequests", executionRequests)
		return s.newPayload(ctx, session, logger, payload, versionedHashes, beaconRoot, executionRequests, true, true)
	})
}

func (s *SyncTester) validatePayload(logger log.Logger, isCanyon, isIsthmus bool, block *types.Block, payload *eth.ExecutionPayload, beaconRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	// Already have the block locally or advance single block without setting the head
	// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#specification
	// Spec: MUST return {status: INVALID, latestValidHash: null, validationError: errorMessage | null} if the blockHash validation has failed.
	blockHash := block.Hash()
	config := &params.ChainConfig{}
	if isCanyon {
		config.CanyonTime = new(uint64)
	}
	if isIsthmus {
		config.IsthmusTime = new(uint64)
	}
	correctPayload, err := eth.BlockAsPayload(block, config)
	if err != nil {
		// The failure is from the EL processing so consider as a server error and make CL retry
		return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("failed to convert block to payload", err))
	}
	// Sanity check parent beacon block root and block hash by recomputation
	if !isIsthmus {
		// Depopulate withdrawal root field for block hash recomputation
		if payload.WithdrawalsRoot != nil {
			logger.Warn("Isthmus disabled but withdrawal roots included in payload not nil", "root", payload.WithdrawalsRoot)
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
	return nil, nil
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
func (s *SyncTester) newPayload(ctx context.Context, session *eth.SyncTesterSession, logger log.Logger, payload *eth.ExecutionPayload, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes,
	isEcotone, isIsthmus bool,
) (*eth.PayloadStatusV1, error) {
	// Validate request shape, fork required fields
	// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_newpayloadv2
	// Spec: Client software MUST return -32602: Invalid params error if the wrong version of the structure is used in the method call.
	if isEcotone {
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
	} else {
		if payload.ExcessBlobGas != nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("non-nil excessBlobGas pre-cancun"))
		}
		if payload.BlobGasUsed != nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("non-nil blobGasUsed pre-cancun"))
		}
	}
	if isIsthmus {
		if executionRequests == nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil executionRequests post-prague"))
		}
	}
	// OP Stack specific request shape validation
	if isEcotone {
		if len(versionedHashes) != 0 {
			// https://github.com/ethereum-optimism/specs/blob/a773587fca6756f8468164613daa79fcee7bbbe4/specs/protocol/exec-engine.md#engine_newpayloadv3
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(fmt.Errorf("versionedHashes length non-zero: %d", len(versionedHashes)))
		}
	}
	if isIsthmus {
		if payload.WithdrawalsRoot == nil {
			// https://github.com/ethereum-optimism/specs/blob/7b39adb0bea3b0a56d6d3a7d61feef5c33e49b73/specs/protocol/isthmus/exec-engine.md#update-to-executionpayload
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil withdrawalsRoot post-isthmus"))
		}
		if len(executionRequests) != 0 {
			// https://github.com/ethereum-optimism/specs/blob/a773587fca6756f8468164613daa79fcee7bbbe4/specs/protocol/exec-engine.md#engine_newpayloadv4
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(fmt.Errorf("executionRequests must be empty array but got %d", len(executionRequests)))
		}
	}
	// Look up canonical block for relay comparison
	block, err := s.elReader.GetBlockByHash(ctx, payload.BlockHash)
	if err != nil {
		if !errors.Is(err, ethereum.NotFound) {
			// Do not retry when error did not occur because of Not found error
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("failed to fetch block", err))
		}
		// Not found error may be recovered when given payload is near the sequencer tip.
		// Read only EL may not be ready yet. In this case, retry once more after waiting block time (2 seconds)
		logger.Warn("Block not found while validating new payload. Retrying", "number", payload.BlockNumber, "hash", payload.BlockHash)
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			// Handle case when context cancelled while waiting.
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(fmt.Errorf("context done: %w", ctx.Err()))
		}
		block, err = s.elReader.GetBlockByHash(ctx, payload.BlockHash)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("block not found after retry", err))
			}
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.GenericServerError.With(wrapSyncTesterError("failed to fetch block after retry", err))
		}
		// Use block info fetched by retrying
	}
	// https://github.com/ethereum-optimism/specs/blob/972dec7c7c967800513c354b2f8e5b79340de1c3/specs/protocol/derivation.md#building-individual-payload-attributes
	// Implicitly determine whether canyon is enabled by inspecting withdrawals from read only EL data
	isCanyon := block.Withdrawals() != nil
	if isCanyon {
		if payload.Withdrawals == nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
		}
	} else {
		if payload.Withdrawals != nil {
			return &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, engine.InvalidParams.With(errors.New("non-nil withdrawals pre-shanghai"))
		}
	}
	blockHash := block.Hash()
	blockNumber := block.NumberU64()
	// We only attempt to advance non-canonical view of the chain, following the read only EL
	if blockNumber <= session.Validated+1 {
		if status, err := s.validatePayload(logger, isCanyon, isIsthmus, block, payload, beaconRoot); status != nil {
			return status, err
		}
		if blockNumber == session.Validated+1 {
			// Advance single block without setting the head, equivalent to geth InsertBlockWithoutSetHead
			session.Validated += 1
			logger.Debug("Advanced non canonical chain", "validated", session.Validated)
		}
		// https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/paris.md#payload-validation
		// Spec: If validation succeeds, the response MUST contain {status: VALID, latestValidHash: payload.blockHash}
		return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &blockHash}, nil
	} else {
		logger.Debug("Received payload which cannot be used to extend non canonical chain", "current", blockNumber, "validated", session.Validated)
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
