package txmgr

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

type queueFunc func(factory TxFactory[int], receiptCh chan TxReceipt[int], q *Queue[int]) (bool, error)

func sendQueueFunc(factory TxFactory[int], receiptCh chan TxReceipt[int], q *Queue[int]) (bool, error) {
	err := q.Send(factory, receiptCh)
	return err == nil, err
}

func trySendQueueFunc(factory TxFactory[int], receiptCh chan TxReceipt[int], q *Queue[int]) (bool, error) {
	return q.TrySend(factory, receiptCh)
}

type queueCall struct {
	call    queueFunc // queue call (either Send or TrySend, use function helpers above)
	queued  bool      // true if the send was queued
	callErr bool      // true if the call should return an error immediately
	txErr   bool      // true if the tx send should return an error
}

type testTx struct {
	factoryErr error // error to return from the factory for this tx
	sendErr    bool  // error to return from send for this tx
}

type testCase struct {
	name   string        // name of the test
	max    uint64        // max concurrency of the queue
	calls  []queueCall   // calls to the queue
	txs    []testTx      // txs to generate from the factory (and potentially error in send)
	nonces []uint64      // expected sent tx nonces after all calls are made
	total  time.Duration // approx. total time it should take to complete all queue calls
}

type mockBackendWithNonce struct {
	mockBackend
}

func newMockBackendWithNonce(g *gasPricer) *mockBackendWithNonce {
	return &mockBackendWithNonce{
		mockBackend: mockBackend{
			g:        g,
			minedTxs: make(map[common.Hash]minedTxInfo),
		},
	}
}

func (b *mockBackendWithNonce) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return uint64(len(b.minedTxs)), nil
}

func TestSend(t *testing.T) {
	testCases := []testCase{
		{
			name: "success",
			max:  5,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: true},
			},
			txs: []testTx{
				{},
				{},
			},
			nonces: []uint64{0, 1},
			total:  1 * time.Second,
		},
		{
			name: "no limit",
			max:  0,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: true},
			},
			txs: []testTx{
				{},
				{},
			},
			nonces: []uint64{0, 1},
			total:  1 * time.Second,
		},
		{
			name: "single threaded",
			max:  1,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: false},
				{call: trySendQueueFunc, queued: false},
			},
			txs: []testTx{
				{},
			},
			nonces: []uint64{0},
			total:  1 * time.Second,
		},
		{
			name: "single threaded blocking",
			max:  1,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: false},
				{call: sendQueueFunc, queued: true},
				{call: sendQueueFunc, queued: true},
			},
			txs: []testTx{
				{},
				{},
				{},
			},
			nonces: []uint64{0, 1, 2},
			total:  3 * time.Second,
		},
		{
			name: "dual threaded blocking",
			max:  2,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, queued: false},
				{call: sendQueueFunc, queued: true},
				{call: sendQueueFunc, queued: true},
				{call: sendQueueFunc, queued: true},
			},
			txs: []testTx{
				{},
				{},
				{},
				{},
				{},
			},
			nonces: []uint64{0, 1, 2, 3, 4},
			total:  3 * time.Second,
		},
		{
			name: "factory returns error",
			max:  5,
			calls: []queueCall{
				{call: trySendQueueFunc, queued: true},
				{call: trySendQueueFunc, callErr: true},
				{call: trySendQueueFunc, queued: true},
			},
			txs: []testTx{
				{},
				{factoryErr: io.EOF},
				{},
			},
			nonces: []uint64{0, 1},
			total:  1 * time.Second,
		},
		{
			name: "subsequent txs fail after tx failure",
			max:  1,
			calls: []queueCall{
				{call: sendQueueFunc, queued: true},
				{call: sendQueueFunc, queued: true, txErr: true},
				{call: sendQueueFunc, queued: true, txErr: true},
			},
			txs: []testTx{
				{},
				{sendErr: true},
				{},
			},
			nonces: []uint64{0, 1, 1},
			total:  3 * time.Second,
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			conf := configWithNumConfs(1)
			conf.ReceiptQueryInterval = 1 * time.Second // simulate a network send
			conf.ResubmissionTimeout = 2 * time.Second  // resubmit to detect errors
			conf.SafeAbortNonceTooLowCount = 1
			backend := newMockBackendWithNonce(newGasPricer(3))
			mgr := &SimpleTxManager{
				chainID: conf.ChainID,
				name:    "TEST",
				cfg:     conf,
				backend: backend,
				l:       testlog.Logger(t, log.LvlCrit),
				metr:    &metrics.NoopTxMetrics{},
			}

			// track the nonces, and return any expected errors from tx sending
			var nonces []uint64
			sendTx := func(ctx context.Context, tx *types.Transaction) error {
				index := int(tx.Data()[0])
				nonces = append(nonces, tx.Nonce())
				var testTx *testTx
				if index < len(test.txs) {
					testTx = &test.txs[index]
				}
				if testTx != nil && testTx.sendErr {
					return core.ErrNonceTooLow
				}
				txHash := tx.Hash()
				backend.mine(&txHash, tx.GasFeeCap())
				return nil
			}
			backend.setTxSender(sendTx)

			// for each factory call, create a candidate from the given test case's tx data
			txIndex := 0
			factory := TxFactory[int](func(ctx context.Context) (TxCandidate, int, error) {
				var testTx *testTx
				if txIndex < len(test.txs) {
					testTx = &test.txs[txIndex]
				}
				txIndex++
				if testTx != nil && testTx.factoryErr != nil {
					return TxCandidate{}, 0, testTx.factoryErr
				}
				return TxCandidate{
					TxData: []byte{byte(txIndex - 1)},
					To:     &common.Address{},
				}, txIndex - 1, nil
			})

			ctx := context.Background()
			queue := NewQueue[int](ctx, mgr, test.max, func(uint64) {})

			// make all the queue calls given in the test case
			start := time.Now()
			for i, c := range test.calls {
				msg := fmt.Sprintf("Call %d", i)
				c := c
				receiptCh := make(chan TxReceipt[int], 1)
				queued, err := c.call(factory, receiptCh, queue)
				require.Equal(t, c.queued, queued, msg)
				if c.callErr {
					require.Error(t, err, msg)
				} else {
					require.NoError(t, err, msg)
				}
				go func() {
					r := <-receiptCh
					if c.txErr {
						require.Error(t, r.Err, msg)
					} else {
						require.NoError(t, r.Err, msg)
					}
				}()
			}
			// wait for the queue to drain (all txs complete or failed)
			queue.Wait()
			duration := time.Since(start)
			// expect the execution time within a certain window
			require.Greater(t, duration, test.total, "test was faster than expected")
			require.Less(t, duration, test.total+500*time.Millisecond, "test was slower than expected")
			// check that the nonces match
			slices.Sort(nonces)
			require.Equal(t, test.nonces, nonces, "expected nonces do not match")
			// check
			require.Equal(t, len(test.txs), txIndex, "number of transactions sent does not match")
		})
	}
}
