package verify

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

func TestLogAutoVerifyFailure(t *testing.T) {
	logger, logs := testlog.CaptureLogger(t, slog.LevelInfo)

	LogAutoVerifyFailure(logger, "deployments/state.json", errors.New("explorer unavailable"))

	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelError),
		testlog.NewMessageFilter("Deployment succeeded but contract verification incomplete"),
		testlog.NewErrContainsFilter("explorer unavailable"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewLevelFilter(slog.LevelError),
		testlog.NewMessageFilter("Retry contract verification"),
		testlog.NewAttributesContainsFilter("command", "deployments/state.json"),
	))
}

func TestAutoVerifyContinuesWhenEtherscanIsUnavailable(t *testing.T) {
	projectDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(projectDir, "forge-artifacts"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "foundry.toml"),
		[]byte("[profile.default]\nsrc = 'src'\nout = 'forge-artifacts'\n"),
		0o644,
	))
	stateFile := filepath.Join(projectDir, "state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte("{}"), 0o644))

	locator, err := artifacts.NewFileLocator(projectDir)
	require.NoError(t, err)
	logger, logs := testlog.CaptureLogger(t, slog.LevelInfo)

	err = AutoVerify(
		context.Background(),
		logger,
		"http://localhost:8545",
		1,
		stateFile,
		stateFile,
		locator,
		"etherscan,blockscout",
		"",
		"",
	)
	require.NoError(t, err)

	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Contract verifier unavailable"),
		testlog.NewAttributesFilter("verifier", "etherscan"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Verification complete"),
		testlog.NewAttributesFilter("verifier", "blockscout"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Deployment succeeded but contract verification incomplete"),
	))
	require.NotNil(t, logs.FindLog(
		testlog.NewMessageFilter("Retry contract verification"),
	))
}
