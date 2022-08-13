package state_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/state-surgery/solc"
	"github.com/ethereum-optimism/optimism/state-surgery/state"
	"github.com/stretchr/testify/require"
)

var layout solc.StorageLayout

func init() {
	data, err := os.ReadFile("./testdata/layout.json")
	if err != nil {
		panic("layout.json not found")

	}
	if err := json.Unmarshal(data, &layout); err != nil {
		panic("cannot unmarshal storage layout")
	}
}

func TestComputeStorageSlots(t *testing.T) {
	values := state.StorageValues{}
	values["time"] = 12
	values["addr"] = "0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8"
	values["boolean"] = true

	slots, err := state.ComputeStorageSlots(&layout, values)
	require.Nil(t, err)
	require.NotNil(t, slots)
}
