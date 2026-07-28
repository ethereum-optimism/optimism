package txintent

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

func execTriggerAt(timestamp uint64) *ExecTrigger {
	return &ExecTrigger{
		Executor: predeploys.CrossL2InboxAddr,
		Msg: messages.Message{
			Identifier: messages.Identifier{
				Origin:      common.Address{0xaa},
				BlockNumber: 10,
				LogIndex:    1,
				Timestamp:   timestamp,
				ChainID:     eth.ChainIDFromUInt64(901),
			},
			PayloadHash: common.Hash{0xbb},
		},
	}
}

// waitRecorder returns plan options installing the dependency wait, and what it waited for.
func waitRecorder() ([]txplan.Option, *[]uint64) {
	waited := new([]uint64)
	return []txplan.Option{
		func(tx *txplan.PlannedTx) {
			tx.AgainstBlock.Fn(func(context.Context) (eth.BlockInfo, error) {
				return nil, nil
			})
		},
		WithInteropDependencyWait(func(_ context.Context, minTime uint64) error {
			*waited = append(*waited, minTime)
			return nil
		}),
	}, waited
}

// planAgainstBlock evaluates the AgainstBlock stage of an intent executing call.
func planAgainstBlock(t *testing.T, call Call, opts ...txplan.Option) []uint64 {
	t.Helper()
	base, waited := waitRecorder()
	tx := NewIntent[Call, *InteropOutput](append(base, opts...)...)
	tx.Content.Set(call)
	_, err := tx.PlannedTx.AgainstBlock.Eval(context.Background())
	require.NoError(t, err)
	return *waited
}

func TestInteropDependencyWait(t *testing.T) {
	t.Run("waits for the executed message", func(t *testing.T) {
		require.Equal(t, []uint64{1000}, planAgainstBlock(t, execTriggerAt(1000)))
	})

	t.Run("waits for the newest of several executed messages", func(t *testing.T) {
		multi := &MultiTrigger{
			Emitter: predeploys.MultiCall3Addr,
			Calls:   []Call{execTriggerAt(1000), execTriggerAt(1200), execTriggerAt(900)},
		}
		require.Equal(t, []uint64{1200}, planAgainstBlock(t, multi))
	})

	t.Run("does not wait when no message is executed", func(t *testing.T) {
		init := &InitTrigger{Emitter: common.Address{0xcc}, Topics: [][32]byte{{0x01}}, OpaqueData: []byte{0x02}}
		require.Empty(t, planAgainstBlock(t, init))
	})

	t.Run("does not wait when opted out", func(t *testing.T) {
		waited := planAgainstBlock(t, execTriggerAt(1000), WithoutInteropDependencyWait())
		require.Empty(t, waited)
	})

	// The interop load test sets the access-list with an Option applied after the plan.
	t.Run("waits for an access-list set after the plan", func(t *testing.T) {
		accessList, err := execTriggerAt(1000).AccessList()
		require.NoError(t, err)
		base, waited := waitRecorder()
		tx := txplan.NewPlannedTx(append(base, txplan.WithAccessList(accessList))...)
		_, err = tx.AgainstBlock.Eval(context.Background())
		require.NoError(t, err)
		require.Equal(t, []uint64{1000}, *waited)
	})

	t.Run("ignores access-list entries of other contracts", func(t *testing.T) {
		minTime, err := executedMessagesMinTime(types.AccessList{
			{Address: common.Address{0xdd}, StorageKeys: []common.Hash{{0x01}}},
		})
		require.NoError(t, err)
		require.Zero(t, minTime)
	})

	t.Run("reports a malformed CrossL2Inbox access-list", func(t *testing.T) {
		_, err := executedMessagesMinTime(types.AccessList{{
			Address:     predeploys.CrossL2InboxAddr,
			StorageKeys: []common.Hash{{0x01}},
		}})
		require.ErrorContains(t, err, "parsing CrossL2Inbox access-list")
	})
}
