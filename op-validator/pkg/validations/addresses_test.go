package validations

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"

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
	t.Parallel()

	for _, network := range []string{"mainnet", "sepolia"} {
		t.Run(network, func(t *testing.T) {
			t.Parallel()
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
		if rpcURL == "" {
			rpcURL = "https://ethereum.publicnode.com"
		}
		stdVersDefs = validation.StandardVersionsMainnet
		chainID = 1
	} else if network == "sepolia" {
		rpcURL = os.Getenv("SEPOLIA_RPC_URL")
		if rpcURL == "" {
			rpcURL = "https://ethereum-sepolia-rpc.publicnode.com"
		}
		stdVersDefs = validation.StandardVersionsSepolia
		chainID = 11155111
	} else {
		t.Fatalf("Invalid network: %s", network)
	}

	contractVersions := []string{
		standard.ContractsV180Tag,
		standard.ContractsV200Tag,
		standard.ContractsV300Tag,
		standard.ContractsV400Tag,
		standard.ContractsV410Tag,
		standard.ContractsV500Tag,
	}

	for _, semver := range contractVersions {
		version, ok := stdVersDefs[validation.Semver(semver)]
		require.True(t, ok, "version %s not found in registry", semver)

		address, err := ValidatorAddress(chainID, semver)
		require.NoError(t, err, "failed to get validator address for %s", semver)
		require.NotEqual(t, common.Address{}, address, "validator address is zero for %s", semver)

		rpcClient, err := rpc.Dial(rpcURL)
		require.NoError(t, err)

		t.Run(semver, func(t *testing.T) {
			testStandardVersion(t, address, rpcClient, version, semver)
		})
	}
}

func testStandardVersion(t *testing.T, address common.Address, rpcClient *rpc.Client, version validation.VersionConfig, semverTag string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w3c := w3.NewClient(rpcClient)

	// Semver tags from the registry include the "op-contracts/" prefix.
	cleanTag := strings.TrimPrefix(strings.TrimPrefix(semverTag, "op-contracts/"), "v")
	releaseVer, err := semver.NewVersion(cleanTag)
	require.NoError(t, err)

	if releaseVer.Major() >= 5 {
		// For v5.0.0+
		type implFieldDef struct {
			implGetter string
			semver     string
		}
		implFields := []implFieldDef{
			{"systemConfigImpl", version.SystemConfig.Version},
			{"mipsImpl", version.Mips.Version},
			{"optimismPortalImpl", version.OptimismPortal.Version},
			{"anchorStateRegistryImpl", version.AnchorStateRegistry.Version},
			{"delayedWETHImpl", version.DelayedWeth.Version},
			{"disputeGameFactoryImpl", version.DisputeGameFactory.Version},
			{"l1CrossDomainMessengerImpl", version.L1CrossDomainMessenger.Version},
			{"l1ERC721BridgeImpl", version.L1ERC721Bridge.Version},
			{"l1StandardBridgeImpl", version.L1StandardBridge.Version},
			{"optimismMintableERC20FactoryImpl", version.OptimismMintableERC20Factory.Version},
		}

		versionFn := w3.MustNewFunc("version()", "string")
		for _, field := range implFields {
			implGetterFn := w3.MustNewFunc(fmt.Sprintf("%s()", field.implGetter), "address")
			var implAddrBytes []byte
			require.NoError(
				t,
				w3c.CallCtx(
					ctx,
					eth.Call(&w3types.Message{
						To:   &address,
						Func: implGetterFn,
					}, nil, nil).Returns(&implAddrBytes),
				),
				"failed to call %s",
				field.implGetter,
			)

			var implAddr common.Address
			require.NoError(t, implGetterFn.DecodeReturns(implAddrBytes, &implAddr), "failed to decode %s", field.implGetter)
			require.NotEqual(t, common.Address{}, implAddr, "implementation address is zero for %s", field.implGetter)

			var versionBytes []byte
			require.NoError(
				t,
				w3c.CallCtx(
					ctx,
					eth.Call(&w3types.Message{
						To:   &implAddr,
						Func: versionFn,
					}, nil, nil).Returns(&versionBytes),
				),
				"failed to call version() on %s implementation",
				field.implGetter,
			)

			var outVersion string
			require.NoError(t, versionFn.DecodeReturns(versionBytes, &outVersion), "failed to decode version for %s", field.implGetter)
			require.Equal(t, field.semver, outVersion, "version mismatch for %s", field.implGetter)
		}

		preimageOracleVersionFn := w3.MustNewFunc("preimageOracleVersion()", "string")
		var preimageOracleVersionBytes []byte
		require.NoError(
			t,
			w3c.CallCtx(
				ctx,
				eth.Call(&w3types.Message{
					To:   &address,
					Func: preimageOracleVersionFn,
				}, nil, nil).Returns(&preimageOracleVersionBytes),
			),
			"failed to call preimageOracleVersion",
		)

		var preimageOracleVersion string
		require.NoError(t, preimageOracleVersionFn.DecodeReturns(preimageOracleVersionBytes, &preimageOracleVersion), "failed to decode preimageOracleVersion")
		require.Equal(t, version.PreimageOracle.Version, preimageOracleVersion, "version mismatch for preimageOracleVersion")
	} else {
		// Older versions < v5.0.0
		type fieldDef struct {
			getter string
			semver string
		}
		fields := []fieldDef{
			{"systemConfigVersion", version.SystemConfig.Version},
			{"mipsVersion", version.Mips.Version},
			{"optimismPortalVersion", version.OptimismPortal.Version},
			{"anchorStateRegistryVersion", version.AnchorStateRegistry.Version},
			{"delayedWETHVersion", version.DelayedWeth.Version},
			{"disputeGameFactoryVersion", version.DisputeGameFactory.Version},
			{"preimageOracleVersion", version.PreimageOracle.Version},
			{"l1CrossDomainMessengerVersion", version.L1CrossDomainMessenger.Version},
			{"l1ERC721BridgeVersion", version.L1ERC721Bridge.Version},
			{"l1StandardBridgeVersion", version.L1StandardBridge.Version},
			{"optimismMintableERC20FactoryVersion", version.OptimismMintableERC20Factory.Version},
		}

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
			require.NoError(t, fn.DecodeReturns(outBytes, &outVersion), "failed to decode response for %s", field.getter)
			require.Equal(t, field.semver, outVersion, "version mismatch for %s", field.getter)
		}
	}
}
