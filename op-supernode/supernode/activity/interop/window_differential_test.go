package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Differential test of op-supernode's expiry rule (verifyExecutingMessage) against the shared vectors in
// packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json, which op-core's LinkChecker
// and kona's MessageGraph are checked against too.
// Spec: valid iff init <= exec && exec - init <= W, W = override > 0 ? override : 604800.

const supernodeWindowVectorsPath = "../../../../packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json"

type supernodeWindowVector struct {
	Name     string `json:"name"`
	Init     uint64 `json:"init"`
	Exec     uint64 `json:"exec"`
	Override uint64 `json:"override"`
	Window   uint64 `json:"window"`
	Rejected bool   `json:"configRejected"`
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason"`
}

var (
	windowSourceChain    = eth.ChainIDFromUInt64(901)
	windowExecutingChain = eth.ChainIDFromUInt64(900)
)

// newWindowInterop builds the Interop with New, from a dependency set carrying the override, activation 0 and block
// time 1 (so the activation rule never fires for timestamps >= 1). It then swaps the source chain's (empty) logsDB
// for one that contains every queried message, so only the timestamp rules decide. It returns nil if the dependency
// set rejects the override. Call it from a fresh subtest: the harness marks its test parallel.
func newWindowInterop(t *testing.T, override uint64) *Interop {
	t.Helper()
	ds, err := depset.NewStaticConfigDependencySetWithMessageExpiryOverride(map[eth.ChainID]*depset.StaticConfigDependency{
		windowSourceChain:    {},
		windowExecutingChain: {},
	}, override)
	if err != nil {
		return nil
	}
	h := newInteropTestHarness(t).WithActivation(0).WithChain(900, nil).WithChain(901, nil).SkipBuild()
	i := New(testLogger(), 0, ds, h.Chains(), h.dataDir, nil, 0, nil)
	require.NotNil(t, i)
	t.Cleanup(func() { _ = i.Stop(context.Background()) })
	require.NoError(t, i.logsDBs[windowSourceChain].Close())
	i.logsDBs[windowSourceChain] = &algoMockLogsDB{}
	return i
}

func verifyWindow(i *Interop, init, exec uint64) error {
	execMsg := &messages.ExecutingMessage{
		ChainID:   windowSourceChain,
		BlockNum:  50,
		LogIdx:    0,
		Timestamp: init,
		Checksum:  messages.MessageChecksum{0x01},
	}
	return i.verifyExecutingMessage(windowExecutingChain, exec, 0, execMsg, nil)
}

func TestVerifyExecutingMessageWindowVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.FromSlash(supernodeWindowVectorsPath))
	require.NoError(t, err)
	var f struct {
		DefaultWindow uint64                  `json:"defaultWindow"`
		Vectors       []supernodeWindowVector `json:"vectors"`
	}
	require.NoError(t, json.Unmarshal(data, &f))
	require.NotEmpty(t, f.Vectors)
	require.Equal(t, defaultMessageExpiryWindow, f.DefaultWindow)

	// One Interop (built with New) per override.
	byOverride := map[uint64][]supernodeWindowVector{}
	var order []uint64
	for _, v := range f.Vectors {
		if _, ok := byOverride[v.Override]; !ok {
			order = append(order, v.Override)
		}
		byOverride[v.Override] = append(byOverride[v.Override], v)
	}
	for _, ov := range order {
		vs := byOverride[ov]
		t.Run(fmt.Sprintf("override=%d", ov), func(t *testing.T) {
			i := newWindowInterop(t, ov)
			for _, v := range vs {
				// An override above the cap is rejected (never clamped); every other one is accepted.
				require.Equal(t, v.Rejected, i == nil, "%s: rejected=%v", v.Name, i == nil)
				if i == nil {
					continue
				}
				require.Equal(t, v.Window, i.messageExpiryWindow, "%s: effective window (verbatim)", v.Name)
				// The activation rule (activation 0, block time 1) must not fire first.
				require.GreaterOrEqual(t, v.Init, uint64(1), v.Name)
				require.GreaterOrEqual(t, v.Exec, uint64(1), v.Name)

				err := verifyWindow(i, v.Init, v.Exec)
				switch v.Reason {
				case "ok":
					require.NoError(t, err, v.Name)
				case "future":
					require.ErrorIs(t, err, ErrTimestampViolation, v.Name)
				case "expired":
					require.ErrorIs(t, err, ErrMessageExpired, v.Name)
				default:
					t.Fatalf("%s: unknown reason %q", v.Name, v.Reason)
				}
			}
		})
	}
}

// TestVerifyExecutingMessageWindowProperty checks verifyExecutingMessage against the spec formula (on unbounded
// integers) for random timestamps biased towards the window boundary and the u64 extremes.
func TestVerifyExecutingMessageWindowProperty(t *testing.T) {
	t.Run("override above the cap is rejected", func(t *testing.T) {
		require.Nil(t, newWindowInterop(t, defaultMessageExpiryWindow+1))
	})
	for _, ov := range []uint64{0, 1, 3600, defaultMessageExpiryWindow - 1, defaultMessageExpiryWindow} {
		t.Run(fmt.Sprintf("override=%d", ov), func(t *testing.T) {
			i := newWindowInterop(t, ov)
			require.NotNil(t, i)
			w := i.messageExpiryWindow
			r := rand.New(rand.NewSource(int64(0x23259 + ov)))
			pick := func() uint64 {
				switch r.Intn(4) {
				case 0:
					return r.Uint64()
				case 1:
					return 1 + uint64(r.Intn(4))
				case 2:
					return ^uint64(0) - uint64(r.Intn(4))
				default:
					return 1 + uint64(r.Int63n(1<<40))
				}
			}
			for n := 0; n < 5000; n++ {
				init := pick()
				var exec uint64
				if r.Intn(2) == 0 {
					delta := w + uint64(r.Intn(3)) - 1
					if r.Intn(2) == 0 {
						exec = init + delta
					} else {
						exec = init - delta
					}
				} else {
					exec = pick()
				}
				if init == 0 || exec == 0 {
					continue // activation rule, not under test
				}
				bi, be, bw := new(big.Int).SetUint64(init), new(big.Int).SetUint64(exec), new(big.Int).SetUint64(w)
				want := bi.Cmp(be) <= 0 && new(big.Int).Sub(be, bi).Cmp(bw) <= 0
				err := verifyWindow(i, init, exec)
				require.Equal(t, want, err == nil, "init=%d exec=%d W=%d err=%v", init, exec, w, err)
				if !want {
					require.True(t, errors.Is(err, ErrTimestampViolation) || errors.Is(err, ErrMessageExpired),
						"unexpected error %v", err)
				}
			}
		})
	}
}
