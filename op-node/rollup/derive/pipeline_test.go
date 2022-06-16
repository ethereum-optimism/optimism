package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"testing"
)

var _ Engine = (*testutils.MockEngine)(nil)

var _ L1Fetcher = (*testutils.MockL1Source)(nil)

func TestDerivationPipeline(t *testing.T) {
	logger := testlog.Logger(t, log.LvlError)
	cfg := &rollup.Config{
		Genesis:                rollup.Genesis{},
		BlockTime:              2,
		MaxSequencerDrift:      10,
		SeqWindowSize:          32,
		L1ChainID:              big.NewInt(900),
		L2ChainID:              big.NewInt(901),
		P2PSequencerAddress:    common.Address{0x0a},
		FeeRecipientAddress:    common.Address{0x0b},
		BatchInboxAddress:      common.Address{0x0c},
		BatchSenderAddress:     common.Address{0x0d},
		DepositContractAddress: common.Address{0x0e},
	}
	eng := &testutils.MockEngine{}
	l1Src := &testutils.MockL1Source{}

	pipeline := NewDerivationPipeline(logger, cfg, l1Src, eng)
	t.Log("created pipeline", pipeline)
	// TODO
	//require.NoError(t, pipeline.Reset(context.Background(), ))

	// TODO: test cases, similar to old driver state-tests:
	// - Simple extensions of L1
	// - Reorg of L1
	// - Simple extensions with multiple steps of stutter
}
