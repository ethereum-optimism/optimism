package node

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// superrootAPI serves the `superroot_atTimestamp` JSON-RPC method against op-node's
// single rollup chain. It is the non-interop counterpart to op-supernode's superroot
// activity: dispute infrastructure (op-challenger, op-dispute-mon) can point
// `--supernode-rpc` at op-node and receive an eth.SuperRootAtTimestampResponse with
// `len(ChainIDs) == 1`.
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

// AtTimestamp serves wire method "superroot_atTimestamp". The namespace is "superroot",
// matching op-supernode exactly so existing clients (sources.SuperNodeClient) work unchanged.
func (s *superrootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	return s.atTimestamp(ctx, uint64(timestamp))
}

// atTimestamp is the internal implementation. Lowercase first letter so it is not
// exposed as an RPC method by go-ethereum's reflection-based registration.
func (s *superrootAPI) atTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	status, err := s.dr.SyncStatus(ctx)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("syncStatus: %w", err)
	}

	chainID := eth.ChainIDFromBig(s.cfg.L2ChainID)
	resp := eth.SuperRootAtTimestampResponse{
		CurrentL1:                 status.CurrentL1.ID(),
		CurrentSafeTimestamp:      status.SafeL2.Time,
		CurrentLocalSafeTimestamp: status.LocalSafeL2.Time,
		CurrentFinalizedTimestamp: status.FinalizedL2.Time,
		ChainIDs:                  []eth.ChainID{chainID},
		OptimisticAtTimestamp:     map[eth.ChainID]eth.OutputWithRequiredL1{},
	}

	blockNum, err := s.cfg.TargetBlockNumber(timestamp)
	if err != nil {
		// Pre-genesis: empty optimistic, nil Data, sync fields populated.
		return resp, nil
	}

	if blockNum > status.UnsafeL2.Number {
		// Beyond local head: empty optimistic, nil Data, sync fields populated.
		return resp, nil
	}

	ref, _, err := s.dr.BlockRefWithStatus(ctx, blockNum)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("blockRefWithStatus@%d: %w", blockNum, err)
	}
	output, err := s.client.OutputV0AtBlock(ctx, ref.Hash)
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("outputV0AtBlock@%s: %w", ref, err)
	}
	outputRoot := eth.OutputRoot(output)

	// Optimistic-only branch: block exists but is not yet local-safe. RequiredL1 is the
	// L2 block's L1 origin (the L1 height required to derive this L2 block).
	if blockNum > status.LocalSafeL2.Number {
		resp.OptimisticAtTimestamp[chainID] = eth.OutputWithRequiredL1{
			Output:     output,
			OutputRoot: outputRoot,
			RequiredL1: ref.L1Origin,
		}
		return resp, nil
	}

	// Verified branch: ask safeDB for the earliest L1 at which ref became safe.
	// Mirror op-supernode's genesis special case: L2 genesis is trivially safe at L1
	// block 0 (not cfg.Genesis.L1, since contracts may pre-date it).
	var requiredL1 eth.BlockID
	if ref.ID() == s.cfg.Genesis.L2 {
		requiredL1 = eth.BlockID{Number: 0}
	} else {
		requiredL1, _, err = s.safeDB.L1AtSafeHead(ctx, ref.Number)
	}
	if err != nil {
		// Mirror op-supernode behavior: when safeDB cannot resolve (transient or permanent),
		// degrade to optimistic-only with L1Origin as RequiredL1 and Data nil. Consumers
		// (op-challenger, op-dispute-mon) treat Data nil as "not yet verified."
		s.log.Debug("L1AtSafeHead unavailable, returning optimistic-only", "err", err, "block", ref)
		resp.OptimisticAtTimestamp[chainID] = eth.OutputWithRequiredL1{
			Output:     output,
			OutputRoot: outputRoot,
			RequiredL1: ref.L1Origin,
		}
		return resp, nil
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
