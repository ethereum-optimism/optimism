package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestSuperchain(t *testing.T) {
	for _, network := range networks {
		t.Run(network, func(t *testing.T) {
			envVar := strings.ToUpper(network) + "_RPC_URL"
			rpcURL := os.Getenv(envVar)
			require.NotEmpty(t, rpcURL, "must specify RPC url via %s env var", envVar)
			testSuperchain(t, rpcURL)
		})
	}
}

func testSuperchain(t *testing.T, forkRPCURL string) {
	t.Parallel()

	if forkRPCURL == "" {
		t.Skip("forkRPCURL not set")
	}

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	forkedL1, stopL1, err := devnet.NewForked(lgr, forkRPCURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, stopL1())
	})
	l1RPC := forkedL1.RPCUrl()

	testCacheDir := testutils.IsolatedTestDirWithAutoCleanup(t)

	out, err := Superchain(ctx, SuperchainConfig{
		L1RPCUrl:         l1RPC,
		PrivateKey:       testutil.AnvilDefaultPrivateKey,
		ArtifactsLocator: artifacts.EmbeddedLocator,
		Logger:           lgr,

		SuperchainProxyAdminOwner:  common.Address{'S'},
		ProtocolVersionsOwner:      common.Address{'P'},
		Guardian:                   common.Address{'G'},
		Paused:                     false,
		RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
		RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
		CacheDir:                   testCacheDir,
		IsOPCMv2:                   false,
	})
	require.NoError(t, err)

	client, err := ethclient.Dial(l1RPC)
	require.NoError(t, err)

	addresses := []common.Address{
		out.SuperchainConfigProxy,
		out.SuperchainConfigImpl,
		out.SuperchainProxyAdmin,
		out.ProtocolVersionsImpl,
		out.ProtocolVersionsProxy,
	}
	for _, addr := range addresses {
		require.NotEmpty(t, addr)

		code, err := client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)
	}
}

