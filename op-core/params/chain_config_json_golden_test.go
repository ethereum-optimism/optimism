package params

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// chainConfigJSONVectors are fork schedules with the encoding of jsonTestConfig for them. The
// encoding has op-geth's key set, and op-geth's key order except that the OP Stack keys
// follow all Ethereum keys.
var chainConfigJSONVectors = []struct {
	name     string
	schedule forkstest.Schedule
	json     string
}{
	{
		name:     "pre-Karst",
		schedule: forkstest.AtGenesisThrough(forks.Jovian),
		json: `{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
			`"shanghaiTime":0,"cancunTime":0,"pragueTime":0,"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000",` +
			`"bedrockBlock":0,"regolithTime":0,"canyonTime":0,"ecotoneTime":0,"fjordTime":0,"graniteTime":0,"holoceneTime":0,"isthmusTime":0,"jovianTime":0,` +
			`"optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}}`,
	},
	{
		name:     "Karst",
		schedule: forkstest.Override(forkstest.AtGenesisThrough(forks.Jovian), forks.Karst, ptr.New[uint64](9000)),
		json: `{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
			`"shanghaiTime":0,"cancunTime":0,"pragueTime":0,"osakaTime":9000,"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000",` +
			`"bedrockBlock":0,"regolithTime":0,"canyonTime":0,"ecotoneTime":0,"fjordTime":0,"graniteTime":0,"holoceneTime":0,"isthmusTime":0,"jovianTime":0,"karstTime":9000,` +
			`"optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}}`,
	},
	{
		name:     "staggered",
		schedule: forkstest.Staggered(),
		json: `{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
			`"shanghaiTime":2000,"cancunTime":3000,"pragueTime":7000,"osakaTime":9000,"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000",` +
			`"bedrockBlock":0,"regolithTime":1000,"canyonTime":2000,"ecotoneTime":3000,"fjordTime":4000,"graniteTime":5000,"holoceneTime":6000,"isthmusTime":7000,"jovianTime":8000,"karstTime":9000,"lagoonTime":10000,` +
			`"optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}}`,
	},
}

// opGethChainConfigJSON is op-geth's encoding of its chain config for the staggered
// schedule, which has the OP Stack keys among the Ethereum keys, and
// opGethChainConfigJSONWithoutOsaka the same without osakaTime, as op-chain-ops writes
// it for Karst.
const (
	opGethChainConfigJSONHead = `{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
		`"shanghaiTime":2000,"cancunTime":3000,"pragueTime":7000,`
	opGethChainConfigJSONTail = `"bedrockBlock":0,"regolithTime":1000,"canyonTime":2000,"ecotoneTime":3000,"fjordTime":4000,"graniteTime":5000,"holoceneTime":6000,"isthmusTime":7000,"jovianTime":8000,"karstTime":9000,"lagoonTime":10000,` +
		`"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000",` +
		`"optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}}`

	opGethChainConfigJSON             = opGethChainConfigJSONHead + `"osakaTime":9000,` + opGethChainConfigJSONTail
	opGethChainConfigJSONWithoutOsaka = opGethChainConfigJSONHead + opGethChainConfigJSONTail
)

// TestChainConfigJSONGoldenVectors pins the encoding of jsonTestConfig, including its key
// order, and that it decodes to the same config.
func TestChainConfigJSONGoldenVectors(t *testing.T) {
	for _, tc := range chainConfigJSONVectors {
		t.Run(tc.name, func(t *testing.T) {
			want := jsonTestConfig(t, tc.schedule)
			data, err := json.Marshal(want)
			require.NoError(t, err)
			require.Equal(t, tc.json, string(data))

			var got ChainConfig
			require.NoError(t, json.Unmarshal([]byte(tc.json), &got))
			requireConfigEqual(t, want, &got)
		})
	}
}

// TestChainConfigJSONDecodeOpGethGoldenVector decodes op-geth's encoding of its chain
// config, and the same without osakaTime.
func TestChainConfigJSONDecodeOpGethGoldenVector(t *testing.T) {
	for name, data := range map[string]string{
		"with osakaTime":    opGethChainConfigJSON,
		"without osakaTime": opGethChainConfigJSONWithoutOsaka,
	} {
		t.Run(name, func(t *testing.T) {
			var got ChainConfig
			require.NoError(t, json.Unmarshal([]byte(data), &got))
			requireConfigEqual(t, jsonTestConfig(t, forkstest.Staggered()), &got)
		})
	}
}
