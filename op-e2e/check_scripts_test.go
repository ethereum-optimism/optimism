package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"

	fjordChecks "github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-fjord/checks"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestCheckFjordScript ensures the op-chain-ops/cmd/check-fjord script runs successfully
// against a test chain with the fjord hardfork activated/unactivated
func TestCheckFjordScript(t *testing.T) {
	log := testlog.Logger(t, log.LevelInfo)

	cfg := DefaultSystemConfig(t)
	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisCanyonTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisActivation

	tests := []struct {
		name            string
		fjordActivation *hexutil.Uint64
		expectErr       bool
	}{
		{
			name:            "fjord_activated",
			fjordActivation: &genesisActivation,
			expectErr:       false,
		},
		{
			name:            "fjord_unactivated",
			fjordActivation: nil,
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitParallel(t)
			cfg.DeployConfig.L2GenesisFjordTimeOffset = tt.fjordActivation

			sys, err := cfg.Start(t)
			require.NoError(t, err, "Error starting up system")
			defer sys.Close()

			checkFjordConfig := &fjordChecks.CheckFjordConfig{
				Log:  log,
				L2:   sys.Clients["sequencer"],
				Key:  sys.Cfg.Secrets.Alice,
				Addr: sys.Cfg.Secrets.Addresses().Alice,
			}
			if tt.expectErr {
				err = fjordChecks.CheckRIP7212(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckRIP7212")
				err = fjordChecks.CheckGasPriceOracle(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckGasPriceOracle")
				err = fjordChecks.CheckTxEmpty(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckTxEmpty")
				err = fjordChecks.CheckTxAllZero(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckTxAllZero")
				err = fjordChecks.CheckTxAll42(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckTxAll42")
				err = fjordChecks.CheckTxRandom(context.Background(), checkFjordConfig)
				require.Error(t, err, "expected error for CheckTxRandom")
			} else {
				err = fjordChecks.CheckAll(context.Background(), checkFjordConfig)
				require.NoError(t, err, "should not error on CheckAll")
			}
		})
	}
}
