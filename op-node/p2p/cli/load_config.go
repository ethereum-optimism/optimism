package cli

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"

	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

func NewConfig(ctx *cli.Context, blockTime uint64) (*p2p.Config, error) {
	conf := &p2p.Config{}

	if ctx.GlobalBool(flags.DisableP2P.Name) {
		conf.DisableP2P = true
		return conf, nil
	}

	p, err := loadNetworkPrivKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p priv key: %w", err)
	}
	conf.Priv = p

	if err := loadListenOpts(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p listen options: %w", err)
	}

	if err := loadDiscoveryOpts(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p discovery options: %w", err)
	}

	if err := loadLibp2pOpts(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p options: %w", err)
	}

	if err := loadGossipOptions(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p gossip options: %w", err)
	}

	if err := loadPeerScoringParams(conf, ctx, blockTime); err != nil {
		return nil, fmt.Errorf("failed to load p2p peer scoring options: %w", err)
	}

	if err := loadBanningOption(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load banning option: %w", err)
	}

	if err := loadTopicScoringParams(conf, ctx, blockTime); err != nil {
		return nil, fmt.Errorf("failed to load p2p topic scoring options: %w", err)
	}

	conf.ConnGater = p2p.DefaultConnGater
	conf.ConnMngr = p2p.DefaultConnManager

	return conf, nil
}

func validatePort(p uint) (uint16, error) {
	if p == 0 {
		return 0, nil
	}
	if p >= (1 << 16) {
		return 0, fmt.Errorf("port out of range: %d", p)
	}
	if p < 1024 {
		return 0, fmt.Errorf("port is reserved for system: %d", p)
	}
	return uint16(p), nil
}

// loadTopicScoringParams loads the topic scoring options from the CLI context.
//
// If the topic scoring options are not set, then the default topic scoring.
func loadTopicScoringParams(conf *p2p.Config, ctx *cli.Context, blockTime uint64) error {
	scoringLevel := ctx.GlobalString(flags.TopicScoring.Name)
	if scoringLevel != "" {
		// Set default block topic scoring parameters
		// See prysm: https://github.com/prysmaticlabs/prysm/blob/develop/beacon-chain/p2p/gossip_scoring_params.go
		// And research from lighthouse: https://gist.github.com/blacktemplar/5c1862cb3f0e32a1a7fb0b25e79e6e2c
		// And docs: https://github.com/libp2p/specs/blob/master/pubsub/gossipsub/gossipsub-v1.1.md#topic-parameter-calculation-and-decay
		topicScoreParams, err := p2p.GetTopicScoreParams(scoringLevel, blockTime)
		if err != nil {
			return err
		}
		conf.TopicScoring = topicScoreParams
	}

	return nil
}

// loadPeerScoringParams loads the scoring options from the CLI context.
//
// If the scoring level is not set, no scoring is enabled.
func loadPeerScoringParams(conf *p2p.Config, ctx *cli.Context, blockTime uint64) error {
	scoringLevel := ctx.GlobalString(flags.PeerScoring.Name)
	if scoringLevel != "" {
		peerScoreParams, err := p2p.GetPeerScoreParams(scoringLevel, blockTime)
		if err != nil {
			return err
		}
		conf.PeerScoring = peerScoreParams
	}

	return nil
}

// loadBanningOption loads whether or not to ban peers from the CLI context.
func loadBanningOption(conf *p2p.Config, ctx *cli.Context) error {
	ban := ctx.GlobalBool(flags.Banning.Name)
	conf.BanningEnabled = ban
	return nil
}

