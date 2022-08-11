package testutils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestConsensus wraps the beacon-chain consensus, which is already very minimal in the El,
// but now explicitly without prior eth1 consensus types.
// This is a test-util, and only safe to use if the chain config ensures that the TTD of any block is reached,
// i.e. PoS from the start.
type TestConsensus struct {
	beacon.Beacon  // embedded, most PoS methods work fine as is
}

func (t TestConsensus) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	panic("no pow seal sealing")
}

func (t TestConsensus) SealHash(header *types.Header) common.Hash {
	panic("no pow sealing")
}

func (t TestConsensus) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return nil
}

func (t TestConsensus) Close() error {
	return nil
}

var _ consensus.Engine = (*TestConsensus)(nil)

