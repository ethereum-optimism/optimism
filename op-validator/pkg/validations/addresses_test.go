package validations

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/superchain-registry/validation"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestValidatorAddress(t *testing.T) {
	tests := []struct {
		name        string
		chainID     uint64
		version     string
		want        common.Address
		expectError bool
	}{
		{
			name:        "Valid Sepolia v1.8.0",
			chainID:     11155111,
			version:     standard.ContractsV180Tag,
			want:        common.HexToAddress("0x0a5bf8ebb4b177b2dcc6eba933db726a2e2e2b4d"),
			expectError: false,
		},
		{
			name:        "Valid Sepolia v2.0.0",
			chainID:     11155111,
			version:     standard.ContractsV200Tag,
			want:        common.HexToAddress("0x37739a6b0a3F1E7429499a4eC4A0685439Daff5C"),
			expectError: false,
		},
		{
			name:        "Invalid Chain ID",
			chainID:     999,
			version:     standard.ContractsV180Tag,
			want:        common.Address{},
			expectError: true,
		},
		{
			name:        "Invalid Version",
			chainID:     11155111,
			version:     "v99.0.0",
			want:        common.Address{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatorAddress(tt.chainID, tt.version)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAddressValidDeployment(t *testing.T) {
	op_e2e.InitParallel(t)

	for _, network := range []string{"mainnet", "sepolia"} {
		t.Run(network, func(t *testing.T) {
			op_e2e.InitParallel(t)
			testStandardVersionNetwork(t, network)
		})
	}
}

func testStandardVersionNetwork(t *testing.T, network string) {
	var rpcURL string
	var stdVersDefs validation.Versions
	var chainID uint64
	if network == "mainnet" {
		rpcURL = os.Getenv("MAINNET_RPC_URL")
		stdVersDefs = validation.StandardVersionsMainnet
		chainID = 1
	} else if network == "sepolia" {
		rpcURL = os.Getenv("SEPOLIA_RPC_URL")
		stdVersDefs = validation.StandardVersionsSepolia
		chainID = 11155111
	} else {
		t.Fatalf("Invalid network: %s", network)
	}

	require.NotEmpty(t, rpcURL, "RPC URL is empty")

	contractVersions := []string{
		standard.ContractsV180Tag,
		standard.ContractsV200Tag,
		standard.ContractsV300Tag,
		standard.ContractsV400Tag,
	}

	for _, semver := range contractVersions {
		version := stdVersDefs[validation.Semver(semver)]

		address, err := ValidatorAddress(chainID, semver)
		require.NoError(t, err)

		rpcClient, err := rpc.Dial(rpcURL)
		require.NoError(t, err)

		t.Run(semver, func(t *testing.T) {
			testStandardVersion(t, address, rpcClient, version)
		})
	}
}

func testStandardVersion(t *testing.T, address common.Address, rpcClient *rpc.Client, version validation.VersionConfig) {
	type fieldDef struct {
		getter string
		semver string
	}
	fields := []fieldDef{
		{
			"systemConfigVersion",
			version.SystemConfig.Version,
		},
		{
			"mipsVersion",
			version.Mips.Version,
		},
		{
			"optimismPortalVersion",
			version.OptimismPortal.Version,
		},
		{
			"anchorStateRegistryVersion",
			version.AnchorStateRegistry.Version,
		},
		{
			"delayedWETHVersion",
			version.DelayedWeth.Version,
		},
		{
			"disputeGameFactoryVersion",
			version.DisputeGameFactory.Version,
		},
		{
			"preimageOracleVersion",
			version.PreimageOracle.Version,
		},
		{
			"l1CrossDomainMessengerVersion",
			version.L1CrossDomainMessenger.Version,
		},
		{
			"l1ERC721BridgeVersion",
			version.L1ERC721Bridge.Version,
		},
		{
			"l1StandardBridgeVersion",
			version.L1StandardBridge.Version,
		},
		{
			"optimismMintableERC20FactoryVersion",
			version.OptimismMintableERC20Factory.Version,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w3c := w3.NewClient(rpcClient)
	for _, field := range fields {
		fn := w3.MustNewFunc(fmt.Sprintf("%s()", field.getter), "string")
		var outBytes []byte
		require.NoError(
			t,
			w3c.CallCtx(
				ctx,
				eth.Call(&w3types.Message{
					To:   &address,
					Func: fn,
				}, nil, nil).Returns(&outBytes),
			),
			"failed to call %s",
			field.getter,
		)

		var outVersion string
		require.NoError(t, fn.DecodeReturns(outBytes, &outVersion))
		require.Equal(t, field.semver, outVersion)
	}
}
