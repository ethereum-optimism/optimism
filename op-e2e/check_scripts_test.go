package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/require"

	fjordChecks "github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-fjord/checks"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestCheckFjordScript ensures the op-chain-ops/cmd/check-fjord script runs successfully
// against a test chain with the fjord hardfork enabled
func TestCheckFjordScript(t *testing.T) {
	InitParallel(t)
	log := testlog.Logger(t, log.LevelInfo)

	cfg := DefaultSystemConfig(t)
	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisCanyonTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisActivation

	one := hexutil.Uint64(1)

	tests := []struct {
		name            string
		fjordActivation *hexutil.Uint64
		expectErr       bool
	}{
		{
			name:            "fjord_activated",
			fjordActivation: &one,
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
			cfg.DeployConfig.L2GenesisFjordTimeOffset = tt.fjordActivation

			sys, err := cfg.Start(t)
			require.Nil(t, err, "Error starting up system")
			defer sys.Close()

			<-time.After(time.Duration(cfg.DeployConfig.L2BlockTime) * time.Second * 2)

			checkFjordConfig := &fjordChecks.CheckFjordConfig{
				Log:  log,
				L2:   sys.Clients["sequencer"],
				Key:  sys.Cfg.Secrets.Alice,
				Addr: sys.Cfg.Secrets.Addresses().Alice,
			}
			err = fjordChecks.CheckAll(context.Background(), checkFjordConfig)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