func TestSuperchainConfig_Check(t *testing.T) {
	validPrivateKey := testutil.AnvilDefaultPrivateKey
	lgr := testlog.Logger(t, slog.LevelInfo)

	baseConfig := func() SuperchainConfig {
		return SuperchainConfig{
			L1RPCUrl:                   "http://localhost:8545",
			PrivateKey:                 validPrivateKey,
			Logger:                     lgr,
			ArtifactsLocator:           artifacts.EmbeddedLocator,
			SuperchainProxyAdminOwner:  common.Address{'S'},
			ProtocolVersionsOwner:      common.Address{'P'},
			Guardian:                   common.Address{'G'},
			Paused:                     false,
			RequiredProtocolVersion:    params.ProtocolVersionV0{Major: 1}.Encode(),
			RecommendedProtocolVersion: params.ProtocolVersionV0{Major: 2}.Encode(),
			IsOPCMv2:                   false,
		}
	}

	tests := []struct {
		name           string
		mutator        func(*SuperchainConfig)
		expectError    bool
		errorSubstring string
	}{
		{
			name:        "valid config for OPCM v1",
			mutator:     func(cfg *SuperchainConfig) {},
			expectError: false,
		},
		{
			name: "valid config for OPCM v2",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = true
				cfg.ProtocolVersionsOwner = common.Address{}
				cfg.RequiredProtocolVersion = params.ProtocolVersion{}
				cfg.RecommendedProtocolVersion = params.ProtocolVersion{}
			},
			expectError: false,
		},
		{
			name: "missing L1RPCUrl",
			mutator: func(cfg *SuperchainConfig) {
				cfg.L1RPCUrl = ""
			},
			expectError:    true,
			errorSubstring: "l1RPCUrl must be specified",
		},
		{
			name: "missing PrivateKey",
			mutator: func(cfg *SuperchainConfig) {
				cfg.PrivateKey = ""
			},
			expectError:    true,
			errorSubstring: "private key must be specified",
		},
		{
			name: "invalid PrivateKey - not hex",
			mutator: func(cfg *SuperchainConfig) {
				cfg.PrivateKey = "not-a-valid-hex-key"
			},
			expectError:    true,
			errorSubstring: "failed to parse private key",
		},
		{
			name: "invalid PrivateKey - wrong length",
			mutator: func(cfg *SuperchainConfig) {
				cfg.PrivateKey = "0x1234"
			},
			expectError:    true,
			errorSubstring: "failed to parse private key",
		},
		{
			name: "missing Logger",
			mutator: func(cfg *SuperchainConfig) {
				cfg.Logger = nil
			},
			expectError:    true,
			errorSubstring: "logger must be specified",
		},
		{
			name: "missing ArtifactsLocator",
			mutator: func(cfg *SuperchainConfig) {
				cfg.ArtifactsLocator = nil
			},
			expectError:    true,
			errorSubstring: "artifacts locator must be specified",
		},
		{
			name: "missing SuperchainProxyAdminOwner",
			mutator: func(cfg *SuperchainConfig) {
				cfg.SuperchainProxyAdminOwner = common.Address{}
			},
			expectError:    true,
			errorSubstring: "superchain proxy admin owner must be specified",
		},
		{
			name: "missing ProtocolVersionsOwner for OPCM v1",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = false
				cfg.ProtocolVersionsOwner = common.Address{}
			},
			expectError:    true,
			errorSubstring: "protocol versions owner must be specified",
		},
		{
			name: "missing RequiredProtocolVersion for OPCM v1",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = false
				cfg.RequiredProtocolVersion = params.ProtocolVersion{}
			},
			expectError:    true,
			errorSubstring: "required protocol version must be specified",
		},
		{
			name: "missing RecommendedProtocolVersion for OPCM v1",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = false
				cfg.RecommendedProtocolVersion = params.ProtocolVersion{}
			},
			expectError:    true,
			errorSubstring: "recommended protocol version must be specified",
		},
		{
			name: "ProtocolVersionsOwner must be zero for OPCM v2",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = true
				cfg.ProtocolVersionsOwner = common.Address{'P'}
				cfg.RequiredProtocolVersion = params.ProtocolVersion{}
				cfg.RecommendedProtocolVersion = params.ProtocolVersion{}
			},
			expectError:    true,
			errorSubstring: "protocol versions owner must be set to 0 for OPCM v2",
		},
		{
			name: "RequiredProtocolVersion must be zero for OPCM v2",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = true
				cfg.ProtocolVersionsOwner = common.Address{}
				cfg.RequiredProtocolVersion = params.ProtocolVersionV0{Major: 1}.Encode()
				cfg.RecommendedProtocolVersion = params.ProtocolVersion{}
			},
			expectError:    true,
			errorSubstring: "required protocol version must be set to 0 for OPCM v2",
		},
		{
			name: "RecommendedProtocolVersion must be zero for OPCM v2",
			mutator: func(cfg *SuperchainConfig) {
				cfg.IsOPCMv2 = true
				cfg.ProtocolVersionsOwner = common.Address{}
				cfg.RequiredProtocolVersion = params.ProtocolVersion{}
				cfg.RecommendedProtocolVersion = params.ProtocolVersionV0{Major: 1}.Encode()
			},
			expectError:    true,
			errorSubstring: "recommended protocol version must be set to 0 for OPCM v2",
		},
		{
			name: "missing Guardian",
			mutator: func(cfg *SuperchainConfig) {
				cfg.Guardian = common.Address{}
			},
			expectError:    true,
			errorSubstring: "guardian must be specified",
		},
		{
			name: "privateKey with 0x prefix",
			mutator: func(cfg *SuperchainConfig) {
				cfg.PrivateKey = validPrivateKey
			},
			expectError: false,
		},
		{
			name: "privateKey without 0x prefix",
			mutator: func(cfg *SuperchainConfig) {
				cfg.PrivateKey = strings.TrimPrefix(validPrivateKey, "0x")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig()
			tt.mutator(&cfg)
			err := cfg.Check()

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorSubstring)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
