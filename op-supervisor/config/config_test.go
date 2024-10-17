package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := validConfig()
	require.NoError(t, cfg.Check())
}

func TestRequireL2RPC(t *testing.T) {
	cfg := validConfig()
	cfg.L2RPCs = []string{}
	require.ErrorIs(t, cfg.Check(), ErrMissingL2RPC)
}

func TestRequireDependencySet(t *testing.T) {
	cfg := validConfig()
	cfg.DependencySetSource = nil
	require.ErrorIs(t, cfg.Check(), ErrMissingDependencySet)
}

func TestRequireDatadir(t *testing.T) {
	cfg := validConfig()
	cfg.Datadir = ""
	require.ErrorIs(t, cfg.Check(), ErrMissingDatadir)
}

func TestValidateMetricsConfig(t *testing.T) {
	cfg := validConfig()
	cfg.MetricsConfig.Enabled = true
	cfg.MetricsConfig.ListenPort = -1
	require.ErrorIs(t, cfg.Check(), metrics.ErrInvalidPort)
}

func TestValidatePprofConfig(t *testing.T) {
	cfg := validConfig()
	cfg.PprofConfig.ListenEnabled = true
	cfg.PprofConfig.ListenPort = -1
	require.ErrorIs(t, cfg.Check(), oppprof.ErrInvalidPort)
}

func TestValidateRPCConfig(t *testing.T) {
	cfg := validConfig()
	cfg.RPC.ListenPort = -1
	require.ErrorIs(t, cfg.Check(), rpc.ErrInvalidPort)
}

func validConfig() *Config {
	depSet := &depset.StaticConfigDependencySet{
		Dependencies: map[types.ChainID]*depset.StaticConfigDependency{
			types.ChainIDFromUInt64(900): &depset.StaticConfigDependency{
				ActivationTime: 0,
				HistoryMinTime: 0,
			},
		},
	}
	// Should be valid using only the required arguments passed in via the constructor.
	return NewConfig([]string{"http://localhost:8545"}, depSet, "./supervisor_testdir")
}
