package gethengine

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

// EngineWithP2P gives the in-process op-geth node a local devp2p stack (ephemeral key, no
// discovery, loopback listener), so it can be peered with another engine for execution-layer sync.
func EngineWithP2P() EngineOption {
	return func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
		p2pKey, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		nodeCfg.P2P = p2p.Config{
			MaxPeers:    100,
			NoDiscovery: true,
			ListenAddr:  "127.0.0.1:0",
			PrivateKey:  p2pKey,
		}
		return nil
	}
}
