package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	leveldb "github.com/ipfs/go-ds-leveldb"
	lconf "github.com/libp2p/go-libp2p/config"
	core "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/mplex"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	cmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// SetupP2P provides a host and discovery service for usage in the rollup node.
type SetupP2P interface {
	Check() error
	// Host creates a libp2p host service. Returns nil, nil if p2p is disabled.
	Host(log log.Logger, reporter metrics.Reporter) (host.Host, error)
	// Discovery creates a disc-v5 service. Returns nil, nil, nil if discovery is disabled.
	Discovery(log log.Logger, rollupCfg *rollup.Config, tcpPort uint16) (*enode.LocalNode, *discover.UDPv5, error)
	TargetPeers() uint
}

// Config sets up a p2p host and discv5 service from configuration.
// This implements SetupP2P.
type Config struct {
	Priv *crypto.Secp256k1PrivateKey

	DisableP2P  bool
	NoDiscovery bool

	ListenIP      net.IP
	ListenTCPPort uint16

	// Port to bind discv5 to
	ListenUDPPort uint16

	AdvertiseIP      net.IP
	AdvertiseTCPPort uint16
	AdvertiseUDPPort uint16
	Bootnodes        []*enode.Node
	DiscoveryDB      *enode.DB

	StaticPeers []core.Multiaddr

	HostMux             []lconf.MsMuxC
	HostSecurity        []lconf.MsSecC
	NoTransportSecurity bool

	PeersLo    uint
	PeersHi    uint
	PeersGrace time.Duration

	// If true a NAT manager will host a NAT port mapping that is updated with PMP and UPNP by libp2p/go-nat
	NAT bool

	UserAgent string

	TimeoutNegotiation time.Duration
	TimeoutAccept      time.Duration
	TimeoutDial        time.Duration

	// Underlying store that hosts connection-gater and peerstore data.
	Store ds.Batching

	ConnGater func(conf *Config) (connmgr.ConnectionGater, error)
	ConnMngr  func(conf *Config) (connmgr.ConnManager, error)
}

type ConnectionGater interface {
	connmgr.ConnectionGater

	// BlockPeer adds a peer to the set of blocked peers.
	// Note: active connections to the peer are not automatically closed.
	BlockPeer(p peer.ID) error
	UnblockPeer(p peer.ID) error
	ListBlockedPeers() []peer.ID

	// BlockAddr adds an IP address to the set of blocked addresses.
	// Note: active connections to the IP address are not automatically closed.
	BlockAddr(ip net.IP) error
	UnblockAddr(ip net.IP) error
	ListBlockedAddrs() []net.IP

	// BlockSubnet adds an IP subnet to the set of blocked addresses.
	// Note: active connections to the IP subnet are not automatically closed.
	BlockSubnet(ipnet *net.IPNet) error
	UnblockSubnet(ipnet *net.IPNet) error
	ListBlockedSubnets() []*net.IPNet
}

func DefaultConnGater(conf *Config) (connmgr.ConnectionGater, error) {
	return conngater.NewBasicConnectionGater(conf.Store)
}

func DefaultConnManager(conf *Config) (connmgr.ConnManager, error) {
	return cmgr.NewConnManager(
		int(conf.PeersLo),
		int(conf.PeersHi),
		cmgr.WithGracePeriod(conf.PeersGrace),
		cmgr.WithSilencePeriod(time.Minute),
		cmgr.WithEmergencyTrim(true))
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

func NewConfig(ctx *cli.Context) (*Config, error) {
	conf := &Config{}

	if ctx.GlobalBool(flags.DisableP2P.Name) {
		conf.DisableP2P = true
		return conf, nil
	}

	p, err := loadNetworkPrivKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p priv key: %w", err)
	}
	conf.Priv = p

	if err := conf.loadListenOpts(ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p listen options: %w", err)
	}

	if err := conf.loadDiscoveryOpts(ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p discovery options: %w", err)
	}

	if err := conf.loadLibp2pOpts(ctx); err != nil {
		return nil, fmt.Errorf("failed to load p2p options: %w", err)
	}

	conf.ConnGater = DefaultConnGater
	conf.ConnMngr = DefaultConnManager

	return conf, nil
}

