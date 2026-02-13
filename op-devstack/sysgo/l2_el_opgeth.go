package sysgo

import (
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type OpGeth struct {
	mu sync.Mutex

	p             devtest.CommonT
	logger        log.Logger
	name          string
	l2Net         *L2Network
	jwtPath       string
	jwtSecret     [32]byte
	supervisorRPC string
	l2Geth        *geth.GethInstance
	cfg           *L2ELConfig

	authRPC string
	userRPC string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy
}

var _ L2ELNode = (*OpGeth)(nil)

func (n *OpGeth) UserRPC() string {
	return n.userRPC
}

func (n *OpGeth) EngineRPC() string {
	return n.authRPC
}

func (n *OpGeth) JWTPath() string {
	return n.jwtPath
}

func (n *OpGeth) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.l2Geth != nil {
		n.logger.Warn("op-geth already started")
		return
	}

	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.logger.New("proxy", "l2el-auth"))
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.logger.New("proxy", "l2el-user"))
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "ws://" + n.userProxy.Addr()
	}

	require := n.p.Require()
	l2Geth, err := geth.InitL2(NewComponentTarget(n.name, n.l2Net.ChainID()).String(), n.l2Net.genesis, n.jwtPath,
		func(ethCfg *ethconfig.Config, nodeCfg *gn.Config) error {
			ethCfg.InteropMessageRPC = n.supervisorRPC
			ethCfg.InteropMempoolFiltering = true

			listenAddr := n.cfg.P2PAddr
			port := n.cfg.P2PPort
			listenAddr = listenAddr + ":" + strconv.Itoa(port)

			nodeCfg.P2P = p2p.Config{
				NoDiscovery: true,
				ListenAddr:  listenAddr,
				MaxPeers:    10,
			}

			if n.cfg.P2PNodeKeyHex != "" {
				priv, err := crypto.HexToECDSA(strings.TrimPrefix(n.cfg.P2PNodeKeyHex, "0x"))
				if err != nil {
					return err
				}
				nodeCfg.P2P.PrivateKey = priv
			}
			if len(n.cfg.StaticPeers) > 0 {
				nodes := make([]*enode.Node, 0, len(n.cfg.StaticPeers))
				for _, p := range n.cfg.StaticPeers {
					nn, err := enode.Parse(enode.ValidSchemes, p)
					if err != nil {
						return err
					}
					nodes = append(nodes, nn)
				}
				nodeCfg.P2P.StaticNodes = nodes
			}
			if len(n.cfg.TrustedPeers) > 0 {
				nodes := make([]*enode.Node, 0, len(n.cfg.TrustedPeers))
				for _, p := range n.cfg.TrustedPeers {
					nn, err := enode.Parse(enode.ValidSchemes, p)
					if err != nil {
						return err
					}
					nodes = append(nodes, nn)
				}
				nodeCfg.P2P.TrustedNodes = nodes
			}
			return nil
		})
	require.NoError(err)
	require.NoError(l2Geth.Node.Start())
	n.l2Geth = l2Geth
	n.authProxy.SetUpstream(ProxyAddr(require, l2Geth.AuthRPC().RPC()))
	n.userProxy.SetUpstream(ProxyAddr(require, l2Geth.UserRPC().RPC()))
}

func (n *OpGeth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.l2Geth == nil {
		n.logger.Warn("op-geth already stopped")
		return
	}
	n.logger.Info("Closing op-geth", "name", n.name, "chain", n.l2Net.ChainID())
	closeErr := n.l2Geth.Close()
	n.logger.Info("Closed op-geth", "name", n.name, "chain", n.l2Net.ChainID(), "err", closeErr)
	n.l2Geth = nil
}
