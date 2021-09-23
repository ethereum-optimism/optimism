package ethash

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type Ethash struct{}

func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (ethash *Ethash) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return nil
}

func (ethash *Ethash) Close() error {
	return nil
}

// Ethash proof-of-work protocol constants.
var (
	FrontierBlockReward           = big.NewInt(5e+18) // Block reward in wei for successfully mining a block
	ByzantiumBlockReward          = big.NewInt(3e+18) // Block reward in wei for successfully mining a block upward from Byzantium
	ConstantinopleBlockReward     = big.NewInt(2e+18) // Block reward in wei for successfully mining a block upward from Constantinople
	maxUncles                     = 2                 // Maximum number of uncles allowed in a single block
	allowedFutureBlockTimeSeconds = int64(15)         // Max seconds from current time allowed for blocks, before they're considered future blocks

	/*
		// calcDifficultyEip3554 is the difficulty adjustment algorithm as specified by EIP 3554.
		// It offsets the bomb a total of 9.7M blocks.
		// Specification EIP-3554: https://eips.ethereum.org/EIPS/eip-3554
		calcDifficultyEip3554 = makeDifficultyCalculator(big.NewInt(9700000))

		// calcDifficultyEip2384 is the difficulty adjustment algorithm as specified by EIP 2384.
		// It offsets the bomb 4M blocks from Constantinople, so in total 9M blocks.
		// Specification EIP-2384: https://eips.ethereum.org/EIPS/eip-2384
		calcDifficultyEip2384 = makeDifficultyCalculator(big.NewInt(9000000))

		// calcDifficultyConstantinople is the difficulty adjustment algorithm for Constantinople.
		// It returns the difficulty that a new block should have when created at time given the
		// parent block's time and difficulty. The calculation uses the Byzantium rules, but with
		// bomb offset 5M.
		// Specification EIP-1234: https://eips.ethereum.org/EIPS/eip-1234
		calcDifficultyConstantinople = makeDifficultyCalculator(big.NewInt(5000000))

		// calcDifficultyByzantium is the difficulty adjustment algorithm. It returns
		// the difficulty that a new block should have when created at time given the
		// parent block's time and difficulty. The calculation uses the Byzantium rules.
		// Specification EIP-649: https://eips.ethereum.org/EIPS/eip-649
		calcDifficultyByzantium = makeDifficultyCalculator(big.NewInt(3000000))*/
)

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	// Skip block reward in catalyst mode
	if config.IsCatalyst(header.Number) {
		return
	}
	// Select the correct block reward based on chain progression
	blockReward := FrontierBlockReward
	if config.IsByzantium(header.Number) {
		blockReward = ByzantiumBlockReward
	}
	if config.IsConstantinople(header.Number) {
		blockReward = ConstantinopleBlockReward
	}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		state.AddBalance(uncle.Coinbase, r)

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	state.AddBalance(header.Coinbase, reward)
}

func (ethash *Ethash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	fmt.Println("consensus finalize")
	// Accumulate any block and uncle rewards and commit the final state root
	accumulateRewards(chain.Config(), state, header, uncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	fmt.Println("new Root", header.Root)
}

func (ethash *Ethash) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	return nil, nil
}

func (ethash *Ethash) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

func (ethash *Ethash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return nil
}

func (ethash *Ethash) SealHash(header *types.Header) (hash common.Hash) {
	return common.Hash{}
}

func (ethash *Ethash) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return nil
}

func (ethash *Ethash) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	return nil, nil
}

func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}
