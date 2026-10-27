package dsl

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestWaitL1OriginHash(t *testing.T) {
	t.Parallel()
	target := eth.BlockID{Hash: common.Hash{7}, Number: 7}
	matching := eth.L2BlockRef{Number: 20, L1Origin: target}
	later := eth.L2BlockRef{Number: 21, L1Origin: eth.BlockID{Hash: common.Hash{8}, Number: 8}}
	wrongHash := matching
	wrongHash.L1Origin.Hash = common.Hash{99}

	for _, test := range []struct {
		name      string
		attempts  int
		prepare   func(*testutils.MockL2Client)
		wantErr   string
		wantCause error
	}{
		{
			name:     "retry missing safe head",
			attempts: 2,
			prepare: func(client *testutils.MockL2Client) {
				client.ExpectL2BlockRefByLabel(eth.Safe, eth.L2BlockRef{}, ethereum.NotFound)
				client.ExpectL2BlockRefByLabel(eth.Safe, matching, nil)
			},
		},
		{
			name:     "retry missing historical block",
			attempts: 2,
			prepare: func(client *testutils.MockL2Client) {
				client.ExpectL2BlockRefByLabel(eth.Safe, later, nil)
				client.ExpectL2BlockRefByNumber(matching.Number, eth.L2BlockRef{}, ethereum.NotFound)
				client.ExpectL2BlockRefByLabel(eth.Safe, later, nil)
				client.ExpectL2BlockRefByNumber(matching.Number, matching, nil)
			},
		},
		{
			name:     "retry old L1 origin hash",
			attempts: 2,
			prepare: func(client *testutils.MockL2Client) {
				client.ExpectL2BlockRefByLabel(eth.Safe, wrongHash, nil)
				client.ExpectL2BlockRefByLabel(eth.Safe, matching, nil)
			},
		},
		{
			name:     "missing safe head exhausts attempts",
			attempts: 2,
			prepare: func(client *testutils.MockL2Client) {
				for range 2 {
					client.ExpectL2BlockRefByLabel(eth.Safe, eth.L2BlockRef{}, ethereum.NotFound)
				}
			},
			wantErr:   "fetch safe head",
			wantCause: ethereum.NotFound,
		},
		{
			name:     "missing historical block exhausts attempts",
			attempts: 2,
			prepare: func(client *testutils.MockL2Client) {
				for range 2 {
					client.ExpectL2BlockRefByLabel(eth.Safe, later, nil)
					client.ExpectL2BlockRefByNumber(matching.Number, eth.L2BlockRef{}, ethereum.NotFound)
				}
			},
			wantErr:   "fetch L2 block 20",
			wantCause: ethereum.NotFound,
		},
		{
			name:     "wrong hash fails",
			attempts: 1,
			prepare: func(client *testutils.MockL2Client) {
				client.ExpectL2BlockRefByLabel(eth.Safe, wrongHash, nil)
			},
			wantErr: "L1 origin hash of safe head does not match target",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &testutils.MockL2Client{}
			client.Mock.Test(t)
			t.Cleanup(func() { client.AssertExpectations(t) })
			test.prepare(client)
			node := NewL2ELNode(&originHashTestEL{
				t: devtest.SerialT(t), client: &originHashTestClient{refs: client},
			})

			if test.wantErr == "" {
				// Exercise the public helper too: a fatal assertion in its polling
				// callback must fail these tests rather than bypass the retry.
				node.WaitL1OriginHash(eth.Safe, target, test.attempts)
				return
			}
			err := node.waitL1OriginHash(eth.Safe, target, test.attempts)
			require.ErrorContains(t, err, test.wantErr)
			var exhausted *retry.ErrFailedPermanently
			require.ErrorAs(t, err, &exhausted)
			if test.wantCause != nil {
				require.ErrorIs(t, err, test.wantCause)
			}
		})
	}
}

type originHashTestEL struct {
	stack.L2ELNode
	t      devtest.T
	client apis.L2EthClient
}

func (el *originHashTestEL) T() devtest.T                  { return el.t }
func (el *originHashTestEL) Name() string                  { return "origin-hash-test" }
func (el *originHashTestEL) ChainID() eth.ChainID          { return eth.ChainIDFromUInt64(901) }
func (el *originHashTestEL) L2EthClient() apis.L2EthClient { return el.client }

type originHashTestClient struct {
	apis.L2EthClient
	refs *testutils.MockL2Client
}

func (client *originHashTestClient) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return client.refs.L2BlockRefByLabel(ctx, label)
}

func (client *originHashTestClient) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	return client.refs.L2BlockRefByNumber(ctx, number)
}
