package p2p

import (
	"bytes"
	"context"
	secureRand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func (conf *Config) Discovery(log log.Logger, rollupCfg *rollup.Config) (*enode.LocalNode, *discover.UDPv5, error) {
	if conf.NoDiscovery {
		return nil, nil, nil
	}
	localNode := enode.NewLocalNode(conf.DiscoveryDB, conf.Priv)
	if conf.AdvertiseIP != nil {
		localNode.SetStaticIP(conf.AdvertiseIP)
	}
	if conf.AdvertiseUDPPort != 0 {
		localNode.SetFallbackUDP(int(conf.AdvertiseUDPPort))
	}
	dat := OptimismENRData{
		chainID: rollupCfg.L2ChainID.Uint64(),
		version: 0,
	}
	localNode.Set(&dat)

	udpAddr := &net.UDPAddr{
		IP:   conf.ListenIP,
		Port: int(conf.ListenUDPPort),
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, err
	}

	cfg := discover.Config{
		PrivateKey:   conf.Priv,
		NetRestrict:  nil,
		Bootnodes:    conf.Bootnodes,
		Unhandled:    nil, // Not used in dv5
		Log:          log,
		ValidSchemes: enode.ValidSchemes,
	}
	udpV5, err := discover.ListenV5(conn, localNode, cfg)
	if err != nil {
		return nil, nil, err
	}

	// TODO: periodically we can pull the external IP from libp2p NAT service,
	// and add it as a statement to keep the localNode accurate (if we trust the NAT device more than the discv5 statements)

	return localNode, udpV5, nil
}

func enrToAddrInfo(r *enode.Node) (*peer.AddrInfo, error) {
	ip := r.IP()
	ipScheme := "ip4"
	if ip4 := ip.To4(); ip4 == nil {
		ipScheme = "ip6"
	} else {
		ip = ip4
	}
	mAddr, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%s/tcp/%d", ipScheme, ip.String(), r.TCP()))
	if err != nil {
		return nil, fmt.Errorf("could not construct multi addr: %v", err)
	}
	pub := r.Pubkey()
	peerID, err := peer.IDFromPublicKey((*crypto.Secp256k1PublicKey)(pub))
	if err != nil {
		return nil, fmt.Errorf("could not compute peer ID from pubkey for multi-addr: %v", err)
	}
	return &peer.AddrInfo{
		ID:    peerID,
		Addrs: []ma.Multiaddr{mAddr},
	}, nil
}

// The discovery ENRs are just key-value lists, and we filter them by records tagged with the "optimism" key,
// and then check the chain ID and version.
type OptimismENRData struct {
	chainID uint64
	version uint64
}

func (o *OptimismENRData) ENRKey() string {
	return "optimism"
}

func (o *OptimismENRData) DecodeRLP(s *rlp.Stream) error {
	b, err := s.Bytes()
	if err != nil {
		return fmt.Errorf("failed to decode outer ENR entry: %v", err)
	}
	// We don't check the byte length: the below readers are limited, and the ENR itself has size limits.
	// Future "optimism" entries may contain additional data, and will be tagged with a newer version etc.
	r := bytes.NewReader(b)
	chainID, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("failed to read chain ID var int: %v", err)
	}
	version, err := binary.ReadUvarint(r)
	if err != nil {
		return fmt.Errorf("failed to read version var int: %v", err)
	}
	o.chainID = chainID
	o.version = version
	return nil
}

var _ enr.Entry = (*OptimismENRData)(nil)

func FilterEnodes(cfg *rollup.Config) func(node *enode.Node) bool {
	return func(node *enode.Node) bool {
		var dat OptimismENRData
		err := node.Load(&dat)
		// if the entry does not exist, or if it is invalid, then ignore the node
		if err != nil {
			return false
		}
		// check chain ID matches
		if cfg.L2ChainID.Uint64() != dat.chainID {
			return false
		}
		// check version matches
		if dat.version != 0 {
			return false
		}
		return true
	}
}

