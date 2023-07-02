package cli

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
)

// TODO: implement remote signer setup (config to authenticated endpoint)
// and remote signer itself (e.g. a open http client to make signing requests)

// LoadSignerSetup loads a configuration for a Signer to be set up later
func LoadSignerSetup(ctx *cli.Context) (p2p.SignerSetup, error) {
	key := ctx.String(flags.SequencerP2PKeyFlag.Name)
	if key != "" {
		// Mnemonics are bad because they leak *all* keys when they leak.
		// Unencrypted keys from file are bad because they are easy to leak (and we are not checking file permissions).
		priv, err := crypto.HexToECDSA(key)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch submitter key: %w", err)
		}

		return &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(priv)}, nil
	}

	// TODO: create remote signer

	return nil, nil
}
