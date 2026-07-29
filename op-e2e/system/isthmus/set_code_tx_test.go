package isthmus

import (
	"context"
	"fmt"
	"strings"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

func TestSetCodeInTxPool(t *testing.T) {
	op_e2e.InitParallel(t)

	tests := []struct {
		isthmus bool
	}{
		{isthmus: true},
		{isthmus: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("isthmus=%t", tt.isthmus), func(t *testing.T) {
			t.Parallel()
			cfg := e2esys.HoloceneSystemConfig(t, ptr[hexutil.Uint64](0))
			if tt.isthmus {
				cfg.DeployConfig.L2GenesisIsthmusTimeOffset = ptr(hexutil.Uint64(cfg.DeployConfig.L2BlockTime))
			} else {
				cfg.DeployConfig.L2GenesisIsthmusTimeOffset = nil
			}

			sys, err := cfg.Start(t)
			require.NoError(t, err, "Error starting up system")

			require.NoError(t, wait.ForNextBlock(context.Background(), sys.NodeClient(e2esys.RoleSeq)))

			receipt, err := helpers.SendL2SetCodeTx(
				cfg,
				sys.NodeClient(e2esys.RoleSeq),
				sys.Cfg.Secrets.Alice,
				func(opts *helpers.TxOpts) {
					opts.ToAddr = &common.Address{0xaa}
					opts.Gas = 10_000_000
				})

			if tt.isthmus {
				require.NoError(t, err)
				require.NotNil(t, receipt)
				require.Equal(t, receipt.Status, uint64(1), "SetCode tx should be successful")
			} else {
				require.Error(t, err, "SetCode tx should fail")
				errMsg := err.Error()
				require.Truef(t,
					strings.Contains(errMsg, "not yet in Prague") || // op-geth
						strings.Contains(errMsg, "transaction type not supported"), // op-reth
					"expected SetCode tx rejection before Isthmus, got %q", errMsg)
				require.Nil(t, receipt)
			}
		})
	}
}

func ptr[T any](t T) *T { return &t }
