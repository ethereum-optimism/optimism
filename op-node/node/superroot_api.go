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

// superrootAPI serves `superroot_atTimestamp` for op-node's single rollup chain,
// the non-interop counterpart to op-supernode's superroot activity. Lets dispute
// infra point `--supernode-rpc` at op-node and get `len(ChainIDs) == 1`.
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

// AtTimestamp serves the "superroot_atTimestamp" wire method.
func (s *superrootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	return s.atTimestamp(ctx, uint64(timestamp))
}

// atTimestamp is unexported so go-ethereum doesn't auto-register it as an RPC method.
func (s *superrootAPI) atTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	chainID := eth.ChainIDFromBig(s.cfg.L2ChainID)

	blockNum, err := s.cfg.TargetBlockNumber(timestamp)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("target block number for timestamp %d: %w", timestamp, err)
	}

	// Use BlockRefWithStatus so the LocalSafeL2 bound below sees a status consistent
	// with the resolved ref.
	ref, status, err := s.dr.BlockRefWithStatus(ctx, blockNum)
	if err != nil {
		// NotFound (beyond unsafe head): omit chain, populate sync fields from a
		// fresh SyncStatus call. Other errors propagate.
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

	// op-supernode omits the chain past LocalSafeL2; any synthetic entry here would
	// diverge for consumers that read OptimisticAtTimestamp without checking Data.
	if blockNum > status.LocalSafeL2.Number {
		return resp, nil
	}

	output, err := s.client.OutputV0AtBlock(ctx, ref.Hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return resp, nil
		}
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("outputV0AtBlock@%s: %w", ref, err)
	}
	outputRoot := eth.OutputRoot(output)

	// L2 genesis is trivially safe at L1 block 0 — not cfg.Genesis.L1, since
	// contracts may pre-date it.
	var requiredL1 eth.BlockID
	if ref.ID() == s.cfg.Genesis.L2 {
		requiredL1 = eth.BlockID{Number: 0}
	} else {
		requiredL1, _, err = s.safeDB.L1AtSafeHead(ctx, ref.Number)
	}
	if err != nil {
		// Only transient SafeDB lag (ErrL1AtSafeHeadNotFound) omits the chain;
		// permanent gaps and disabled-DB / IO errors propagate.
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
