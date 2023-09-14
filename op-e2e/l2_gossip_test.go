package op_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestTxGossip(t *testing.T) {
	InitParallel(t)
	cfg := DefaultSystemConfig(t)
	gethOpts := []geth.GethOption{
		geth.WithP2P(),
	}
	cfg.GethOptions["sequencer"] = append(cfg.GethOptions["sequencer"], gethOpts...)
	cfg.GethOptions["verifier"] = append(cfg.GethOptions["verifier"], gethOpts...)
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Start system")

	seqClient := sys.Clients["sequencer"]
	verifClient := sys.Clients["verifier"]
	geth.ConnectP2P(t, seqClient, verifClient)

	// Send a transaction to the verifier and it should be gossiped to the sequencer and included in a block.
	SendL2Tx(t, cfg, verifClient, cfg.Secrets.Alice, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xaa}
		opts.Value = common.Big1
		opts.VerifyOnClients(seqClient, verifClient)
	})
}
