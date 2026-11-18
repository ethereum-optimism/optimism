package test

import (
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// RandomL2Payload returns a random L2 execution payload with the given timestamp. Its first
// transaction is a random pre-Ecotone upgrade L1 Info Deposit transaction, followed by txCount
// random user transactions signed for chainID. BlockHash commits to the payload, so a caller
// that changes a header field must recompute it.
func RandomL2Payload(rng *rand.Rand, txCount int, chainID *big.Int, timestamp uint64) *eth.ExecutionPayload {
	rollupCfg := rollup.Config{}
	if testutils.RandomBool(rng) {
		t := uint64(0)
		rollupCfg.RegolithTime = &t
	}
	l1Info := eth.HeaderBlockInfo(testutils.RandomHeader(rng))
	l1InfoTx, err := derive.L1InfoDepositBytes(&rollupCfg, params.MergedTestChainConfig, eth.SystemConfig{}, 0, l1Info, 0)
	if err != nil {
		panic("L1InfoDepositBytes: " + err.Error())
	}

	baseFee := big.NewInt(rng.Int63n(300_000_000_000))
	signer := types.NewPragueSigner(chainID)
	txs := make([]eth.Data, 0, 1+txCount)
	txs = append(txs, l1InfoTx)
	for i := 0; i < txCount; i++ {
		tx, err := testutils.RandomTx(rng, baseFee, signer).MarshalBinary()
		if err != nil {
			panic("tx MarshalBinary: " + err.Error())
		}
		txs = append(txs, tx)
	}

	payload := &eth.ExecutionPayload{
		ParentHash:    testutils.RandomHash(rng),
		FeeRecipient:  testutils.RandomAddress(rng),
		StateRoot:     eth.Bytes32(testutils.RandomHash(rng)),
		BlockNumber:   eth.Uint64Quantity(1 + rng.Int63n(100_000_000)),
		Timestamp:     eth.Uint64Quantity(timestamp),
		ExtraData:     testutils.RandomData(rng, rng.Intn(33)),
		BaseFeePerGas: eth.Uint256Quantity(*uint256.MustFromBig(baseFee)),
		Transactions:  txs,
	}
	payload.BlockHash, _ = (&eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}).CheckBlockHash()
	return payload
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
