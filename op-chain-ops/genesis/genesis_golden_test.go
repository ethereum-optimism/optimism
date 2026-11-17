package genesis

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// testL2Genesis returns the L2 genesis of the test deploy config with the fork schedule s,
// relative to the genesis time.
func testL2Genesis(t *testing.T, s forkstest.Schedule) *opgenesis.Genesis {
	t.Helper()
	config := testDeployConfig(t)
	for _, fork := range forkstest.TimestampELForks {
		config.SetForkTimeOffset(fork, s(fork))
	}
	g, err := NewL2Genesis(config, &eth.BlockRef{Time: 1_700_000_000})
	require.NoError(t, err)
	// Non-empty storage, so that the Isthmus withdrawals root is not the empty root.
	g.Alloc[predeploys.L2ToL1MessagePasserAddr] = types.Account{
		Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
		Balance: common.Big0,
	}
	return g
}

// TestNewL2GenesisHashGoldenVectors pins the genesis block hash of NewL2Genesis, as
// op-geth computes it, for a range of fork schedules.
func TestNewL2GenesisHashGoldenVectors(t *testing.T) {
	// Each hash is named after the last fork that changes the genesis block; the forks after it,
	// up to the next name, give the same block.
	const (
		bedrock  = "0x6bfba05d9126f48e5eedc91b55a5833b3b3b3e61c9547289660a9d3794b81ed3"
		canyon   = "0x4674eeaa7212ca57660498a8702babe935e4fd604959581484ed9227e431886e"
		ecotone  = "0x223e9633f540095a871e18094e74eec30e036fd36d9211ca7dd62b039624f6d0"
		holocene = "0xeecb9038e10c9e3e5cfcd4683e795f9a2fd03dc116e6cf8bb86c0e4a10af5c0c"
		isthmus  = "0x0143b340162366f67f6dd9021a44ceb2d8272292703beb02a2cb7047492f6331"
		jovian   = "0x05e17369a50a85899838878005765ce64886dada9139349ad9360066ca8be908"
	)
	for _, tc := range []struct {
		name     string
		schedule forkstest.Schedule
		hash     string
	}{
		{name: "staggered", schedule: forkstest.Staggered(), hash: bedrock},
		{name: "all after genesis", schedule: forkstest.StaticSchedule(2), hash: bedrock},
		{name: "through Regolith", schedule: forkstest.AtGenesisThrough(forks.Regolith), hash: bedrock},
		{name: "through Canyon", schedule: forkstest.AtGenesisThrough(forks.Canyon), hash: canyon},
		{name: "through Ecotone", schedule: forkstest.AtGenesisThrough(forks.Ecotone), hash: ecotone},
		{name: "through Fjord", schedule: forkstest.AtGenesisThrough(forks.Fjord), hash: ecotone},
		{name: "through Granite", schedule: forkstest.AtGenesisThrough(forks.Granite), hash: ecotone},
		{name: "through Holocene", schedule: forkstest.AtGenesisThrough(forks.Holocene), hash: holocene},
		{name: "through Isthmus", schedule: forkstest.AtGenesisThrough(forks.Isthmus), hash: isthmus},
		{name: "through Jovian", schedule: forkstest.AtGenesisThrough(forks.Jovian), hash: jovian},
		{name: "through Karst", schedule: forkstest.AtGenesisThrough(forks.Karst), hash: jovian},
		{name: "through Lagoon", schedule: forkstest.AtGenesisThrough(forks.Lagoon), hash: jovian},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := testL2Genesis(t, tc.schedule)
			require.Equal(t, common.HexToHash(tc.hash), g.ToBlock().Hash())
		})
	}
}

// TestNewL2GenesisJSONGoldenVector pins the genesis.json of NewL2Genesis for the
// staggered schedule, as op-geth encodes it.
func TestNewL2GenesisJSONGoldenVector(t *testing.T) {
	const want = `{"config":{"chainId":902,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
		`"shanghaiTime":1700002000,"cancunTime":1700003000,"pragueTime":1700007000,"osakaTime":1700009000,` +
		`"bedrockBlock":0,"regolithTime":1700001000,"canyonTime":1700002000,"ecotoneTime":1700003000,"fjordTime":1700004000,"graniteTime":1700005000,"holoceneTime":1700006000,"isthmusTime":1700007000,"jovianTime":1700008000,"karstTime":1700009000,"lagoonTime":1700010000,` +
		`"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000","optimism":{"eip1559Elasticity":2,"eip1559Denominator":8,"eip1559DenominatorCanyon":12}},` +
		`"nonce":"0x0","timestamp":"0x6553f100","extraData":"0x","gasLimit":"0x1c9c380","difficulty":"0x1","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","coinbase":"0x4200000000000000000000000000000000000011",` +
		`"alloc":{"4200000000000000000000000000000000000016":{"storage":{"0x0100000000000000000000000000000000000000000000000000000000000000":"0x0200000000000000000000000000000000000000000000000000000000000000"},"balance":"0x0"}},` +
		`"number":"0x0","gasUsed":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","baseFeePerGas":"0x3b9aca00","excessBlobGas":null,"blobGasUsed":null,"slotNumber":null}`

	g := testL2Genesis(t, forkstest.Staggered())
	data, err := json.Marshal(g)
	require.NoError(t, err)
	require.JSONEq(t, want, string(data))
}
