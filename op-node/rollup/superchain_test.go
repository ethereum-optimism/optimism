package rollup

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/superchain"
	"github.com/stretchr/testify/require"
)

func TestApplyHardforks_NoForks(t *testing.T) {
	cfg := Config{}
	hardforks := superchain.HardforkConfig{}
	applyHardforks(&cfg, hardforks)
	requireAllHardforksSetCorrectly(t, cfg, hardforks)
}

func TestApplyHardforks(t *testing.T) {
	cfg := Config{}
	hardforkCfg := superchain.HardforkConfig{}

	// Set all hardforks
	hardforkVal := reflect.ValueOf(&hardforkCfg).Elem()
	for i := 0; i < hardforkVal.NumField(); i++ {
		val := uint64(i + 10) // +10 just so they're all arbitrary non-zero values
		hardforkVal.Field(i).Set(reflect.ValueOf(&val))
	}

	applyHardforks(&cfg, hardforkCfg)

	requireAllHardforksSetCorrectly(t, cfg, hardforkCfg)
}

func requireAllHardforksSetCorrectly(t *testing.T, cfg Config, hardforkCfg superchain.HardforkConfig) {
	hardforkType := reflect.TypeOf(hardforkCfg)
	hardforkVal := reflect.ValueOf(hardforkCfg)
	cfgVal := reflect.ValueOf(&cfg).Elem()
	for i := 0; i < hardforkVal.NumField(); i++ {
		hardforkField := hardforkType.Field(i)
		cfgField := cfgVal.FieldByName(hardforkField.Name)
		require.Equalf(t, hardforkVal.Field(i).Elem(), cfgField.Elem(), "missing hard fork field %v", hardforkField.Name)
	}
	// Regolith is always activated at genesis
	require.NotNil(t, cfg.RegolithTime)
	require.Zero(t, *cfg.RegolithTime)
}
