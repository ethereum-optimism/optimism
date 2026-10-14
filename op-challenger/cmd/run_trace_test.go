package main

import (
	"strings"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/runner"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestParseRunArg(t *testing.T) {
	tests := []struct {
		arg      string
		expected runner.RunConfig
		err      error
	}{
		{arg: "unknown/test1/0x1234", err: gameTypes.ErrUnknownGameType},
		{arg: "cannon", expected: runner.RunConfig{GameType: gameTypes.CannonGameType, Name: gameTypes.CannonGameType.String()}},
		{arg: "cannon-kona", expected: runner.RunConfig{GameType: gameTypes.CannonKonaGameType, Name: gameTypes.CannonKonaGameType.String()}},
		{arg: "super-cannon-kona", expected: runner.RunConfig{GameType: gameTypes.SuperCannonKonaGameType, Name: gameTypes.SuperCannonKonaGameType.String()}},
		{arg: "zk", err: ErrNotTraceGameType},
		{arg: "alphabet", err: ErrNotTraceGameType},
		{arg: "fast/test1", err: ErrNotTraceGameType},
		{arg: "permissioned/test1/0x1234", err: ErrNotTraceGameType},
		{arg: "cannon/test1", expected: runner.RunConfig{GameType: gameTypes.CannonGameType, Name: "test1"}},
		{arg: "cannon/test1/0x1234", expected: runner.RunConfig{GameType: gameTypes.CannonGameType, Name: "test1", Prestate: common.HexToHash("0x1234")}},
		{arg: "cannon/test1/0xinvalid", err: ErrInvalidPrestateHash},
		{arg: "cannon/test1/develop.bin.gz", expected: runner.RunConfig{GameType: gameTypes.CannonGameType, Name: "test1", PrestateFilename: "develop.bin.gz"}},
	}
	for _, test := range tests {
		test := test
		// Slash characters in test names confuse some things that parse the output as it looks like a subtest
		t.Run(strings.ReplaceAll(test.arg, "/", "_"), func(t *testing.T) {
			actual, err := parseRunArg(test.arg)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestDefaultRunConfigs(t *testing.T) {
	const skipWarning = "Skipping game type not supported by run-trace"
	tests := []struct {
		name       string
		configured []gameTypes.GameType
		expected   []runner.RunConfig
		skipped    int
		err        error
	}{
		{
			name:       "AllTraceTypes",
			configured: []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.CannonKonaGameType, gameTypes.SuperCannonKonaGameType},
			expected: []runner.RunConfig{
				{GameType: gameTypes.CannonGameType},
				{GameType: gameTypes.CannonKonaGameType},
				{GameType: gameTypes.SuperCannonKonaGameType},
			},
		},
		{
			name:       "SkipsNonTraceTypes",
			configured: []gameTypes.GameType{gameTypes.ZKDisputeGameType, gameTypes.CannonKonaGameType, gameTypes.PermissionedGameType},
			expected:   []runner.RunConfig{{GameType: gameTypes.CannonKonaGameType}},
			skipped:    2,
		},
		{
			name:       "NoneRemain",
			configured: []gameTypes.GameType{gameTypes.ZKDisputeGameType, gameTypes.AlphabetGameType},
			skipped:    2,
			err:        ErrNotTraceGameType,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, logs := testlog.CaptureLogger(t, log.LevelWarn)
			actual, err := defaultRunConfigs(logger, test.configured)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, test.expected, actual)
			logs.RequireMessageContainedNTimes(t, skipWarning, test.skipped, testlog.NewLevelFilter(log.LevelWarn))
		})
	}
}
