package flags

import (
	"time"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
)

func p2pEnv(v string) string {
	return prefixEnvVar("P2P_" + v)
}

var (
	DisableP2P = cli.BoolFlag{
		Name:     "p2p.disable",
		Usage:    "Completely disable the P2P stack",
		Required: false,
		EnvVar:   p2pEnv("DISABLE"),
	}
	NoDiscovery = cli.BoolFlag{
		Name:     "p2p.no-discovery",
		Usage:    "Disable Discv5 (node discovery)",
		Required: false,
		EnvVar:   p2pEnv("NO_DISCOVERY"),
	}
	PeerScoring = cli.StringFlag{
		Name: "p2p.scoring.peers",
		Usage: "Sets the peer scoring strategy for the P2P stack. " +
			"Can be one of: none or light." +
			"Custom scoring strategies can be defined in the config file.",
		Required: false,
		Value:    "none",
		EnvVar:   p2pEnv("PEER_SCORING"),
	}
	PeerScoreBands = cli.StringFlag{
		Name: "p2p.score.bands",
		Usage: "Sets the peer score bands used primarily for peer score metrics. " +
			"Should be provided in following format: <threshold>:<label>;<threshold>:<label>;..." +
			"For example: -40:graylist;-20:restricted;0:nopx;20:friend;",
		Required: false,
		Value:    "-40:<=-40;-10:<=-10;-5:<=-05;-0.05:<=-00.05;0:<=0;0.05:<=00.05;5:<=05;10:<=10;20:<=20;100:>20;",
		EnvVar:   p2pEnv("SCORE_BANDS"),
	}

	// Banning Flag - whether or not we want to act on the scoring
	Banning = cli.BoolFlag{
		Name:     "p2p.ban.peers",
		Usage:    "Enables peer banning. This should ONLY be enabled once certain peer scoring is working correctly.",
		Required: false,
		EnvVar:   p2pEnv("PEER_BANNING"),
	}

	TopicScoring = cli.StringFlag{
		Name: "p2p.scoring.topics",
		Usage: "Sets the topic scoring strategy. " +
			"Can be one of: none or light." +
			"Custom scoring strategies can be defined in the config file.",
		Required: false,
		Value:    "none",
		EnvVar:   p2pEnv("TOPIC_SCORING"),
	}
	P2PPrivPath = cli.StringFlag{
		Name: "p2p.priv.path",
		Usage: "Read the hex-encoded 32-byte private key for the peer ID from this txt file. Created if not already exists." +
			"Important to persist to keep the same network identity after restarting, maintaining the previous advertised identity.",
		Required:  false,
		Value:     "opnode_p2p_priv.txt",
		EnvVar:    p2pEnv("PRIV_PATH"),
		TakesFile: true,
	}
	P2PPrivRaw = cli.StringFlag{
		// sometimes it may be ok to not persist the peer priv key as file, and instead pass it directly.
		Name:     "p2p.priv.raw",
		Usage:    "The hex-encoded 32-byte private key for the peer ID",
		Required: false,
		Hidden:   true,
		Value:    "",
		EnvVar:   p2pEnv("PRIV_RAW"),
	}
	ListenIP = cli.StringFlag{
		Name:     "p2p.listen.ip",
		Usage:    "IP to bind LibP2P and Discv5 to",
		Required: false,
		Value:    "0.0.0.0",
		EnvVar:   p2pEnv("LISTEN_IP"),
	}
	ListenTCPPort = cli.UintFlag{
		Name:     "p2p.listen.tcp",
		Usage:    "TCP port to bind LibP2P to. Any available system port if set to 0.",
		Required: false,
		Value:    9222,
		EnvVar:   p2pEnv("LISTEN_TCP_PORT"),
	}
	ListenUDPPort = cli.UintFlag{
		Name:     "p2p.listen.udp",
		Usage:    "UDP port to bind Discv5 to. Same as TCP port if left 0.",
		Required: false,
		Value:    0, // can simply match the TCP libp2p port
		EnvVar:   p2pEnv("LISTEN_UDP_PORT"),
	}
	AdvertiseIP = cli.StringFlag{
		Name:     "p2p.advertise.ip",
		Usage:    "The IP address to advertise in Discv5, put into the ENR of the node. This may also be a hostname / domain name to resolve to an IP.",
		Required: false,
		// Ignored by default, nodes can discover their own external IP in the happy case,
		// by communicating with bootnodes. Fixed IP is recommended for faster bootstrap though.
		Value:  "",
		EnvVar: p2pEnv("ADVERTISE_IP"),
	}
	AdvertiseTCPPort = cli.UintFlag{
		Name:     "p2p.advertise.tcp",
		Usage:    "The TCP port to advertise in Discv5, put into the ENR of the node. Set to p2p.listen.tcp value if 0.",
		Required: false,
		Value:    0,
		EnvVar:   p2pEnv("ADVERTISE_TCP"),
	}
	AdvertiseUDPPort = cli.UintFlag{
		Name:     "p2p.advertise.udp",
		Usage:    "The UDP port to advertise in Discv5 as fallback if not determined by Discv5, put into the ENR of the node. Set to p2p.listen.udp value if 0.",
		Required: false,
		Value:    0,
		EnvVar:   p2pEnv("ADVERTISE_UDP"),
	}
	Bootnodes = cli.StringFlag{
		Name:     "p2p.bootnodes",
		Usage:    "Comma-separated base64-format ENR list. Bootnodes to start discovering other node records from.",
		Required: false,
		Value:    "",
		EnvVar:   p2pEnv("BOOTNODES"),
	}
	StaticPeers = cli.StringFlag{
		Name:     "p2p.static",
		Usage:    "Comma-separated multiaddr-format peer list. Static connections to make and maintain, these peers will be regarded as trusted.",
		Required: false,
		Value:    "",
		EnvVar:   p2pEnv("STATIC"),
	}
	HostMux = cli.StringFlag{
		Name:     "p2p.mux",
		Usage:    "Comma-separated list of multiplexing protocols in order of preference. At least 1 required. Options: 'yamux','mplex'.",
		Hidden:   true,
		Required: false,
		Value:    "yamux,mplex",
		EnvVar:   p2pEnv("MUX"),
	}
	HostSecurity = cli.StringFlag{
		Name:     "p2p.security",
		Usage:    "Comma-separated list of transport security protocols in order of preference. At least 1 required. Options: 'noise','tls'. Set to 'none' to disable.",
		Hidden:   true,
		Required: false,
		Value:    "noise",
		EnvVar:   p2pEnv("SECURITY"),
	}
	PeersLo = cli.UintFlag{
		Name:     "p2p.peers.lo",
		Usage:    "Low-tide peer count. The node actively searches for new peer connections if below this amount.",
		Required: false,
		Value:    20,
		EnvVar:   p2pEnv("PEERS_LO"),
	}
	PeersHi = cli.UintFlag{
		Name:     "p2p.peers.hi",
		Usage:    "High-tide peer count. The node starts pruning peer connections slowly after reaching this number.",
		Required: false,
		Value:    30,
		EnvVar:   p2pEnv("PEERS_HI"),
	}
	PeersGrace = cli.DurationFlag{
		Name:     "p2p.peers.grace",
		Usage:    "Grace period to keep a newly connected peer around, if it is not misbehaving.",
		Required: false,
		Value:    30 * time.Second,
		EnvVar:   p2pEnv("PEERS_GRACE"),
	}
	NAT = cli.BoolFlag{
		Name:     "p2p.nat",
		Usage:    "Enable NAT traversal with PMP/UPNP devices to learn external IP.",
		Required: false,
		EnvVar:   p2pEnv("NAT"),
	}
	UserAgent = cli.StringFlag{
		Name:     "p2p.useragent",
		Usage:    "User-agent string to share via LibP2P identify. If empty it defaults to 'optimism'.",
		Hidden:   true,
		Required: false,
		Value:    "optimism",
		EnvVar:   p2pEnv("AGENT"),
	}
	TimeoutNegotiation = cli.DurationFlag{
		Name:     "p2p.timeout.negotiation",
		Usage:    "Negotiation timeout, time for new peer connections to share their their supported p2p protocols",
		Hidden:   true,
		Required: false,
		Value:    10 * time.Second,
		EnvVar:   p2pEnv("TIMEOUT_NEGOTIATION"),
	}
	TimeoutAccept = cli.DurationFlag{
		Name:     "p2p.timeout.accept",
		Usage:    "Accept timeout, time for connection to be accepted.",
		Hidden:   true,
		Required: false,
		Value:    10 * time.Second,
		EnvVar:   p2pEnv("TIMEOUT_ACCEPT"),
	}
	TimeoutDial = cli.DurationFlag{
		Name:     "p2p.timeout.dial",
		Usage:    "Dial timeout for outgoing connection requests",
		Hidden:   true,
		Required: false,
		Value:    10 * time.Second,
		EnvVar:   p2pEnv("TIMEOUT_DIAL"),
	}
	PeerstorePath = cli.StringFlag{
		Name: "p2p.peerstore.path",
		Usage: "Peerstore database location. Persisted peerstores help recover peers after restarts. " +
			"Set to 'memory' to never persist the peerstore. Peerstore records will be pruned / expire as necessary. " +
			"Warning: a copy of the priv network key of the local peer will be persisted here.", // TODO: bad design of libp2p, maybe we can avoid this from happening
		Required:  false,
		TakesFile: true,
		Value:     "opnode_peerstore_db",
		EnvVar:    p2pEnv("PEERSTORE_PATH"),
	}
	DiscoveryPath = cli.StringFlag{
		Name:      "p2p.discovery.path",
		Usage:     "Discovered ENRs are persisted in a database to recover from a restart without having to bootstrap the discovery process again. Set to 'memory' to never persist the peerstore.",
		Required:  false,
		TakesFile: true,
		Value:     "opnode_discovery_db",
		EnvVar:    p2pEnv("DISCOVERY_PATH"),
	}
	SequencerP2PKeyFlag = cli.StringFlag{
		Name:     "p2p.sequencer.key",
		Usage:    "Hex-encoded private key for signing off on p2p application messages as sequencer.",
		Required: false,
		Value:    "",
		EnvVar:   p2pEnv("SEQUENCER_KEY"),
	}
	GossipMeshDFlag = cli.UintFlag{
		Name:     "p2p.gossip.mesh.d",
		Usage:    "Configure GossipSub topic stable mesh target count, a.k.a. desired outbound degree, number of peers to gossip to",
		Required: false,
		Hidden:   true,
		Value:    p2p.DefaultMeshD,
		EnvVar:   p2pEnv("GOSSIP_MESH_D"),
	}
	GossipMeshDloFlag = cli.UintFlag{
		Name:     "p2p.gossip.mesh.lo",
		Usage:    "Configure GossipSub topic stable mesh low watermark, a.k.a. lower bound of outbound degree",
		Required: false,
		Hidden:   true,
		Value:    p2p.DefaultMeshDlo,
		EnvVar:   p2pEnv("GOSSIP_MESH_DLO"),
	}
	GossipMeshDhiFlag = cli.UintFlag{
		Name:     "p2p.gossip.mesh.dhi",
		Usage:    "Configure GossipSub topic stable mesh high watermark, a.k.a. upper bound of outbound degree, additional peers will not receive gossip",
		Required: false,
		Hidden:   true,
		Value:    p2p.DefaultMeshDhi,
		EnvVar:   p2pEnv("GOSSIP_MESH_DHI"),
	}
	GossipMeshDlazyFlag = cli.UintFlag{
		Name:     "p2p.gossip.mesh.dlazy",
		Usage:    "Configure GossipSub gossip target, a.k.a. target degree for gossip only (not messaging like p2p.gossip.mesh.d, just announcements of IHAVE",
		Required: false,
		Hidden:   true,
		Value:    p2p.DefaultMeshDlazy,
		EnvVar:   p2pEnv("GOSSIP_MESH_DLAZY"),
	}
	GossipFloodPublishFlag = cli.BoolFlag{
		Name:     "p2p.gossip.mesh.floodpublish",
		Usage:    "Configure GossipSub to publish messages to all known peers on the topic, outside of the mesh, also see Dlazy as less aggressive alternative.",
		Required: false,
		Hidden:   true,
		EnvVar:   p2pEnv("GOSSIP_FLOOD_PUBLISH"),
	}
	SyncReqRespFlag = cli.BoolFlag{
		Name:     "p2p.sync.req-resp",
		Usage:    "Enables experimental P2P req-resp alternative sync method, on both server and client side.",
		Required: false,
		EnvVar:   p2pEnv("SYNC_REQ_RESP"),
	}
)

// None of these flags are strictly required.
// Some are hidden if they are too technical, or not recommended.
var p2pFlags = []cli.Flag{
	DisableP2P,
	NoDiscovery,
	P2PPrivPath,
	P2PPrivRaw,
	PeerScoring,
	PeerScoreBands,
	Banning,
	TopicScoring,
	ListenIP,
	ListenTCPPort,
	ListenUDPPort,
	AdvertiseIP,
	AdvertiseTCPPort,
	AdvertiseUDPPort,
	Bootnodes,
	StaticPeers,
	HostMux,
	HostSecurity,
	PeersLo,
	PeersHi,
	PeersGrace,
	NAT,
	UserAgent,
	TimeoutNegotiation,
	TimeoutAccept,
	TimeoutDial,
	PeerstorePath,
	DiscoveryPath,
	SequencerP2PKeyFlag,
	GossipMeshDFlag,
	GossipMeshDloFlag,
	GossipMeshDhiFlag,
	GossipMeshDlazyFlag,
	GossipFloodPublishFlag,
	SyncReqRespFlag,
}
