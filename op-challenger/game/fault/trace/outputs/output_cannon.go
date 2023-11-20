package outputs

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func NewOutputCannonTraceAccessor(ctx context.Context, logger log.Logger, rollupRpc string, gameDepth uint64, prestateBlock uint64, poststateBlock uint64) (*trace.Accessor, error) {
	topDepth := gameDepth / 2 // TODO(client-pod#43): Load this from the contract
	outputProvider, err := NewTraceProvider(ctx, logger, rollupRpc, topDepth, prestateBlock, poststateBlock)
	if err != nil {
		return nil, err
	}

	cannonCreator := func(ctx context.Context, localContext common.Hash, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error) {
		// TODO(client-pod#43): Actually create the cannon trace provider for the trace between the given claims.
		return nil, errors.New("not implemented")
	}

	selector := split.NewSplitProviderSelector(outputProvider, int(topDepth), OutputRootSplitAdapter(outputProvider, cannonCreator))
	return trace.NewAccessor(selector), nil
}
