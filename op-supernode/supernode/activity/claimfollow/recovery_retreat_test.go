package claimfollow

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// A consumer cannot infer the reason for a retreat from this recovery plan
// alone: temporarily lower local safety and an unprocessed carrier denial can
// produce the same plan. Keeping the old unsafe suffix therefore needs more
// evidence than just Target == Anchor and Prefix == nil.
func TestRecoveryRetreatDoesNotProveUnsafeSuffixValid(t *testing.T) {
	var plans []*sources.FollowRecoveryStatus
	for _, denial := range []bool{false, true} {
		h := newHarness(t)
		h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
		h.r.fill(2, 8, "a", 0)
		h.r.safe = 8
		require.NoError(t, h.step())
		require.Equal(t, wantRef(8), h.status().LocalSafeL2)
		if denial {
			h.r.denied[1] = []common.Hash{h.r.blocks[1].env.ExecutionPayload.BlockHash}
		} else {
			h.r.localSafe, h.r.safe = 0, 0
		}
		require.NoError(t, h.step())
		status, err := NewAPI(h.f).SyncStatus(t.Context())
		require.NoError(t, err)
		plans = append(plans, status.Recovery)
	}
	require.Equal(t, plans[0], plans[1])
	require.Nil(t, plans[0].Prefix)
}
