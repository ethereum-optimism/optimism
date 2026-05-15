package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
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
	// SafeDB is required: the verified branch walks safe-head-at-L1 history to compute
	// RequiredL1. Without it, requests within LocalSafeL2 would surface as
	// safedb.ErrNotEnabled from the helper, while requests beyond LocalSafeL2 would
	// short-circuit successfully with Data=nil — so an operator who forgot --safedb.path
	// on a node serving dispute infrastructure would see inconsistent responses.
	// Reject up front instead.
	if reader, ok := s.safeDB.(interface{ Enabled() bool }); ok && !reader.Enabled() {
		return eth.SuperRootAtTimestampResponse{}, errors.New("safedb not enabled: --safedb.path is required to serve superroot_atTimestamp")
	}

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
		// Pre-genesis. op-supernode equivalent: LocalSafeBlockAtTimestamp returns the same
		// error → both VerifiedAt and OptimisticAt fail → chain absent from OptimisticAtTimestamp,
		// Data nil. We return success with the same shape.
		return resp, nil
	}

	// op-supernode bounds the optimistic branch by LocalSafeL2 (LocalSafeBlockAtTimestamp
	// returns ethereum.NotFound when blockNum > localSafe). Beyond LocalSafeL2 — including
	// beyond UnsafeL2 — both VerifiedAt and OptimisticAt return NotFound, so the chain is
	// absent from OptimisticAtTimestamp and Data is nil. Mirror that exactly: do NOT add an
	// optimistic entry derived from L1Origin; consumers (op-challenger SuperNodeTraceProvider
	// at step > 0) read OptimisticAtTimestamp[chainID] without checking Data, so any extra
	// entry here would create a behavioral divergence at non-zero game steps.
	if blockNum > status.LocalSafeL2.Number {
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
		// Match op-supernode chain_container.safeDBAtL2 mapping exactly:
		//   ErrL1AtSafeHeadUnavailable (permanent gap) → ErrHistoryUnavailable → bubbled to RPC
		//   ErrL1AtSafeHeadNotFound (transient lag)    → ethereum.NotFound → chain omitted from
		//                                                OptimisticAtTimestamp, Data nil.
		// Returning a fabricated optimistic entry with RequiredL1 = L1Origin would understate the
		// requirement and could cause op-challenger to include a chain at step > 0 where
		// op-supernode would have triggered InvalidTransition.
		if errors.Is(err, safedb.ErrL1AtSafeHeadUnavailable) {
			return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("L1 at safe head unavailable for L2 %s: %w", ref, err)
		}
		s.log.Debug("L1AtSafeHead transient unavailable, returning Data nil", "err", err, "block", ref)
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
