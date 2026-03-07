package sysgo

import (
	"encoding/hex"
	"flag"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
)

func newDevstackP2PConfig(
	p devtest.P,
	logger log.Logger,
	blockTime uint64,
	noDiscovery bool,
	enableReqRespSync bool,
	sequencerP2PKeyHex string,
) (*p2p.Config, p2p.SignerSetup) {
	require := p.Require()

	// make a dummy flagset since p2p config initialization helpers only input cli context
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	// use default flags
	for _, f := range opNodeFlags.P2PFlags(opNodeFlags.EnvVarPrefix) {
		require.NoError(f.Apply(fs))
	}
	// mandatory P2P flags
	require.NoError(fs.Set(opNodeFlags.AdvertiseIPName, "127.0.0.1"))
	require.NoError(fs.Set(opNodeFlags.AdvertiseTCPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.AdvertiseUDPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.ListenIPName, "127.0.0.1"))
	require.NoError(fs.Set(opNodeFlags.ListenTCPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.ListenUDPPortName, "0"))
	// avoid resource unavailable error by using memorydb
	require.NoError(fs.Set(opNodeFlags.DiscoveryPathName, "memory"))
	require.NoError(fs.Set(opNodeFlags.PeerstorePathName, "memory"))
	// Explicitly set to empty; do not default to resolving DNS of external bootnodes
	require.NoError(fs.Set(opNodeFlags.BootnodesName, ""))
	// For peer ID
	networkPrivKey, err := crypto.GenerateKey()
	networkPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(networkPrivKey))
	require.NoError(err)
	require.NoError(fs.Set(opNodeFlags.P2PPrivRawName, networkPrivKeyHex))
	if noDiscovery {
		require.NoError(fs.Set(opNodeFlags.NoDiscoveryName, "true"))
	}
	if enableReqRespSync {
		require.NoError(fs.Set(opNodeFlags.SyncReqRespName, "true"))
	}

	cliCtx := cli.NewContext(&cli.App{}, fs, nil)

	var p2pSignerSetup p2p.SignerSetup
	if sequencerP2PKeyHex != "" {
		require.NoError(fs.Set(opNodeFlags.SequencerP2PKeyName, sequencerP2PKeyHex))
		p2pSignerSetup, err = p2pcli.LoadSignerSetup(cliCtx, logger)
		require.NoError(err, "failed to load p2p signer")
		logger.Info("Sequencer key acquired")
	}

	p2pConfig, err := p2pcli.NewConfig(cliCtx, blockTime)
	require.NoError(err, "failed to load p2p config")
	p2pConfig.NoDiscovery = noDiscovery
	p2pConfig.EnableReqRespSync = enableReqRespSync

	return p2pConfig, p2pSignerSetup
}
