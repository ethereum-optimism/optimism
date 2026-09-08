package test

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// RandomL2Block returns a random block whose first transaction is a random pre-Ecotone upgrade
// L1 Info Deposit transaction.
func RandomL2Block(rng *rand.Rand, txCount int, t time.Time) (*types.Block, []*types.Receipt) {
	body := types.Body{}
	l1Block := types.NewBlock(testutils.RandomHeader(rng), &body, nil, trie.NewStackTrie(nil), types.DefaultBlockConfig)
	rollupCfg := rollup.Config{}
	if testutils.RandomBool(rng) {
		t := uint64(0)
		rollupCfg.RegolithTime = &t
	}
	l1InfoTx, err := derive.L1InfoDeposit(&rollupCfg, params.MergedTestChainConfig, eth.SystemConfig{}, 0, eth.BlockToInfo(l1Block), 0)
	if err != nil {
		panic("L1InfoDeposit: " + err.Error())
	}
	if t.IsZero() {
		return testutils.RandomBlockPrependTxs(rng, txCount, testutils.TxFromDeposit(l1InfoTx))
	} else {
		return testutils.RandomBlockPrependTxsWithTime(rng, txCount, uint64(t.Unix()), testutils.TxFromDeposit(l1InfoTx))
	}

}

func RandomL2BlockWithChainId(rng *rand.Rand, txCount int, chainId *big.Int) *types.Block {
	return RandomL2BlockWithChainIdAndTime(rng, txCount, chainId, time.Time{})
}

func RandomL2BlockWithChainIdAndTime(rng *rand.Rand, txCount int, chainId *big.Int, t time.Time) *types.Block {
	signer := types.NewPragueSigner(chainId)
	block, _ := RandomL2Block(rng, 0, t)
	txs := []*types.Transaction{block.Transactions()[0]} // L1 info deposit TX
	for i := 0; i < txCount; i++ {
		txs = append(txs, testutils.RandomTx(rng, big.NewInt(int64(rng.Uint32())), signer))
	}
	return block.WithBody(types.Body{Transactions: txs})
}

// RandomSingularBatch returns a batch of randomly generated transactions, all
// signed for chainID.
//
// It lives here rather than in derive itself so that derive's production build
// stays clear of op-service/testutils, whose closure reaches op-service/apis
// and the libp2p stack. derive's own tests use an equivalent helper in
// batch_test_utils_test.go, which cannot be shared with this package without an
// import cycle.
func RandomSingularBatch(rng *rand.Rand, txCount int, chainID *big.Int) *derive.SingularBatch {
	signer := types.NewPragueSigner(chainID)
	baseFee := big.NewInt(rng.Int63n(300_000_000_000))
	txsEncoded := make([]hexutil.Bytes, 0, txCount)
	// force each tx to have equal chainID
	for i := 0; i < txCount; i++ {
		tx := testutils.RandomTx(rng, baseFee, signer)
		txEncoded, err := tx.MarshalBinary()
		if err != nil {
			panic("tx Marshal binary" + err.Error())
		}
		txsEncoded = append(txsEncoded, txEncoded)
	}
	return &derive.SingularBatch{
		ParentHash:   testutils.RandomHash(rng),
		EpochNum:     rollup.Epoch(1 + rng.Int63n(100_000_000)),
		EpochHash:    testutils.RandomHash(rng),
		Timestamp:    uint64(rng.Int63n(2_000_000_000)),
		Transactions: txsEncoded,
	}
}
