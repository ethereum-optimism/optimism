package deployer

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// fakeL1 is a stub l1BlockFetcher that serves block refs keyed by the RPC argument
// (a tag for eth_getBlockByNumber, or a block hash for eth_getBlockByHash). A key
// that is absent yields a null result (zero hash), mirroring an unknown block.
type fakeL1 struct {
	byArg map[string]*state.L1BlockRefJSON
	err   error
}

func (f *fakeL1) CallContext(_ context.Context, result any, _ string, args ...any) error {
	if f.err != nil {
		return f.err
	}
	key, _ := args[0].(string)
	out := result.(*state.L1BlockRefJSON)
	if ref, ok := f.byArg[key]; ok {
		*out = *ref
	} else {
		*out = state.L1BlockRefJSON{}
	}
	return nil
}

func anchorRef(hash common.Hash, num uint64) *state.L1BlockRefJSON {
	return &state.L1BlockRefJSON{Hash: hash, Number: hexutil.Uint64(num)}
}

func TestSelectAnchorBlock(t *testing.T) {
	ctx := context.Background()
	safeHash := common.HexToHash("0x5afe")
	overrideHash := common.HexToHash("0xa11c0")

	t.Run("nil override returns the safe block", func(t *testing.T) {
		f := &fakeL1{byArg: map[string]*state.L1BlockRefJSON{
			"safe": anchorRef(safeHash, 100),
		}}
		got, err := selectAnchorBlock(ctx, f, nil)
		require.NoError(t, err)
		require.Equal(t, safeHash, got.Hash)
		require.EqualValues(t, 100, got.Number)
	})

	t.Run("override at or below the safe head is accepted", func(t *testing.T) {
		for _, height := range []uint64{50, 100} {
			f := &fakeL1{byArg: map[string]*state.L1BlockRefJSON{
				"safe":             anchorRef(safeHash, 100),
				overrideHash.Hex(): anchorRef(overrideHash, height),
			}}
			got, err := selectAnchorBlock(ctx, f, &overrideHash)
			require.NoError(t, err)
			require.Equal(t, overrideHash, got.Hash)
			require.EqualValues(t, height, got.Number)
		}
	})

	t.Run("override above the safe head is rejected", func(t *testing.T) {
		f := &fakeL1{byArg: map[string]*state.L1BlockRefJSON{
			"safe":             anchorRef(safeHash, 100),
			overrideHash.Hex(): anchorRef(overrideHash, 101),
		}}
		_, err := selectAnchorBlock(ctx, f, &overrideHash)
		require.ErrorContains(t, err, "above the L1 safe head")
	})

	t.Run("unknown override hash is rejected", func(t *testing.T) {
		f := &fakeL1{byArg: map[string]*state.L1BlockRefJSON{
			"safe": anchorRef(safeHash, 100),
		}}
		_, err := selectAnchorBlock(ctx, f, &overrideHash)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("safe fetch error is propagated", func(t *testing.T) {
		f := &fakeL1{err: errors.New("rpc down")}
		_, err := selectAnchorBlock(ctx, f, nil)
		require.ErrorContains(t, err, "failed to fetch L1 safe block")
	})
}