// DiscoveryProcess runs a discovery process that randomly walks the DHT to fill the peerstore,
// and connects to nodes in the peerstore that we are not already connected to.
// Nodes from the peerstore will be shuffled, unsuccessful connection attempts will cause peers to be avoided,
// and only nodes with addresses (under TTL) will be connected to.
func (n *NodeP2P) DiscoveryProcess(ctx context.Context, log log.Logger, cfg *rollup.Config, connectGoal uint) {
	if n.dv5Udp == nil {
		log.Warn("peer discovery is disabled")
		return
	}
	randomNodeIter := n.dv5Udp.RandomNodes()
	randomNodeIter = enode.Filter(randomNodeIter, FilterEnodes(cfg))

	discoverTicker := time.NewTicker(time.Second * 5)
	defer discoverTicker.Stop()

	connectTicker := time.NewTicker(time.Second * 20)
	defer connectTicker.Stop()

	connAttempts := make(chan peer.ID, 10)

	connectWorker := func(ctx context.Context) {
		for {
			id, ok := <-connAttempts
			if !ok {
				return
			}
			addrs := n.Host().Peerstore().Addrs(id)
			log.Info("attempting connection", "peer", id)
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			err := n.Host().Connect(ctx, peer.AddrInfo{ID: id, Addrs: addrs})
			cancel()
			if err != nil {
				log.Debug("failed connection attempt", "peer", id, "err", err)
			}
		}
	}
	// stops all the workers when we are done
	defer close(connAttempts)
	// start workers to try connect to peers
	for i := 0; i < 4; i++ {
		go connectWorker(ctx)
	}

	pstore := n.Host().Peerstore()
	for {
		select {
		case <-ctx.Done():
			log.Info("stopped peer discovery")
			return // no ctx error, expected close
		case <-discoverTicker.C:
			if !randomNodeIter.Next() {
				log.Info("discv5 DHT iteration stopped, closing peer discovery now...")
				return
			}
			found := randomNodeIter.Node()
			var dat OptimismENRData
			if err := found.Load(&dat); err != nil { // we already filtered on chain ID and version
				continue
			}
			info, err := enrToAddrInfo(found)
			if err != nil {
				continue
			}
			// We add the addresses to the peerstore, and update the address TTL to 24 hours.
			//After that we stop using the address, assuming it may not be valid anymore (until we rediscover the node)
			pstore.AddAddrs(info.ID, info.Addrs, time.Hour*24)
			_ = pstore.AddPubKey(info.ID, (*crypto.Secp256k1PublicKey)(found.Pubkey()))
			// Tag the peer, we'd rather have the connection manager prune away old peers,
			// or peers on different chains, or anyone we have not seen via discovery.
			// There is no tag score decay yet, so just set it to 42.
			n.ConnectionManager().TagPeer(info.ID, fmt.Sprintf("optimism-%d-%d", dat.chainID, dat.version), 42)
			log.Debug("discovered peer", "peer", info.ID, "nodeID", found.ID(), "addr", info.Addrs[0])
		case <-connectTicker.C:
			connected := n.Host().Network().Peers()
			if uint(len(connected)) < connectGoal {
				peersWithAddrs := n.Host().Peerstore().PeersWithAddrs()
				if err := shufflePeers(peersWithAddrs); err != nil {
					continue
				}

				existing := make(map[peer.ID]struct{})
				for _, p := range connected {
					existing[p] = struct{}{}
				}

				// For 30 seconds, keep using these peers, and don't try new discovery/connections.
				// We don't need to search for more peers and try new connections if we already have plenty
				ctx, cancel := context.WithTimeout(ctx, time.Second*30)
				// connect to 4 peers in parallel
			peerLoop:
				for _, id := range peersWithAddrs {
					// skip peers that we are already connected to
					if _, ok := existing[id]; ok {
						continue
					}
					// skip peers that we were just connected to
					if n.Host().Network().Connectedness(id) == network.CannotConnect {
						continue
					}
					// schedule, if there is still space to schedule
					select {
					case connAttempts <- id:
					case <-ctx.Done():
						break peerLoop
					}
				}
				cancel()
			}
		}
	}
}

// shuffle the slice of peer IDs in-place with a RNG seeded by secure randomness.
func shufflePeers(ids peer.IDSlice) error {
	var x [8]byte // shuffling is not critical, just need to avoid basic predictability by outside peers
	if _, err := io.ReadFull(secureRand.Reader, x[:]); err != nil {
		return err
	}
	rng := rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(x[:]))))
	rng.Shuffle(len(ids), ids.Swap)
	return nil
}
