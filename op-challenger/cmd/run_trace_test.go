package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/runner"
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

func TestValidateRunPrestateUrls(t *testing.T) {
	cannonPrestatesURL := mustParseURL(t, "http://localhost/cannon")
	konaPrestatesURL := mustParseURL(t, "http://localhost/kona")

	tests := []struct {
		name       string
		cfg        *config.Config
		runConfigs []runner.RunConfig
		err        error
		errMsg     string
	}{
		{
			name: "NoPrestateFilename",
			cfg:  &config.Config{},
			runConfigs: []runner.RunConfig{{
				GameType: gameTypes.CannonGameType,
				Name:     "cannon",
			}},
		},
		{
			name: "SpecificPrestateHash",
			cfg:  &config.Config{},
			runConfigs: []runner.RunConfig{{
				GameType: gameTypes.CannonGameType,
				Name:     "cannon",
				Prestate: common.Hash{0xaa},
			}},
		},
		{
			name: "LocalPrestateFile",
			cfg:  &config.Config{},
			runConfigs: []runner.RunConfig{{
				GameType:         gameTypes.CannonGameType,
				Name:             "local",
				PrestateFilename: "file:/tmp/prestate.bin.gz",
			}},
		},
		{
			name: "CannonNamedPrestateRequiresBaseURL",
			cfg:  &config.Config{},
			runConfigs: []runner.RunConfig{{
				GameType:         gameTypes.CannonGameType,
				Name:             "develop",
				PrestateFilename: "develop.bin.gz",
			}},
			err:    config.ErrMissingPrestateBaseURL,
			errMsg: "--prestates-url/cannon-prestates-url",
		},
		{
			name: "CannonNamedPrestateWithBaseURL",
			cfg: &config.Config{
				CannonAbsolutePreStateBaseURL: cannonPrestatesURL,
			},
			runConfigs: []runner.RunConfig{{
				GameType:         gameTypes.CannonGameType,
				Name:             "develop",
				PrestateFilename: "develop.bin.gz",
			}},
		},
		{
			name: "KonaNamedPrestateRequiresBaseURL",
			cfg:  &config.Config{},
			runConfigs: []runner.RunConfig{{
				GameType:         gameTypes.CannonKonaGameType,
				Name:             "develop",
				PrestateFilename: "develop.bin.gz",
			}},
			err:    config.ErrMissingPrestateBaseURL,
			errMsg: "--prestates-url/cannon-kona-prestates-url",
		},
		{
			name: "SuperKonaNamedPrestateWithBaseURL",
			cfg: &config.Config{
				CannonKonaAbsolutePreStateBaseURL: konaPrestatesURL,
			},
			runConfigs: []runner.RunConfig{{
				GameType:         gameTypes.SuperCannonKonaGameType,
				Name:             "develop",
				PrestateFilename: "develop.bin.gz",
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRunPrestateUrls(test.runConfigs, test.cfg)
			require.ErrorIs(t, err, test.err)
			if test.errMsg != "" {
				require.ErrorContains(t, err, test.errMsg)
			}
		})
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}