func loadListenOpts(conf *p2p.Config, ctx *cli.Context) error {
	listenIP := ctx.GlobalString(flags.ListenIP.Name)
	if listenIP != "" { // optional
		conf.ListenIP = net.ParseIP(listenIP)
		if conf.ListenIP == nil {
			return fmt.Errorf("failed to parse IP %q", listenIP)
		}
	}
	var err error
	conf.ListenTCPPort, err = validatePort(ctx.GlobalUint(flags.ListenTCPPort.Name))
	if err != nil {
		return fmt.Errorf("bad listen TCP port: %w", err)
	}
	conf.ListenUDPPort, err = validatePort(ctx.GlobalUint(flags.ListenUDPPort.Name))
	if err != nil {
		return fmt.Errorf("bad listen UDP port: %w", err)
	}
	return nil
}

func loadDiscoveryOpts(conf *p2p.Config, ctx *cli.Context) error {
	if ctx.GlobalBool(flags.NoDiscovery.Name) {
		conf.NoDiscovery = true
	}

	var err error
	conf.AdvertiseTCPPort, err = validatePort(ctx.GlobalUint(flags.AdvertiseTCPPort.Name))
	if err != nil {
		return fmt.Errorf("bad advertised TCP port: %w", err)
	}
	conf.AdvertiseUDPPort, err = validatePort(ctx.GlobalUint(flags.AdvertiseUDPPort.Name))
	if err != nil {
		return fmt.Errorf("bad advertised UDP port: %w", err)
	}
	adIP := ctx.GlobalString(flags.AdvertiseIP.Name)
	if adIP != "" { // optional
		ips, err := net.LookupIP(adIP)
		if err != nil {
			return fmt.Errorf("failed to lookup IP of %q to advertise in ENR: %w", adIP, err)
		}
		// Find the first v4 IP it resolves to
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				conf.AdvertiseIP = ipv4
				break
			}
		}
		if conf.AdvertiseIP == nil {
			return fmt.Errorf("failed to parse IP %q", adIP)
		}
	}

	dbPath := ctx.GlobalString(flags.DiscoveryPath.Name)
	if dbPath == "" {
		dbPath = "opnode_discovery_db"
	}
	if dbPath == "memory" {
		dbPath = ""
	}
	conf.DiscoveryDB, err = enode.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open discovery db: %w", err)
	}

	conf.Bootnodes = p2p.DefaultBootnodes
	records := strings.Split(ctx.GlobalString(flags.Bootnodes.Name), ",")
	for i, recordB64 := range records {
		recordB64 = strings.TrimSpace(recordB64)
		if recordB64 == "" { // ignore empty records
			continue
		}
		nodeRecord, err := enode.Parse(enode.ValidSchemes, recordB64)
		if err != nil {
			return fmt.Errorf("bootnode record %d (of %d) is invalid: %q err: %w", i, len(records), recordB64, err)
		}
		conf.Bootnodes = append(conf.Bootnodes, nodeRecord)
	}

	return nil
}

