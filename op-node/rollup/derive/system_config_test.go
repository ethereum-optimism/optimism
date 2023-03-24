package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestProcessSystemConfigUpdateLogEvent(t *testing.T) {
	tests := []struct {
		name   string
		log    *types.Log
		config eth.SystemConfig
		err    bool
	}{
		{
			// The log data is ignored by consensus and no modifications to the
			// system config occur.
			name: "SystemConfigUpdateResourceConfig",
			log: &types.Log{
				Topics: []common.Hash{
					ConfigUpdateEventABIHash,
					ConfigUpdateEventVersion0,
					SystemConfigUpdateResourceConfig,
				},
			},
			config: eth.SystemConfig{},
			err:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := eth.SystemConfig{}
			err := ProcessSystemConfigUpdateLogEvent(&config, test.log)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, config, test.config)
		})
	}
}
