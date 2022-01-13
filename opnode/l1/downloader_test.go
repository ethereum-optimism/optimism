package l1

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
)

type retryReceipt struct {
	tries   int
	receipt *types.Receipt
}
type mockDownloaderSource struct {
	block    *types.Block
	receipts map[common.Hash]*retryReceipt
}

func (m *mockDownloaderSource) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if m.block == nil {
		return nil, errors.New("no block here")
	}
	return m.block, nil
}

func (m *mockDownloaderSource) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	ret, ok := m.receipts[txHash]
	if !ok {
		return nil, errors.New("no receipt here")
	}
	ret.tries -= 1
	if ret.tries < 0 {
		return ret.receipt, nil
	}
	return nil, fmt.Errorf("receipt takes %d more tries to download", ret.tries)
}

func RandomL1Block(txCount int) (*types.Block, []*types.Receipt) {
	// insecure but reproducible secret key RNG for test txs
	rng := rand.New(rand.NewSource(123))
	key, _ := ecdsa.GenerateKey(crypto.S256(), rng)
	signer := types.NewLondonSigner(big.NewInt(1234))

	var txs []*types.Transaction
	for i := 0; i < txCount; i++ {
		tx, err := types.SignNewTx(key, signer, &types.LegacyTx{
			Nonce:    uint64(i),
			GasPrice: big.NewInt(7),
			Gas:      21000,
			To:       &common.Address{}, // burn, send to zero address
			Value:    big.NewInt(1337),
			Data:     nil,
		})
		if err != nil {
			panic(fmt.Errorf("failed to sign tx %d: %v", i, err))
		}
		txs = append(txs, tx)
	}

	receipts := make([]*types.Receipt, 0, len(txs))
	for i, tx := range txs {
		h := tx.Hash()
		receipts = append(receipts, &types.Receipt{
			Type:   tx.Type(),
			Status: types.ReceiptStatusSuccessful,
			// not part of the receipt, but extra optional info, which we use for testing
			TxHash:           h,
			TransactionIndex: uint(i),
		})
	}
	hasher := trie.NewStackTrie(nil)

	var parent common.Hash
	rng.Read(parent[:])
	var state common.Hash
	rng.Read(state[:])
	block := types.NewBlock(&types.Header{
		ParentHash:  parent,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        state,
		TxHash:      types.DeriveSha(types.Transactions(txs), hasher),
		ReceiptHash: types.DeriveSha(types.Receipts(receipts), hasher),
		Bloom:       types.Bloom{},
		Difficulty:  nil,
		Number:      big.NewInt(123),
	}, txs, nil, receipts, trie.NewStackTrie(nil))
	return block, receipts
}

func TestDownloader_Fetch(t *testing.T) {
	checks := func(t *testing.T, bl *types.Block, rs []*types.Receipt, workers int, change func(src *mockDownloaderSource) bool) {
		receiptRetries := make(map[common.Hash]*retryReceipt)
		for _, r := range rs {
			receiptRetries[r.TxHash] = &retryReceipt{tries: 0, receipt: r}
		}
		src := &mockDownloaderSource{block: bl, receipts: receiptRetries}
		expectOK := true
		if change != nil {
			expectOK = change(src)
		}
		dl := NewDownloader(src)
		dl.AddReceiptWorkers(workers)
		block, receipts, err := dl.Fetch(context.Background(), eth.BlockID{Hash: bl.Hash(), Number: bl.NumberU64()})
		if expectOK {
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, bl, block, "block retrieved")
			assert.Equal(t, rs, receipts, "receipts received")
		} else {
			if err == nil {
				t.Fatal("expected error, but got none")
			}
		}
		dl.Close()
		_, _, err = dl.Fetch(context.Background(), eth.BlockID{Hash: common.Hash{0xff}, Number: 42})
		assert.Error(t, err)
		assert.ErrorIs(t, err, DownloadClosedErr)
	}

	testWithWorkers := func(workers int) {
		t.Run(fmt.Sprintf("%d workers", workers), func(t *testing.T) {

			t.Run("empty block", func(t *testing.T) {
				bl, rs := RandomL1Block(0)
				checks(t, bl, rs, 2, nil)
			})

			t.Run("missing block", func(t *testing.T) {
				bl, rs := RandomL1Block(0)
				checks(t, bl, rs, 2, func(src *mockDownloaderSource) bool {
					src.block = nil
					return false
				})
			})

			t.Run("single tx block", func(t *testing.T) {
				bl, rs := RandomL1Block(1)
				checks(t, bl, rs, 2, nil)
			})

			t.Run("single tx single retry block", func(t *testing.T) {
				bl, rs := RandomL1Block(1)
				checks(t, bl, rs, 2, func(src *mockDownloaderSource) bool {
					for _, r := range src.receipts {
						r.tries = 1
					}
					return true
				})
			})

			t.Run("single tx too many retries block", func(t *testing.T) {
				bl, rs := RandomL1Block(1)
				checks(t, bl, rs, 2, func(src *mockDownloaderSource) bool {
					for _, r := range src.receipts {
						r.tries = 9999
					}
					return false
				})
			})

			t.Run("two tx retry block", func(t *testing.T) {
				bl, rs := RandomL1Block(2)
				checks(t, bl, rs, 2, func(src *mockDownloaderSource) bool {
					for _, r := range src.receipts {
						r.tries = 1
					}
					return true
				})
			})

			t.Run("few tx no retry block", func(t *testing.T) {
				bl, rs := RandomL1Block(10)
				checks(t, bl, rs, 2, nil)
			})

			t.Run("many tx no retry block", func(t *testing.T) {
				bl, rs := RandomL1Block(100)
				checks(t, bl, rs, 2, nil)
			})

			t.Run("many tx random retries block", func(t *testing.T) {
				bl, rs := RandomL1Block(100)
				rng := rand.New(rand.NewSource(123))
				checks(t, bl, rs, 2, func(src *mockDownloaderSource) bool {
					for _, r := range src.receipts {
						r.tries = rng.Intn(4)
					}
					return true
				})
			})
		})
	}

	testWithWorkers(1)
	testWithWorkers(2)
	testWithWorkers(5)
}
