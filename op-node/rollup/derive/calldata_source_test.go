package derive

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type testTx struct {
	to      *common.Address
	dataLen int
	author  *ecdsa.PrivateKey
	good    bool
	value   int
}

func (tx *testTx) Create(t *testing.T, signer types.Signer, rng *rand.Rand) *types.Transaction {
	out, err := types.SignNewTx(tx.author, signer, &types.DynamicFeeTx{
		ChainID:   signer.ChainID(),
		Nonce:     0,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: big.NewInt(30 * params.GWei),
		Gas:       100_000,
		To:        tx.to,
		Value:     big.NewInt(int64(tx.value)),
		Data:      testutils.RandomData(rng, tx.dataLen),
	})
	require.NoError(t, err)
	return out
}

type calldataTestSetup struct {
	inboxPriv   *ecdsa.PrivateKey
	batcherPriv *ecdsa.PrivateKey
	cfg         *rollup.Config
	signer      types.Signer
}

type calldataTest struct {
	name string
	txs  []testTx
	err  error
}

func (ct *calldataTest) Run(t *testing.T, setup *calldataTestSetup) {
	rng := rand.New(rand.NewSource(1234))
	l1Src := &testutils.MockL1Source{}
	txs := make([]*types.Transaction, len(ct.txs))

	expectedData := make([]eth.Data, 0)

	for i, tx := range ct.txs {
		txs[i] = tx.Create(t, setup.signer, rng)
		if tx.good {
			expectedData = append(expectedData, txs[i].Data())
		}
	}

	info := testutils.RandomL1Info(rng)
	l1Src.ExpectInfoAndTxsByHash(info.Hash(), info, txs, ct.err)

	defer l1Src.Mock.AssertExpectations(t)

	src := NewCalldataSource(testlog.Logger(t, log.LvlError), setup.cfg, l1Src)
	dataIter, err := src.OpenData(context.Background(), info.ID())

	if ct.err != nil {
		require.ErrorIs(t, err, ct.err)
		return
	}
	require.NoError(t, err)

	for {
		dat, err := dataIter.Next(context.Background())
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Equal(t, dat, expectedData[0], "data must match next expected value")
		expectedData = expectedData[1:]
	}
	require.Len(t, expectedData, 0, "all expected data should have been read")
}

func TestCalldataSource_OpenData(t *testing.T) {

	inboxPriv := testutils.RandomKey()
	batcherPriv := testutils.RandomKey()
	cfg := &rollup.Config{
		L1ChainID:          big.NewInt(100),
		BatchInboxAddress:  crypto.PubkeyToAddress(inboxPriv.PublicKey),
		BatchSenderAddress: crypto.PubkeyToAddress(batcherPriv.PublicKey),
	}
	signer := cfg.L1Signer()
	setup := &calldataTestSetup{
		inboxPriv:   inboxPriv,
		batcherPriv: batcherPriv,
		cfg:         cfg,
		signer:      signer,
	}

	altInbox := testutils.RandomAddress(rand.New(rand.NewSource(1234)))
	altAuthor := testutils.RandomKey()

	testCases := []calldataTest{
		{name: "simple", txs: []testTx{{to: &cfg.BatchInboxAddress, dataLen: 1234, author: batcherPriv, good: true}}},
		{name: "other inbox", txs: []testTx{{to: &altInbox, dataLen: 1234, author: batcherPriv, good: false}}},
		{name: "other author", txs: []testTx{{to: &cfg.BatchInboxAddress, dataLen: 1234, author: altAuthor, good: false}}},
		{name: "inbox is author", txs: []testTx{{to: &cfg.BatchInboxAddress, dataLen: 1234, author: inboxPriv, good: false}}},
		{name: "author is inbox", txs: []testTx{{to: &cfg.BatchSenderAddress, dataLen: 1234, author: batcherPriv, good: false}}},
		{name: "unrelated", txs: []testTx{{to: &altInbox, dataLen: 1234, author: altAuthor, good: false}}},
		{name: "contract creation", txs: []testTx{{to: nil, dataLen: 1234, author: batcherPriv, good: false}}},
		{name: "empty tx", txs: []testTx{{to: &cfg.BatchInboxAddress, dataLen: 0, author: batcherPriv, good: true}}},
		{name: "value tx", txs: []testTx{{to: &cfg.BatchInboxAddress, dataLen: 1234, value: 42, author: batcherPriv, good: true}}},
		{name: "empty block", txs: []testTx{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.Run(t, setup)
		})
	}

	t.Run("random combinations", func(t *testing.T) {
		var all []testTx
		for _, tc := range testCases {
			all = append(all, tc.txs...)
		}
		var combiTestCases []calldataTest
		for i := 0; i < 100; i++ {
			txs := append(make([]testTx, 0), all...)
			rng := rand.New(rand.NewSource(42 + int64(i)))
			rng.Shuffle(len(txs), func(i, j int) {
				txs[i], txs[j] = txs[j], txs[i]
			})
			subset := txs[:rng.Intn(len(txs))]
			combiTestCases = append(combiTestCases, calldataTest{
				name: fmt.Sprintf("combi_%d_subset_%d", i, len(subset)),
				txs:  subset,
			})
		}

		for _, testCase := range combiTestCases {
			t.Run(testCase.name, func(t *testing.T) {
				testCase.Run(t, setup)
			})
		}
	})
}
