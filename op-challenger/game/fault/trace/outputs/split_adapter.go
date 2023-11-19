package outputs

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type ProposalTraceProviderCreator func(ctx context.Context, localContext uint64, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error)

func OutputRootSplitAdapter(topProvider *OutputTraceProvider, creator ProposalTraceProviderCreator) split.ProviderCreator {
	return func(ctx context.Context, pre types.Claim, post types.Claim) (types.TraceProvider, error) {
		usePrestateBlock := pre == (types.Claim{})
		preContractIndex := pre.ContractIndex
		if usePrestateBlock {
			preContractIndex = math.MaxUint32
		}
		localContext := uint64(preContractIndex)<<32 + uint64(post.ContractIndex)
		var agreed contracts.Proposal
		if usePrestateBlock {
			prestateRoot, err := topProvider.AbsolutePreStateCommitment(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve absolute prestate output root: %w", err)
			}
			agreed = contracts.Proposal{
				L2BlockNumber: new(big.Int).SetUint64(topProvider.prestateBlock),
				OutputRoot:    prestateRoot,
			}
		} else {
			preBlockNum, err := topProvider.BlockNumber(pre.Position)
			if err != nil {
				return nil, fmt.Errorf("unable to calculate pre-claim block number: %w", err)
			}
			agreed = contracts.Proposal{
				L2BlockNumber: new(big.Int).SetUint64(preBlockNum),
				OutputRoot:    pre.Value,
			}
		}
		postBlockNum, err := topProvider.BlockNumber(post.Position)
		if err != nil {
			return nil, fmt.Errorf("unable to calculate post-claim block number: %w", err)
		}
		claimed := contracts.Proposal{
			L2BlockNumber: new(big.Int).SetUint64(postBlockNum),
			OutputRoot:    post.Value,
		}

		return creator(ctx, localContext, agreed, claimed)
	}
}
