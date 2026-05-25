package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// superrootAPI serves `superroot_atTimestamp` against op-node's single rollup chain,
// the non-interop counterpart to op-supernode's superroot activity. Dispute infra
// (op-challenger, op-dispute-mon) can point `--supernode-rpc` at op-node and get a
// SuperRootAtTimestampResponse with `len(ChainIDs) == 1`.
type superrootAPI struct {
	cfg    *rollup.Config
	client l2EthClient
	dr     driverClient
	safeDB SafeDBReader
	log    log.Logger
}

func NewSuperrootAPI(cfg *rollup.Config, client l2EthClient, dr driverClient, safeDB SafeDBReader, log log.Logger) *superrootAPI {
	return &superrootAPI{cfg: cfg, client: client, dr: dr, safeDB: safeDB, log: log}
}

// AtTimestamp serves wire method "superroot_atTimestamp" (namespace "superroot",
// matching op-supernode so existing clients work unchanged).
func (s *superrootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	return s.atTimestamp(ctx, uint64(timestamp))
}

// atTimestamp is unexported so go-ethereum's reflection-based registration doesn't
// expose it as a separate RPC method.
func (s *superrootAPI) atTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	chainID := eth.ChainIDFromBig(s.cfg.L2ChainID)

	blockNum, err := s.cfg.TargetBlockNumber(timestamp)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("target block number for timestamp %d: %w", timestamp, err)
	}

	// BlockRefWithStatus returns ref and the SyncStatus captured at the same instant,
	// so the LocalSafeL2 bound check below is consistent with the ref we just resolved.
	ref, status, err := s.dr.BlockRefWithStatus(ctx, blockNum)
	if err != nil {
		// ethereum.NotFound means the block isn't known yet (beyond unsafe head). Mirror
		// op-supernode: omit the chain and return sync timestamps from a separate
		// SyncStatus call. All other errors (context deadlines, EL transport, etc.)
		// fail the RPC — silently emitting an empty response would mask real failures.
		if !errors.Is(err, ethereum.NotFound) {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("blockRefWithStatus@%d: %w", blockNum, err)
		}
		status, ssErr := s.dr.SyncStatus(ctx)
		if ssErr != nil {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("syncStatus after blockRef NotFound@%d: %w", blockNum, ssErr)
		}
		return responseSkeleton(status, chainID), nil
	}

	resp := responseSkeleton(status, chainID)

	// op-supernode omits the chain when blockNum > LocalSafeL2: LocalSafeBlockAtTimestamp
	// returns ethereum.NotFound, so both VerifiedAt and OptimisticAt fail. Consumers
	// (op-challenger SuperNodeTraceProvider at step > 0) read OptimisticAtTimestamp
	// without checking Data, so any synthetic entry here would diverge from supernode.
	if blockNum > status.LocalSafeL2.Number {
		return resp, nil
	}

	output, err := s.client.OutputV0AtBlock(ctx, ref.Hash)
	if err != nil {
		// op-supernode parity: omit the chain on NotFound, propagate other errors.
		if errors.Is(err, ethereum.NotFound) {
			return resp, nil
		}
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("outputV0AtBlock@%s: %w", ref, err)
	}
	outputRoot := eth.OutputRoot(output)

	// L2 genesis is trivially safe at L1 block 0 (not cfg.Genesis.L1, since contracts may
	// pre-date it). Otherwise ask SafeDB for the earliest L1 at which ref became safe.
	var requiredL1 eth.BlockID
	if ref.ID() == s.cfg.Genesis.L2 {
		requiredL1 = eth.BlockID{Number: 0}
	} else {
		requiredL1, _, err = s.safeDB.L1AtSafeHead(ctx, ref.Number)
	}
	if err != nil {
		// ErrL1AtSafeHeadNotFound is transient (SafeDB lag); op-supernode maps it to
		// ethereum.NotFound and omits the chain. All other errors (ErrL1AtSafeHeadUnavailable,
		// ErrNotEnabled from a disabled DB, network/IO failures) propagate.
		if errors.Is(err, safedb.ErrL1AtSafeHeadNotFound) {
			s.log.Debug("L1AtSafeHead transient, omitting chain", "err", err, "block", ref)
			return resp, nil
		}
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("l1AtSafeHead@%s: %w", ref, err)
	}

	resp.OptimisticAtTimestamp[chainID] = eth.OutputWithRequiredL1{
		Output:     output,
		OutputRoot: outputRoot,
		RequiredL1: requiredL1,
	}

	superV1 := eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{
		ChainID: chainID,
		Output:  outputRoot,
	})
	resp.Data = &eth.SuperRootResponseData{
		VerifiedRequiredL1: requiredL1,
		Super:              superV1,
		SuperRoot:          eth.SuperRoot(superV1),
	}
	return resp, nil
}

func responseSkeleton(status *eth.SyncStatus, chainID eth.ChainID) eth.SuperRootAtTimestampResponse {
	return eth.SuperRootAtTimestampResponse{
		CurrentL1:                 status.CurrentL1.ID(),
		CurrentSafeTimestamp:      status.SafeL2.Time,
		CurrentLocalSafeTimestamp: status.LocalSafeL2.Time,
		CurrentFinalizedTimestamp: status.FinalizedL2.Time,
		ChainIDs:                  []eth.ChainID{chainID},
		OptimisticAtTimestamp:     map[eth.ChainID]eth.OutputWithRequiredL1{},
	}
}
