package depset

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestDependencySet(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		testDependencySetSerialization(t, "json",
			func(depSet *StaticConfigDependencySet) ([]byte, error) { return json.Marshal(depSet) },
			func(data []byte, depSet *StaticConfigDependencySet) error { return json.Unmarshal(data, depSet) },
		)
	})

	t.Run("TOML serialization", func(t *testing.T) {
		testDependencySetSerialization(t, "toml",
			func(depSet *StaticConfigDependencySet) ([]byte, error) {
				var buf bytes.Buffer
				encoder := toml.NewEncoder(&buf)
				if err := encoder.Encode(depSet); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			},
			func(data []byte, depSet *StaticConfigDependencySet) error {
				_, err := toml.Decode(string(data), depSet)
				return err
			},
		)
	})

	t.Run("expiry window override above the protocol window", func(t *testing.T) {
		deps := map[eth.ChainID]*StaticConfigDependency{eth.ChainIDFromUInt64(900): {}}
		_, err := NewStaticConfigDependencySetWithMessageExpiryOverride(deps, MessageExpiryTimeSecondsInterop+1)
		require.Error(t, err)
		var ds StaticConfigDependencySet
		require.Error(t, json.Unmarshal([]byte(`{"dependencies":{"900":{}},"overrideMessageExpiryWindow":604801}`), &ds))
		_, err = toml.Decode("override_message_expiry_window = 604801\n[dependencies.900]\n", &ds)
		require.Error(t, err)

		ok, err := NewStaticConfigDependencySetWithMessageExpiryOverride(deps, MessageExpiryTimeSecondsInterop)
		require.NoError(t, err)
		require.Equal(t, MessageExpiryTimeSecondsInterop, ok.MessageExpiryWindow())
	})

	// The same 7 days is MESSAGE_EXPIRY_WINDOW in L2ToL2CrossDomainMessenger.sol and in kona-genesis.
	t.Run("protocol expiry window is 7 days", func(t *testing.T) {
		require.Equal(t, uint64(604800), MessageExpiryTimeSecondsInterop)
	})

	t.Run("invalid TOML", func(t *testing.T) {
		bad := []byte(`dependencies = { bad = 1 }`)
		var ds StaticConfigDependencySet
		_, err := toml.Decode(string(bad), &ds)
		require.Error(t, err)
	})
}

func testDependencySetSerialization(
	t *testing.T,
	fileExt string,
	marshal func(*StaticConfigDependencySet) ([]byte, error),
	unmarshal func([]byte, *StaticConfigDependencySet) error,
) {
	d := path.Join(t.TempDir(), "tmp_dep_set."+fileExt)

	depSet, err := NewStaticConfigDependencySet(
		map[eth.ChainID]*StaticConfigDependency{
			eth.ChainIDFromUInt64(900): {},
			eth.ChainIDFromUInt64(901): {},
		})
	require.NoError(t, err)

	t.Run("DefaultExpiryWindow", func(t *testing.T) {
		data, err := marshal(depSet)
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(d, data, 0o644))

		// For JSON, use the loader. For TOML, unmarshal directly
		var result DependencySet
		if fileExt == "json" {
			loader := &JSONDependencySetLoader{Path: d}
			result, err = loader.LoadDependencySet()
			require.NoError(t, err)
		} else {
			fileData, err := os.ReadFile(d)
			require.NoError(t, err)

			var newDepSet StaticConfigDependencySet
			err = unmarshal(fileData, &newDepSet)
			require.NoError(t, err)
			result = &newDepSet
		}

		chainIDs := result.Chains()
		require.ElementsMatch(t, []eth.ChainID{
			eth.ChainIDFromUInt64(900),
			eth.ChainIDFromUInt64(901),
		}, chainIDs)

		require.Equal(t, MessageExpiryTimeSecondsInterop, result.MessageExpiryWindow())
	})

	t.Run("CustomExpiryWindow", func(t *testing.T) {
		depSet.overrideMessageExpiryWindow = 15

		data, err := marshal(depSet)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(d, data, 0o644))

		var result DependencySet
		if fileExt == "json" {
			loader := &JSONDependencySetLoader{Path: d}
			result, err = loader.LoadDependencySet()
			require.NoError(t, err)
		} else {
			fileData, err := os.ReadFile(d)
			require.NoError(t, err)

			var newDepSet StaticConfigDependencySet
			err = unmarshal(fileData, &newDepSet)
			require.NoError(t, err)
			result = &newDepSet
		}

		require.Equal(t, uint64(15), result.MessageExpiryWindow())
	})

	t.Run("HasChain", func(t *testing.T) {
		require.True(t, depSet.HasChain(eth.ChainIDFromUInt64(900)))
		require.False(t, depSet.HasChain(eth.ChainIDFromUInt64(902)))
	})
}
