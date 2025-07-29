package validations

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/superchain-registry/validation"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testutils/mockrpc"
)

const callParamsTemplate = `[
      {
        "to": "%s",
        "data": "0x30d14888000000000000000000000000189abaaaa82dfc015a588a7dbad6f13b1d3485bc000000000000000000000000034edd2a225f7f429a63e0f1d2084b9e0a93b538038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c0000000000000000000000000000000000000000000000000000000000aa37dc0000000000000000000000000000000000000000000000000000000000000001"
      },
      "latest"
]`

const callResult = "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000004b504444472d34302c504444472d44574554482d33302c504444472d414e43484f52502d34302c504c44472d34302c504c44472d44574554482d33302c504c44472d414e43484f52502d3430000000000000000000000000000000000000000000"

func TestValidate_Mocked(t *testing.T) {

	tests := []struct {
		version   validation.Semver
		validator func(rpcClient *rpc.Client) Validator
	}{
		{
			version: standard.ContractsV180Tag,
			validator: func(rpcClient *rpc.Client) Validator {
				return NewV180Validator(rpcClient)
			},
		},
		{
			version: standard.ContractsV200Tag,
			validator: func(rpcClient *rpc.Client) Validator {
				return NewV200Validator(rpcClient)
			},
		},
		{
			version: standard.ContractsV300Tag,
			validator: func(rpcClient *rpc.Client) Validator {
				return NewV300Validator(rpcClient)
			},
		},
		{
			version: standard.ContractsV400Tag,
			validator: func(rpcClient *rpc.Client) Validator {
				return NewV400Validator(rpcClient)
			},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			addr := addresses[11155111][string(tt.version)]
			callParams := fmt.Sprintf(callParamsTemplate, strings.ToLower(addr.Hex()))
			mockRPC := mockrpc.NewMockRPC(
				t,
				testlog.Logger(t, slog.LevelInfo),
				mockrpc.WithOkCall("eth_chainId", mockrpc.NullMatcher(), "0xaa36a7"), // sepolia chain ID in hex
				mockrpc.WithOkCall("eth_call", mockrpc.JSONParamsMatcher([]byte(callParams)), callResult),
			)
			rpcClient, err := rpc.Dial(mockRPC.Endpoint())
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			errCodes, err := tt.validator(rpcClient).Validate(ctx, BaseValidatorInput{
				ProxyAdminAddress:   common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
				SystemConfigAddress: common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"),
				AbsolutePrestate:    common.HexToHash("0x038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c"),
				L2ChainID:           big.NewInt(11155420),
			})
			require.NoError(t, err)
			require.Equal(t, []string{"PDDG-40", "PDDG-DWETH-30", "PDDG-ANCHORP-40", "PLDG-40", "PLDG-DWETH-30", "PLDG-ANCHORP-40"}, errCodes)
			mockRPC.AssertExpectations(t)
		})
	}
}
