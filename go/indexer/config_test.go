package indexer_test

import (
	"fmt"
	"testing"

	indexer "github.com/ethereum-optimism/optimism/go/indexer"
	"github.com/stretchr/testify/require"
)

var validateConfigTests = []struct {
	name   string
	cfg    indexer.Config
	expErr error
}{
	{
		name: "bad log level",
		cfg: indexer.Config{
			LogLevel: "unknown",
		},
		expErr: fmt.Errorf("unknown level: unknown"),
	},
	{
		name: "sequencer priv key or mnemonic none set",
		cfg: indexer.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "",
			SequencerHDPath:     "",
		},
		expErr: indexer.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic both set",
		cfg: indexer.Config{
			LogLevel: "info",

			SequencerPrivateKey: "sequencer-privkey",
			Mnemonic:            "mnemonic",
			SequencerHDPath:     "sequencer-path",
		},
		expErr: indexer.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic only mnemonic set",
		cfg: indexer.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "mnemonic",
			SequencerHDPath:     "",
		},
		expErr: indexer.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "sequencer priv key or mnemonic only hdpath set",
		cfg: indexer.Config{
			LogLevel: "info",

			SequencerPrivateKey: "",
			Mnemonic:            "",
			SequencerHDPath:     "sequencer-path",
		},
		expErr: indexer.ErrSequencerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic none set",
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "",
			ProposerHDPath:     "",
		},
		expErr: indexer.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic both set",
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "proposer-privkey",
			Mnemonic:           "mnemonic",
			ProposerHDPath:     "proposer-path",
		},
		expErr: indexer.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic only mnemonic set",
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "mnemonic",
			ProposerHDPath:     "",
		},
		expErr: indexer.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "proposer priv key or mnemonic only hdpath set",
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",

			ProposerPrivateKey: "",
			Mnemonic:           "",
			ProposerHDPath:     "proposer-path",
		},
		expErr: indexer.ErrProposerPrivKeyOrMnemonic,
	},
	{
		name: "same sequencer and proposer hd path",
		cfg: indexer.Config{
			LogLevel: "info",

			Mnemonic:        "mnemonic",
			SequencerHDPath: "path",
			ProposerHDPath:  "path",
		},
		expErr: indexer.ErrSameSequencerAndProposerHDPath,
	},
	{
		name: "same sequencer and proposer privkey",
		cfg: indexer.Config{
			LogLevel: "info",

			SequencerPrivateKey: "privkey",
			ProposerPrivateKey:  "privkey",
		},
		expErr: indexer.ErrSameSequencerAndProposerPrivKey,
	},
	{
		name: "sentry-dsn not set when sentry-enable is true",
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",
			ProposerPrivateKey:  "proposer-privkey",

			SentryEnable: true,
			SentryDsn:    "",
		},
		expErr: indexer.ErrSentryDSNNotSet,
	},
	// Valid configs
	{
		name: "valid config with privkeys and no sentry",
		cfg: indexer.Config{
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
		cfg: indexer.Config{
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
		cfg: indexer.Config{
			LogLevel:            "info",
			SequencerPrivateKey: "sequencer-privkey",
			ProposerPrivateKey:  "proposer-privkey",
			SentryEnable:        true,
			SentryDsn:           "indexer",
		},
		expErr: nil,
	},
	{
		name: "valid config with mnemonic and sentry",
		cfg: indexer.Config{
			LogLevel:        "info",
			Mnemonic:        "mnemonic",
			SequencerHDPath: "sequencer-path",
			ProposerHDPath:  "proposer-path",
			SentryEnable:    true,
			SentryDsn:       "indexer",
		},
		expErr: nil,
	},
}

// TestValidateConfig asserts the behavior of ValidateConfig by testing expected
// error and success configurations.
func TestValidateConfig(t *testing.T) {
	for _, test := range validateConfigTests {
		t.Run(test.name, func(t *testing.T) {
			err := indexer.ValidateConfig(&test.cfg)
			require.Equal(t, err, test.expErr)
		})
	}
}