func loadLibp2pOpts(conf *p2p.Config, ctx *cli.Context) error {
	addrs := strings.Split(ctx.GlobalString(flags.StaticPeers.Name), ",")
	for i, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue // skip empty multi addrs
		}
		a, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("failed to parse multi addr of static peer %d (out of %d): %q err: %w", i, len(addrs), addr, err)
		}
		conf.StaticPeers = append(conf.StaticPeers, a)
	}

	for _, v := range strings.Split(ctx.GlobalString(flags.HostMux.Name), ",") {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "yamux":
			conf.HostMux = append(conf.HostMux, p2p.YamuxC())
		case "mplex":
			conf.HostMux = append(conf.HostMux, p2p.MplexC())
		default:
			return fmt.Errorf("could not recognize mux %s", v)
		}
	}

	secArr := strings.Split(ctx.GlobalString(flags.HostSecurity.Name), ",")
	for _, v := range secArr {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "none": // no security, for debugging etc.
			if len(conf.HostSecurity) > 0 || len(secArr) > 1 {
				return errors.New("cannot mix secure transport protocols with no-security")
			}
			conf.NoTransportSecurity = true
		case "noise":
			conf.HostSecurity = append(conf.HostSecurity, p2p.NoiseC())
		case "tls":
			conf.HostSecurity = append(conf.HostSecurity, p2p.TlsC())
		default:
			return fmt.Errorf("could not recognize security %s", v)
		}
	}

	conf.PeersLo = ctx.GlobalUint(flags.PeersLo.Name)
	conf.PeersHi = ctx.GlobalUint(flags.PeersHi.Name)
	conf.PeersGrace = ctx.GlobalDuration(flags.PeersGrace.Name)
	conf.NAT = ctx.GlobalBool(flags.NAT.Name)
	conf.UserAgent = ctx.GlobalString(flags.UserAgent.Name)
	conf.TimeoutNegotiation = ctx.GlobalDuration(flags.TimeoutNegotiation.Name)
	conf.TimeoutAccept = ctx.GlobalDuration(flags.TimeoutAccept.Name)
	conf.TimeoutDial = ctx.GlobalDuration(flags.TimeoutDial.Name)

	peerstorePath := ctx.GlobalString(flags.PeerstorePath.Name)
	if peerstorePath == "" {
		return errors.New("peerstore path must be specified, use 'memory' to explicitly not persist peer records")
	}

	var err error
	var store ds.Batching
	if peerstorePath == "memory" {
		store = sync.MutexWrap(ds.NewMapDatastore())
	} else {
		store, err = leveldb.NewDatastore(peerstorePath, nil) // default leveldb options are fine
		if err != nil {
			return fmt.Errorf("failed to open leveldb db for peerstore: %w", err)
		}
	}
	conf.Store = store

	return nil
}

func loadNetworkPrivKey(ctx *cli.Context) (*crypto.Secp256k1PrivateKey, error) {
	raw := ctx.GlobalString(flags.P2PPrivRaw.Name)
	if raw != "" {
		return parsePriv(raw)
	}
	keyPath := ctx.GlobalString(flags.P2PPrivPath.Name)
	if keyPath == "" {
		return nil, errors.New("no p2p private key path specified, cannot auto-generate key without path")
	}
	f, err := os.OpenFile(keyPath, os.O_RDONLY, 0600)
	if os.IsNotExist(err) {
		p, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate new p2p priv key: %w", err)
		}
		b, err := p.Raw()
		if err != nil {
			return nil, fmt.Errorf("failed to encode new p2p priv key: %w", err)
		}
		f, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to store new p2p priv key: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(hex.EncodeToString(b)); err != nil {
			return nil, fmt.Errorf("failed to write new p2p priv key: %w", err)
		}
		return (p).(*crypto.Secp256k1PrivateKey), nil
	} else {
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read priv key file: %w", err)
		}
		return parsePriv(strings.TrimSpace(string(data)))
	}
}

func parsePriv(data string) (*crypto.Secp256k1PrivateKey, error) {
	if len(data) > 2 && data[:2] == "0x" {
		data = data[2:]
	}
	b, err := hex.DecodeString(data)
	if err != nil {
		return nil, errors.New("p2p priv key is not formatted in hex chars")
	}
	p, err := crypto.UnmarshalSecp256k1PrivateKey(b)
	if err != nil {
		// avoid logging the priv key in the error, but hint at likely input length problem
		return nil, fmt.Errorf("failed to parse priv key from %d bytes", len(b))
	}
	return (p).(*crypto.Secp256k1PrivateKey), nil
}

func loadGossipOptions(conf *p2p.Config, ctx *cli.Context) error {
	conf.MeshD = ctx.GlobalInt(flags.GossipMeshDFlag.Name)
	conf.MeshDLo = ctx.GlobalInt(flags.GossipMeshDloFlag.Name)
	conf.MeshDHi = ctx.GlobalInt(flags.GossipMeshDhiFlag.Name)
	conf.MeshDLazy = ctx.GlobalInt(flags.GossipMeshDlazyFlag.Name)
	conf.FloodPublish = ctx.GlobalBool(flags.GossipFloodPublishFlag.Name)
	return nil
}
