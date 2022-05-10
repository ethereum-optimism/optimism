package batchsubmitter_test

import (
	"fmt"
	"testing"

	batchsubmitter "github.com/ethereum-optimism/optimism/batch-submitter"
	"github.com/stretchr/testify/require"
)

var validateConfigTests = []struct {
	name   string
	cfg    batchsubmitter.Config
	expErr error
}{
	{
		name: "bad log level",
		cfg: batchsubmitter.Config{
			LogLevel: "unknown",
		},
		expErr: fmt.Errorf("unknown level: unknown"),
	},
	{
		name: "sequencer priv key or mnemonic none set",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "",
			SequencerHDPath:     "",
		},
		expErr: batchsubmitter.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic both set",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			SequencerPrivateKey: "sequencer-privkey",
			Mnemonic:            "mnemonic",
			SequencerHDPath:     "sequencer-path",
		},
		expErr: batchsubmitter.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic only mnemonic set",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "mnemonic",
			SequencerHDPath:     "",
		},
		expErr: batchsubmitter.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic only hdpath set",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "",
			SequencerHDPath:     "sequencer-path",
		},
		expErr: batchsubmitter.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic none set",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "",
			ProposerHDPath:     "",
		},
		expErr: batchsubmitter.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic both set",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "proposer-privkey",
			Mnemonic:           "mnemonic",
			ProposerHDPath:     "proposer-path",
		},
		expErr: batchsubmitter.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic only mnemonic set",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "mnemonic",
			ProposerHDPath:     "",
		},
		expErr: batchsubmitter.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic only hdpath set",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "",
			ProposerHDPath:     "proposer-path",
		},
		expErr: batchsubmitter.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "same sequencer and proposer hd path",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			Mnemonic:        "mnemonic",
			SequencerHDPath: "path",
			ProposerHDPath:  "path",
		},
		expErr: batchsubmitter.ErrSameSequencerAndProposerHDPath,
	},
	{
		name: "same sequencer and proposer privkey",
		cfg: batchsubmitter.Config{
			LogLevel: "info",

			SequencerPrivateKey: "privkey",
			ProposerPrivateKey:  "privkey",
		},
		expErr: batchsubmitter.ErrSameSequencerAndProposerPrivKey,
	},
	{
		name: "sentry-dsn not set when sentry-enable is true",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",
			ProposerPrivateKey:  "proposer-privkey",

			SentryEnable: true,
			SentryDsn:    "",
		},
		expErr: batchsubmitter.ErrSentryDSNNotSet,
	},
	// Valid configs
	{
		name: "valid config with privkeys and no sentry",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",
			ProposerPrivateKey:  "proposer-privkey",
			SentryEnable:        false,
			SentryDsn:           "",
		},
		expErr: nil,
	},
	{
		name: "valid config with mnemonic and no sentry",
		cfg: batchsubmitter.Config{
			LogLevel:        "info",
			Mnemonic:        "mnemonic",
			SequencerHDPath: "sequencer-path",
			ProposerHDPath:  "proposer-path",
			SentryEnable:    false,
			SentryDsn:       "",
		},
		expErr: nil,
	},
	{
		name: "valid config with privkeys and sentry",
		cfg: batchsubmitter.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",
			ProposerPrivateKey:  "proposer-privkey",
			SentryEnable:        true,
			SentryDsn:           "batch-submitter",
		},
		expErr: nil,
	},
	{
		name: "valid config with mnemonic and sentry",
		cfg: batchsubmitter.Config{
			LogLevel:        "info",
			Mnemonic:        "mnemonic",
			SequencerHDPath: "sequencer-path",
			ProposerHDPath:  "proposer-path",
			SentryEnable:    true,
			SentryDsn:       "batch-submitter",
		},
		expErr: nil,
	},
}

// TestValidateConfig asserts the behavior of ValidateConfig by testing expected
// error and success configurations.
func TestValidateConfig(t *testing.T) {
	for _, test := range validateConfigTests {
		t.Run(test.name, func(t *testing.T) {
			err := batchsubmitter.ValidateConfig(&test.cfg)
			require.Equal(t, err, test.expErr)
		})
	}
}
