package interop

import (
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// Differential test of op-supernode's expiry rule (verifyExecutingMessage) against the shared vectors in
// packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json, which op-core's LinkChecker
// and kona's MessageRules are checked against too.
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

// newWindowInterop wires an Interop the way New does (messageExpiryWindow = dependencySet.MessageExpiryWindow()),
// with activation at 0, block time 1, and a logsDB that contains every queried message. It returns nil if the
// dependency set rejects the override.
func newWindowInterop(t *testing.T, override uint64) *Interop {
	t.Helper()
	ds, err := depset.NewStaticConfigDependencySetWithMessageExpiryOverride(map[eth.ChainID]*depset.StaticConfigDependency{
		windowSourceChain:    {},
		windowExecutingChain: {},
	}, override)
	if err != nil {
		return nil
	}
	return &Interop{
		activationTimestamp: 0,
		dependencySet:       ds,
		messageExpiryWindow: ds.MessageExpiryWindow(),
		logsDBs:             map[eth.ChainID]LogsDB{windowSourceChain: &algoMockLogsDB{}},
		chains: map[eth.ChainID]cc.InteropChain{
			windowSourceChain:    &algoMockChain{id: windowSourceChain},
			windowExecutingChain: &algoMockChain{id: windowExecutingChain},
		},
	}
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

	for _, v := range f.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			// The activation rule (activation 0, block time 1) must not fire first.
			require.GreaterOrEqual(t, v.Init, uint64(1))
			require.GreaterOrEqual(t, v.Exec, uint64(1))

			i := newWindowInterop(t, v.Override)
			require.Equal(t, v.Rejected, i == nil, "override %d: rejected=%v", v.Override, i == nil)
			if i == nil {
				return
			}
			require.Equal(t, v.Window, i.messageExpiryWindow, "effective window (verbatim)")

			err := verifyWindow(i, v.Init, v.Exec)
			switch v.Reason {
			case "ok":
				require.NoError(t, err)
			case "future":
				require.ErrorIs(t, err, ErrTimestampViolation)
			case "expired":
				require.ErrorIs(t, err, ErrMessageExpired)
			default:
				t.Fatalf("unknown reason %q", v.Reason)
			}
		})
	}
}

// TestVerifyExecutingMessageWindowProperty checks verifyExecutingMessage against the spec formula (on unbounded
// integers) for random timestamps biased towards the window boundary and the u64 extremes.
func TestVerifyExecutingMessageWindowProperty(t *testing.T) {
	r := rand.New(rand.NewSource(0x23259))
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
	require.Nil(t, newWindowInterop(t, defaultMessageExpiryWindow+1), "an override above the cap is rejected")
	overrides := []uint64{0, 1, defaultMessageExpiryWindow - 1, defaultMessageExpiryWindow, 3600}
	for _, ov := range overrides {
		i := newWindowInterop(t, ov)
		require.NotNil(t, i)
		w := i.messageExpiryWindow
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
				require.True(t, errorsIsAny(err, ErrTimestampViolation, ErrMessageExpired), "unexpected error %v", err)
			}
		}
	}
}

func errorsIsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
