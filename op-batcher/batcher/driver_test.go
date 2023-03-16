package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type FakeClient struct {
	ethclient.Client
}

func constructDefaultBatchSubmitter(l log.Logger) (*BatchSubmitter, error) {
	resubmissionTimeout, err := time.ParseDuration("30s")
	if err != nil {
		return nil, err
	}
	rpcClient := rpc.Client{}
	l1Client := ethclient.NewClient(&rpcClient)
	l2Client := ethclient.NewClient(&rpcClient)
	b, err := NewBatchSubmitter(
		context.Background(),
		Config{
			L1Client:     l1Client,
			L2Client:     l2Client,
			RollupNode:   nil,
			PollInterval: 10,
			TxManagerConfig: txmgr.Config{
				ResubmissionTimeout:       resubmissionTimeout,
				ReceiptQueryInterval:      time.Second,
				NumConfirmations:          1,
				SafeAbortNonceTooLowCount: 3,
				From:                      common.Address{},
				Signer:                    nil,
			},
			From:   common.Address{},
			Rollup: &rollup.Config{},
			Channel: ChannelConfig{
				SeqWindowSize:      15,
				ChannelTimeout:     40,
				MaxChannelDuration: 1,
				SubSafetyMargin:    4,
				MaxFrameSize:       120000,
				TargetFrameSize:    100000,
				TargetNumFrames:    1,
				ApproxComprRatio:   0.4,
			},
		},
		l,
	)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// TestDriverLoadBlocksIntoState ensures that the [BatchSubmitter] can load blocks into the state.
func TestDriverLoadBlocksIntoState(t *testing.T) {
	// Create a new [BatchSubmitter]
	log := testlog.Logger(t, log.LvlCrit)
	b, err := constructDefaultBatchSubmitter(log)
	require.NoError(t, err)

	// Load blocks into the state
	b.loadBlocksIntoState(context.Background())

}
