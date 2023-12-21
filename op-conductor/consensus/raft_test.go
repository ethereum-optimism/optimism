package consensus

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestCommitAndRead(t *testing.T) {
	log := testlog.Logger(t, log.LvlInfo)
	serverID := "SequencerA"
	serverAddr := "127.0.0.1:50050"
	bootstrap := true
	now := uint64(time.Now().Unix())
	rollupCfg := &rollup.Config{
		CanyonTime: &now,
	}
	storageDir := "/tmp/sequencerA"
	if err := os.RemoveAll(storageDir); err != nil {
		t.Fatal(err)
	}

	cons, err := NewRaftConsensus(log, serverID, serverAddr, storageDir, bootstrap, rollupCfg)
	require.NoError(t, err)

	// wait till it became leader
	<-cons.LeaderCh()

	// eth.BlockV1
	payload := eth.ExecutionPayload{
		BlockNumber:  1,
		Timestamp:    hexutil.Uint64(now - 20),
		Transactions: []eth.Data{},
		ExtraData:    []byte{},
	}

	err = cons.CommitUnsafePayload(payload)
	require.NoError(t, err)

	unsafeHead := cons.LatestUnsafePayload()
	require.Equal(t, payload, unsafeHead)

	// eth.BlockV2
	payload = eth.ExecutionPayload{
		BlockNumber:  2,
		Timestamp:    hexutil.Uint64(time.Now().Unix()),
		Transactions: []eth.Data{},
		ExtraData:    []byte{},
		Withdrawals:  &types.Withdrawals{},
	}

	err = cons.CommitUnsafePayload(payload)
	require.NoError(t, err)

	unsafeHead = cons.LatestUnsafePayload()
	require.Equal(t, payload, unsafeHead)
}
