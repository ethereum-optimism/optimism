package test

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// createGethNode creates an in-memory geth node based on the configuration.
// The private keys are added to the keystore and are unlocked.
// If the node is l2, catalyst is enabled.
// The node should be started and then closed when done.
func createGethNode(l2 bool, nodeCfg *node.Config, ethCfg *ethconfig.Config, privateKeys []*ecdsa.PrivateKey) (*node.Node, *eth.Ethereum, error) {
	n, err := node.New(nodeCfg)
	if err != nil {
		n.Close()
		return nil, nil, err
	}

	keydir := n.KeyStoreDir()
	scryptN := keystore.LightScryptN
	scryptP := keystore.LightScryptP
	n.AccountManager().AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	ks := n.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	password := "foobar"
	for _, pk := range privateKeys {
		act, err := ks.ImportECDSA(pk, password)
		if err != nil {
			n.Close()
			return nil, nil, err
		}
		err = ks.Unlock(act, password)
		if err != nil {
			n.Close()
			return nil, nil, err
		}
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		n.Close()
		return nil, nil, err

	}
	// Enable catalyst if l2
	if l2 {
		if err := catalyst.Register(n, backend); err != nil {
			n.Close()
			return nil, nil, err
		}
	}
	return n, backend, nil

}

func l1Geth(cfg *systemConfig) (*node.Node, *eth.Ethereum, error) {
	wallet, err := hdwallet.NewFromMnemonic(cfg.mnemonic)
	if err != nil {
		return nil, nil, err
	}

	signer := deriveAccount(wallet, cfg.cliqueSigners[0])
	pk, _ := wallet.PrivateKey(signer)

	return createGethNode(false, cfg.l1.nodeConfig, cfg.l1.ethConfig, []*ecdsa.PrivateKey{pk})
}

func l2Geth(cfg *systemConfig) (*node.Node, *eth.Ethereum, error) {
	return createGethNode(true, cfg.l2Verifier.nodeConfig, cfg.l2Verifier.ethConfig, nil)
}

func l2SequencerGeth(cfg *systemConfig) (*node.Node, *eth.Ethereum, error) {
	return createGethNode(true, cfg.l2Sequencer.nodeConfig, cfg.l2Sequencer.ethConfig, nil)
}
