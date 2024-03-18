package outputs

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type ProposalTraceProviderCreator func(ctx context.Context, localContext common.Hash, depth types.Depth, agreed contracts.Proposal, claimed contracts.Proposal) (types.TraceProvider, error)

func OutputRootSplitAdapter(topProvider *OutputTraceProvider, creator ProposalTraceProviderCreator) split.ProviderCreator {
	return func(ctx context.Context, depth types.Depth, pre types.Claim, post types.Claim) (types.TraceProvider, error) {
		localContext := CreateLocalContext(pre, post)
		agreed, disputed, err := FetchProposals(ctx, topProvider, pre, post)
		if err != nil {
			return nil, err
		}
		return creator(ctx, localContext, depth, agreed, disputed)
	}
}

func FetchProposals(ctx context.Context, topProvider *OutputTraceProvider, pre types.Claim, post types.Claim) (contracts.Proposal, contracts.Proposal, error) {
	usePrestateBlock := pre == (types.Claim{})
	var agreed contracts.Proposal
	if usePrestateBlock {
		prestateRoot, err := topProvider.AbsolutePreStateCommitment(ctx)
		if err != nil {
			return contracts.Proposal{}, contracts.Proposal{}, fmt.Errorf("failed to retrieve absolute prestate output root: %w", err)
		}
		agreed = contracts.Proposal{
			L2BlockNumber: new(big.Int).SetUint64(topProvider.prestateBlock),
			OutputRoot:    prestateRoot,
		}
	} else {
		preBlockNum, err := topProvider.HonestBlockNumber(ctx, pre.Position)
		if err != nil {
			return contracts.Proposal{}, contracts.Proposal{}, fmt.Errorf("unable to calculate pre-claim block number: %w", err)
		}
		agreed = contracts.Proposal{
			L2BlockNumber: new(big.Int).SetUint64(preBlockNum),
			OutputRoot:    pre.Value,
		}
	}
	postBlockNum, err := topProvider.ClaimedBlockNumber(post.Position)
	if err != nil {
		return contracts.Proposal{}, contracts.Proposal{}, fmt.Errorf("unable to calculate post-claim block number: %w", err)
	}
	claimed := contracts.Proposal{
		L2BlockNumber: new(big.Int).SetUint64(postBlockNum),
		OutputRoot:    post.Value,
	}
	return agreed, claimed, nil
}

func CreateLocalContext(pre types.Claim, post types.Claim) common.Hash {
	return crypto.Keccak256Hash(localContextPreimage(pre, post))
}

func localContextPreimage(pre types.Claim, post types.Claim) []byte {
	encodeClaim := func(c types.Claim) []byte {
		data := make([]byte, 64)
		copy(data[0:32], c.Value.Bytes())
		c.Position.ToGIndex().FillBytes(data[32:])
		return data
	}
	var data []byte
	if pre != (types.Claim{}) {
		data = encodeClaim(pre)
	}
	data = append(data, encodeClaim(post)...)
	return data
}
