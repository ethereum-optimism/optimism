package p2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	rollupP2P "github.com/ethereum-optimism/optimism/op-node/p2p"
	rollupCli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	rollup "github.com/ethereum-optimism/optimism/op-node/rollup"
)

// enrPrefix is the prefix for enrs.
const enrPrefix = "enr:"

// b64format is the base64 encoding format used for enrs.
var b64format = base64.RawURLEncoding

// RequestEnr requests an enr from a node peer.
func RequestEnr(ctx *cli.Context) error {
	quitC := make(chan struct{})

	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stdout, log.TerminalFormat(true))),
	)

	// Read the enr from the command line
	enrStr := ctx.String("enr")
	if enrStr == "" {
		enrStr = ctx.Args().First()
	}
	if enrStr == "" {
		return fmt.Errorf("no enr provided")
	}
	log.Debug("Found enr", "enr", enrStr)

	// Read in the block time
	blockTime := ctx.Uint64("block-time")
	if blockTime == 0 {
		blockTime = 2
		log.Info("No block time provided, using default", "block-time", blockTime)
	}

	// Parse the enr argument string
	remoteNode, err := ParseENR(enrStr, enode.ValidSchemes)
	if err != nil {
		return fmt.Errorf("failed to parse enr: %w", err)
	}
	log.Debug("Parsed enr", "enr", remoteNode.String())

	// Create a new op-node rollup p2p config
	p2pConfig, err := rollupCli.NewConfig(ctx, blockTime)
	if err != nil {
		return fmt.Errorf("failed to load p2p config: %w", err)
	}
	log.Debug("Created p2p config, starting discovery...")

	// Create a new host
	bwc := metrics.NewBandwidthCounter()
	host, err := p2pConfig.Host(log.Root(), bwc)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	log.Info("Created host", "addrs", host.Addrs(), "peerID", host.ID().Pretty())

	// Monitor peers in a goroutine
	log.Info("Monitoring peers...")
	go MonitorPeers(host, log.Root(), quitC)

	// Find the active p2p port
	tcpPort, err := rollupP2P.FindActiveTCPPort(host)
	if err != nil {
		return fmt.Errorf("failed to find active tcp port: %w", err)
	}
	log.Debug("Found active tcp port", "tcpPort", tcpPort)

	// Notify of any new connections/streams/etc.
	host.Network().Notify(rollupP2P.NewNetworkNotifier(log.Root(), nil))
	log.Info("Setup new network notifier")

	// All nil if disabled.
	rollupCfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	_, dv5Udp, err := p2pConfig.Discovery(log.New("p2p", "discv5"), rollupCfg, tcpPort)
	if err != nil {
		return fmt.Errorf("failed to start discv5: %w", err)
	}

	log.Debug("Discv5 started, requesting enr...")

	for {
		// Printing discovery stats
		go LogDiscoveryStats(log.Root(), dv5Udp)
		nodeRecord, err := dv5Udp.RequestENR(remoteNode)
		if err != nil {
			log.Warn("Failed to request ENR", "err", err)
		} else {
			log.Info("Found ENR", "nodeRecord", nodeRecord)
		}
		// Calling Sleep method
		time.Sleep(5 * time.Second)
	}

	// Stop monitoring peers
	// quitC <- struct{}{}

	// return nil
}

// LogDiscoveryStats logs discovery stats.
func LogDiscoveryStats(log log.Logger, dv5Udp *discover.UDPv5) {
	log.Debug("Printing discovery stats...")
	log.Debug("Node ID", "node_id", dv5Udp.LocalNode().Node().IP())
	log.Debug("Udp Address", "udp_addr", dv5Udp.LocalNode().Node().UDP())
	log.Debug("Pubkey", "pubkey", dv5Udp.LocalNode().Node().Pubkey())
	for _, node := range dv5Udp.AllNodes() {
		log.Debug("Node", "node", node)
		time.Sleep(10 * time.Millisecond)
	}
	log.Debug("Total Nodes", "total_nodes", len(dv5Udp.AllNodes()))
}

// MonitorPeers periodically checks the status of static peers and reconnects
func MonitorPeers(h host.Host, log log.Logger, quitC chan struct{}) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	peers := h.Network().Peers()

	for {
		select {
		case <-tick.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			var wg sync.WaitGroup

			log.Debug("polling peers", "peers", len(peers))
			for _, peerId := range peers {
				connectedness := h.Network().Connectedness(peerId)
				log.Trace("static peer connectedness", "peer", peerId, "connectedness", connectedness)

				if connectedness == network.Connected {
					continue
				}

				wg.Add(1)
				go func(peerId peer.ID) {
					log.Warn("static peer disconnected, reconnecting", "peer", peerId)
					if err := DialStaticPeer(ctx, log, h.Network(), peerId); err != nil {
						log.Warn("error reconnecting to static peer", "peer", peerId, "err", err)
					}
					wg.Done()
				}(peerId)
			}

			wg.Wait()
			cancel()
		case <-quitC:
			return
		}
	}
}

// DialStaticPeer dials a static peer using a peer ID and a Network.
func DialStaticPeer(ctx context.Context, log log.Logger, net network.Network, peerId peer.ID) error {
	log.Info("dialing static peer", "peer", peerId)
	if _, err := net.DialPeer(ctx, peerId); err != nil {
		return err
	}
	return nil
}

// ParseENR parses an enr string into a [wrappedEnr].
func ParseENR(e string, validSchemes enr.IdentityScheme) (*enode.Node, error) {
	e = e[len(enrPrefix):]
	enc, err := b64format.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	var rec enr.Record
	if err := rlp.DecodeBytes(enc, &rec); err != nil {
		return nil, fmt.Errorf("invalid enr: %w", err)
	}
	n, err := enode.New(validSchemes, &rec)
	if err != nil {
		return nil, fmt.Errorf("invalid enr: %w", err)
	}
	return n, nil
}
