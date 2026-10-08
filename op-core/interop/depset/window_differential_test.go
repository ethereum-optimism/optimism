package depset

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Differential test of the interop message expiry rule against the shared vectors in
// packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json, which the kona test
// (rust/kona/crates/protocol/interop/tests/window_differential.rs) checks too.
// Spec: valid iff init <= exec && exec - init <= W, W = override > 0 ? override : 604800.

const windowVectorsPath = "../../../packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json"

type windowVector struct {
	Name     string `json:"name"`
	Init     uint64 `json:"init"`
	Exec     uint64 `json:"exec"`
	Override uint64 `json:"override"`
	Window   uint64 `json:"window"`
	Rejected bool   `json:"configRejected"`
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason"`
}

type windowVectorFile struct {
	DefaultWindow uint64         `json:"defaultWindow"`
	Cap           uint64         `json:"cap"`
	Vectors       []windowVector `json:"vectors"`
}

func loadWindowVectors(t *testing.T) windowVectorFile {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(windowVectorsPath))
	require.NoError(t, err)
	var f windowVectorFile
	require.NoError(t, json.Unmarshal(data, &f))
	require.NotEmpty(t, f.Vectors)
	return f
}

var (
	windowChainExec = eth.ChainIDFromUInt64(900)
	windowChainInit = eth.ChainIDFromUInt64(901)
)

// alwaysActiveLinkCfg uses a real StaticConfigDependencySet for HasChain and MessageExpiryWindow, with interop
// active (and past its activation block) at every timestamp, so CanExecute reduces to the ordering and window rules.
type alwaysActiveLinkCfg struct {
	*StaticConfigDependencySet
}

func (alwaysActiveLinkCfg) IsInterop(eth.ChainID, uint64) bool                { return true }
func (alwaysActiveLinkCfg) IsInteropActivationBlock(eth.ChainID, uint64) bool { return false }

var _ LinkerConfig = alwaysActiveLinkCfg{}

// newWindowDepSet builds the dependency set every way a node can (constructor, JSON config, TOML config) and requires
// them to agree. It returns nil if the override was rejected.
func newWindowDepSet(t *testing.T, override uint64) *StaticConfigDependencySet {
	t.Helper()
	deps := map[eth.ChainID]*StaticConfigDependency{windowChainExec: {}, windowChainInit: {}}
	fromCtor, ctorErr := NewStaticConfigDependencySetWithMessageExpiryOverride(deps, override)

	var fromJSON StaticConfigDependencySet
	cfg := fmt.Sprintf(`{"dependencies":{"900":{},"901":{}},"overrideMessageExpiryWindow":%d}`, override)
	jsonErr := json.Unmarshal([]byte(cfg), &fromJSON)

	var fromTOML StaticConfigDependencySet
	_, tomlErr := toml.Decode(fmt.Sprintf("override_message_expiry_window = %d\n[dependencies.900]\n[dependencies.901]\n", override), &fromTOML)

	require.Equal(t, ctorErr == nil, jsonErr == nil, "constructor and JSON disagree on override %d (ctor: %v, json: %v)", override, ctorErr, jsonErr)
	require.Equal(t, ctorErr == nil, tomlErr == nil, "constructor and TOML disagree on override %d (ctor: %v, toml: %v)", override, ctorErr, tomlErr)
	if ctorErr != nil {
		// Rejected by the cap itself, not by something else on the way.
		require.ErrorContains(t, ctorErr, "exceeds")
		require.ErrorContains(t, jsonErr, "exceeds")
		if override <= math.MaxInt64 {
			require.ErrorContains(t, tomlErr, "exceeds")
		} // else TOML integers are int64: the TOML parser rejects the value before the cap sees it.
		return nil
	}
	require.Equal(t, fromCtor.MessageExpiryWindow(), fromJSON.MessageExpiryWindow())
	require.Equal(t, fromCtor.MessageExpiryWindow(), fromTOML.MessageExpiryWindow())
	return fromCtor
}

func TestWindowDifferentialVectors(t *testing.T) {
	f := loadWindowVectors(t)
	require.Equal(t, MessageExpiryTimeSecondsInterop, f.DefaultWindow)

	for _, v := range f.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			ds := newWindowDepSet(t, v.Override)
			// An override above the cap is rejected (not clamped); every other override is accepted.
			require.Equal(t, v.Rejected, ds == nil, "override %d: rejected=%v", v.Override, ds == nil)
			if ds == nil {
				return
			}
			// Accepted: the override is used verbatim.
			require.Equal(t, v.Window, ds.MessageExpiryWindow(), "effective window")

			got := LinkerFromConfig(alwaysActiveLinkCfg{ds}).CanExecute(windowChainExec, v.Exec, windowChainInit, v.Init)
			require.Equal(t, v.Valid, got, "CanExecute(exec=%d, init=%d, W=%d), want reason %q", v.Exec, v.Init, v.Window, v.Reason)
		})
	}
}

// specValid is the spec formula on unbounded integers.
func specValid(init, exec, window uint64) bool {
	i, e, w := new(big.Int).SetUint64(init), new(big.Int).SetUint64(exec), new(big.Int).SetUint64(window)
	return i.Cmp(e) <= 0 && new(big.Int).Sub(e, i).Cmp(w) <= 0
}

// windowCase is a random (init, exec, override) triple, biased towards the window boundary and the u64 extremes.
type windowCase struct{ Init, Exec, Override uint64 }

func (windowCase) Generate(r *rand.Rand, _ int) reflect.Value {
	pick := func() uint64 {
		switch r.Intn(4) {
		case 0:
			return r.Uint64()
		case 1:
			return uint64(r.Intn(4)) // 0..3
		case 2:
			return ^uint64(0) - uint64(r.Intn(4)) // max-3..max
		default:
			return uint64(r.Int63n(1 << 40))
		}
	}
	var c windowCase
	switch r.Intn(4) {
	case 0:
		c.Override = 0
	case 1:
		c.Override = MessageExpiryTimeSecondsInterop + uint64(r.Intn(3)) - 1
	default:
		c.Override = pick()
	}
	w := c.Override
	if w == 0 {
		w = MessageExpiryTimeSecondsInterop
	}
	c.Init = pick()
	switch r.Intn(3) {
	case 0: // around the boundary, both directions
		delta := w + uint64(r.Intn(3)) - 1
		if r.Intn(2) == 0 {
			c.Exec = c.Init + delta // may wrap: still a valid u64 input
		} else {
			c.Exec = c.Init - delta
		}
	default:
		c.Exec = pick()
	}
	return reflect.ValueOf(c)
}

func TestWindowDifferentialProperty(t *testing.T) {
	prop := func(c windowCase) bool {
		ds := newWindowDepSet(t, c.Override)
		if (ds == nil) != (c.Override > MessageExpiryTimeSecondsInterop) {
			return false // must reject exactly the overrides above the cap
		}
		if ds == nil {
			return true
		}
		w := ds.MessageExpiryWindow()
		if c.Override != 0 && w != c.Override {
			return false // clamped or otherwise altered
		}
		got := LinkerFromConfig(alwaysActiveLinkCfg{ds}).CanExecute(windowChainExec, c.Exec, windowChainInit, c.Init)
		return got == specValid(c.Init, c.Exec, w)
	}
	cfg := &quick.Config{MaxCount: 20000, Rand: rand.New(rand.NewSource(0x23259))}
	require.NoError(t, quick.Check(prop, cfg))
}
