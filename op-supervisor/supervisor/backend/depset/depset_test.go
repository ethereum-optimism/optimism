package depset

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-node/params"
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
			result, err = loader.LoadDependencySet(context.Background())
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

		require.Equal(t, uint64(params.MessageExpiryTimeSecondsInterop), result.MessageExpiryWindow())
	})

	t.Run("CustomExpiryWindow", func(t *testing.T) {
		depSet.overrideMessageExpiryWindow = 15

		data, err := marshal(depSet)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(d, data, 0o644))

		var result DependencySet
		if fileExt == "json" {
			loader := &JSONDependencySetLoader{Path: d}
			result, err = loader.LoadDependencySet(context.Background())
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
