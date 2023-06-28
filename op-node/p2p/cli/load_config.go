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

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

func NewConfig(ctx *cli.Context, rollupCfg *rollup.Config) (*p2p.Config, error) {
	conf := &p2p.Config{}

	if ctx.Bool(flags.DisableP2P.Name) {
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

	if err := loadScoringParams(conf, ctx, rollupCfg); err != nil {
		return nil, fmt.Errorf("failed to load p2p peer scoring options: %w", err)
	}

	if err := loadBanningOptions(conf, ctx); err != nil {
		return nil, fmt.Errorf("failed to load banning option: %w", err)
	}

	conf.EnableReqRespSync = ctx.Bool(flags.SyncReqRespFlag.Name)

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

// loadScoringParams loads the peer scoring options from the CLI context.
func loadScoringParams(conf *p2p.Config, ctx *cli.Context, rollupCfg *rollup.Config) error {
	scoringLevel := ctx.String(flags.Scoring.Name)
	// Check old names for backwards compatibility
	if scoringLevel == "" {
		scoringLevel = ctx.String(flags.PeerScoring.Name)
	}
	if scoringLevel == "" {
		scoringLevel = ctx.String(flags.TopicScoring.Name)
	}
	if scoringLevel != "" {
		params, err := p2p.GetScoringParams(scoringLevel, rollupCfg)
		if err != nil {
			return err
		}
		conf.ScoringParams = params
	}

	return nil
}

// loadBanningOptions loads whether or not to ban peers from the CLI context.
func loadBanningOptions(conf *p2p.Config, ctx *cli.Context) error {
	conf.BanningEnabled = ctx.Bool(flags.Banning.Name)
	conf.BanningThreshold = ctx.Float64(flags.BanningThreshold.Name)
	conf.BanningDuration = ctx.Duration(flags.BanningDuration.Name)
	return nil
}

func loadListenOpts(conf *p2p.Config, ctx *cli.Context) error {
	listenIP := ctx.String(flags.ListenIP.Name)
	if listenIP != "" { // optional
		conf.ListenIP = net.ParseIP(listenIP)
		if conf.ListenIP == nil {
			return fmt.Errorf("failed to parse IP %q", listenIP)
		}
	}
	var err error
	conf.ListenTCPPort, err = validatePort(ctx.Uint(flags.ListenTCPPort.Name))
	if err != nil {
		return fmt.Errorf("bad listen TCP port: %w", err)
	}
	conf.ListenUDPPort, err = validatePort(ctx.Uint(flags.ListenUDPPort.Name))
	if err != nil {
		return fmt.Errorf("bad listen UDP port: %w", err)
	}
	return nil
}

func loadDiscoveryOpts(conf *p2p.Config, ctx *cli.Context) error {
	if ctx.Bool(flags.NoDiscovery.Name) {
		conf.NoDiscovery = true
	}

	var err error
	conf.AdvertiseTCPPort, err = validatePort(ctx.Uint(flags.AdvertiseTCPPort.Name))
	if err != nil {
		return fmt.Errorf("bad advertised TCP port: %w", err)
	}
	conf.AdvertiseUDPPort, err = validatePort(ctx.Uint(flags.AdvertiseUDPPort.Name))
	if err != nil {
		return fmt.Errorf("bad advertised UDP port: %w", err)
	}
	adIP := ctx.String(flags.AdvertiseIP.Name)
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

	dbPath := ctx.String(flags.DiscoveryPath.Name)
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

	bootnodes := make([]*enode.Node, 0)
	records := strings.Split(ctx.String(flags.Bootnodes.Name), ",")
	for i, recordB64 := range records {
		recordB64 = strings.TrimSpace(recordB64)
		if recordB64 == "" { // ignore empty records
			continue
		}
		nodeRecord, err := enode.Parse(enode.ValidSchemes, recordB64)
		if err != nil {
			return fmt.Errorf("bootnode record %d (of %d) is invalid: %q err: %w", i, len(records), recordB64, err)
		}
		bootnodes = append(bootnodes, nodeRecord)
	}
	if len(bootnodes) > 0 {
		conf.Bootnodes = bootnodes
	} else {
		conf.Bootnodes = p2p.DefaultBootnodes
	}

	return nil
}

func loadLibp2pOpts(conf *p2p.Config, ctx *cli.Context) error {
	addrs := strings.Split(ctx.String(flags.StaticPeers.Name), ",")
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

	for _, v := range strings.Split(ctx.String(flags.HostMux.Name), ",") {
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

	secArr := strings.Split(ctx.String(flags.HostSecurity.Name), ",")
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

	conf.PeersLo = ctx.Uint(flags.PeersLo.Name)
	conf.PeersHi = ctx.Uint(flags.PeersHi.Name)
	conf.PeersGrace = ctx.Duration(flags.PeersGrace.Name)
	conf.NAT = ctx.Bool(flags.NAT.Name)
	conf.UserAgent = ctx.String(flags.UserAgent.Name)
	conf.TimeoutNegotiation = ctx.Duration(flags.TimeoutNegotiation.Name)
	conf.TimeoutAccept = ctx.Duration(flags.TimeoutAccept.Name)
	conf.TimeoutDial = ctx.Duration(flags.TimeoutDial.Name)

	peerstorePath := ctx.String(flags.PeerstorePath.Name)
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
	raw := ctx.String(flags.P2PPrivRaw.Name)
	if raw != "" {
		return parsePriv(raw)
	}
	keyPath := ctx.String(flags.P2PPrivPath.Name)
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
	conf.MeshD = ctx.Int(flags.GossipMeshDFlag.Name)
	conf.MeshDLo = ctx.Int(flags.GossipMeshDloFlag.Name)
	conf.MeshDHi = ctx.Int(flags.GossipMeshDhiFlag.Name)
	conf.MeshDLazy = ctx.Int(flags.GossipMeshDlazyFlag.Name)
	conf.FloodPublish = ctx.Bool(flags.GossipFloodPublishFlag.Name)
	return nil
}