func (conf *Config) TargetPeers() uint {
	return conf.PeersLo
}

func (conf *Config) loadListenOpts(ctx *cli.Context) error {
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

func (conf *Config) loadDiscoveryOpts(ctx *cli.Context) error {
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

func yamuxC() (lconf.MsMuxC, error) {
	mtpt, err := lconf.MuxerConstructor(yamux.DefaultTransport)
	if err != nil {
		return lconf.MsMuxC{}, err
	}
	return lconf.MsMuxC{MuxC: mtpt, ID: "/yamux/1.0.0"}, nil
}

func mplexC() (lconf.MsMuxC, error) {
	mtpt, err := lconf.MuxerConstructor(mplex.DefaultTransport)
	if err != nil {
		return lconf.MsMuxC{}, err
	}
	return lconf.MsMuxC{MuxC: mtpt, ID: "/mplex/6.7.0"}, nil
}

func noiseC() (lconf.MsSecC, error) {
	stpt, err := lconf.SecurityConstructor(noise.New)
	if err != nil {
		return lconf.MsSecC{}, err
	}
	return lconf.MsSecC{SecC: stpt, ID: noise.ID}, nil
}

func tlsC() (lconf.MsSecC, error) {
	stpt, err := lconf.SecurityConstructor(tls.New)
	if err != nil {
		return lconf.MsSecC{}, err
	}
	return lconf.MsSecC{SecC: stpt, ID: tls.ID}, nil
}

func (conf *Config) loadLibp2pOpts(ctx *cli.Context) error {

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
		var mc lconf.MsMuxC
		var err error
		switch v {
		case "yamux":
			mc, err = yamuxC()
		case "mplex":
			mc, err = mplexC()
		default:
			return fmt.Errorf("could not recognize mux %s", v)
		}
		if err != nil {
			return fmt.Errorf("failed to make %s constructor: %w", v, err)
		}
		conf.HostMux = append(conf.HostMux, mc)
	}

	secArr := strings.Split(ctx.GlobalString(flags.HostSecurity.Name), ",")
	for _, v := range secArr {
		v = strings.ToLower(strings.TrimSpace(v))
		var sc lconf.MsSecC
		var err error
		switch v {
		case "none": // no security, for debugging etc.
			if len(conf.HostSecurity) > 0 || len(secArr) > 1 {
				return errors.New("cannot mix secure transport protocols with no-security")
			}
			conf.NoTransportSecurity = true
		case "noise":
			sc, err = noiseC()
		case "tls":
			sc, err = tlsC()
		default:
			return fmt.Errorf("could not recognize security %s", v)
		}
		if err != nil {
			return fmt.Errorf("failed to make %s constructor: %w", v, err)
		}
		conf.HostSecurity = append(conf.HostSecurity, sc)
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

func (conf *Config) Check() error {
	if conf.DisableP2P {
		return nil
	}
	if conf.Store == nil {
		return errors.New("p2p requires a persistent or in-memory peerstore, but found none")
	}
	if !conf.NoDiscovery {
		if conf.DiscoveryDB == nil {
			return errors.New("discovery requires a persistent or in-memory discv5 db, but found none")
		}
	}
	if conf.PeersLo == 0 || conf.PeersHi == 0 || conf.PeersLo > conf.PeersHi {
		return fmt.Errorf("peers lo/hi tides are invalid: %d, %d", conf.PeersLo, conf.PeersHi)
	}
	if conf.ConnMngr == nil {
		return errors.New("need a connection manager")
	}
	if conf.ConnGater == nil {
		return errors.New("need a connection gater")
	}
	return nil
}
