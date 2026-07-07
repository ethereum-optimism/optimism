package sysgo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// OpConP2PSidecar is the op-conp2p gossip sidecar fronting an op-con-node /
// op-con-ex-node verifier. The verifier has no built-in P2P; the sidecar joins
// the OP gossip network, receives unsafe payloads, and delegates each block's
// signature verdict back to the verifier over JSON-RPC (admin_verifyUnsafePayload).
//
// This is the launcher half of docs/sidecar-p2p-design.md phase 2. It is
// parameterized (it takes the verifier RPC, the rollup config path, and the
// sequencer's gossip multiaddr) so a with-P2P preset can wire it in place of the
// follow-mode (`--l2-follow-rpc`) source. Full preset + acceptance-test wiring is
// tracked as the remaining run-verification step.
type OpConP2PSidecar struct {
	name string
	p    devtest.T
	sub  *SubProcess

	// listenHost/listenTCP are the gossip listen address the launcher assigned;
	// combined with the peer id (scraped from the startup log) they form the
	// sidecar's own dialable multiaddr — see GossipMultiaddr. Needed to peer two
	// sidecars directly (publish-side ↔ receive-side), since a sidecar exposes no
	// opp2p RPC to query its own address.
	listenHost string
	listenTCP  string
	peerIDCh   chan string
}

// opConP2PPeerIDRe extracts the libp2p peer id from op-conp2p's Info startup log
// line ("op-conp2p sidecar started ... peer_id=16Uiu2...") after ANSI color
// codes are stripped (the sidecar logs in color, so the key/value are wrapped in
// escape sequences: "\x1b[32mpeer_id\x1b[0m=16Uiu2...").
var (
	opConP2PPeerIDRe = regexp.MustCompile(`peer_id=([0-9A-Za-z]+)`)
	ansiEscapeRe     = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// opConP2PSidecarBinaryEnv names the env var pointing at a prebuilt op-conp2p
// binary. op-conp2p is in-repo Go (built with `go build ./op-conp2p`), so unlike
// the rust binaries it is located by path rather than via rustbin.
const opConP2PSidecarBinaryEnv = "OP_CONP2P_BIN"

// OpConP2PSidecarOption customizes the sidecar's command line.
type OpConP2PSidecarOption func(args []string) []string

// WithSignedPayloadWS enables the sidecar's PUBLISH path: subscribe to a
// sequencing op-con-node's signed-payload websocket at wsURL and publish each
// node-signed unsafe block to the OP gossip /blocks topics.
func WithSignedPayloadWS(wsURL string) OpConP2PSidecarOption {
	return func(args []string) []string {
		return append(args, "--signed-payload-ws", wsURL)
	}
}

// WithFloodPublish makes the sidecar's gossipsub publish to all known peers on
// the topic instead of only the mesh. With a single static peer and no
// discovery, this removes the mesh-formation (GRAFT) delay so the very first
// published block reaches the peer.
func WithFloodPublish() OpConP2PSidecarOption {
	return func(args []string) []string {
		return append(args, "--p2p.gossip.mesh.floodpublish=true")
	}
}

// startOpConP2PSidecar spawns op-conp2p, static-peered to the sequencer's gossip
// address, delegating verdicts to the verifier at nodeUserRPC.
//
//   - nodeUserRPC: the verifier's JSON-RPC endpoint (exposes admin_verifyUnsafePayload).
//   - rollupConfigPath: the standard op-node rollup.json (chain id + fork times
//     drive the gossip topic selection); the same file the verifier consumes.
//   - staticPeerMultiaddr: a gossip multiaddr (from an op-node's opp2p_self, or
//     another sidecar's GossipMultiaddr) the sidecar dials as a static peer,
//     since it runs no discovery. On the receive path this is the block source;
//     on the publish path, the receiving verifier(s). Comma-separated for
//     multiple peers. Empty means "dial nobody" — the sidecar only accepts
//     inbound connections (e.g. a receive sidecar dialed by the publish side).
func StartOpConP2PSidecar(
	t devtest.T,
	name string,
	nodeUserRPC string,
	rollupConfigPath string,
	staticPeerMultiaddr string,
	opts ...OpConP2PSidecarOption,
) *OpConP2PSidecar {
	bin := os.Getenv(opConP2PSidecarBinaryEnv)
	t.Require().NotEmpty(bin, "set %s to the op-conp2p binary path (go build ./op-conp2p)", opConP2PSidecarBinaryEnv)

	dir := t.TempDir()
	tcpPort, err := getAvailableLocalPort()
	t.Require().NoError(err, "op-conp2p tcp port")
	udpPort, err := getAvailableLocalPort()
	t.Require().NoError(err, "op-conp2p udp port")

	args := []string{
		"--rollup.config", rollupConfigPath,
		"--node.rpc", nodeUserRPC,
		"--p2p.no-discovery=true",
		"--p2p.listen.ip", "127.0.0.1",
		"--p2p.listen.tcp", tcpPort,
		"--p2p.listen.udp", udpPort,
		"--p2p.priv.path", filepath.Join(dir, "p2p_priv.txt"),
		"--p2p.peerstore.path", "memory",
		"--p2p.discovery.path", "memory",
	}
	// An empty static peer means the sidecar only accepts inbound connections;
	// op-node's --p2p.static rejects an empty value, so omit the flag entirely.
	if staticPeerMultiaddr != "" {
		args = append(args, "--p2p.static", staticPeerMultiaddr)
	}
	for _, opt := range opts {
		args = opt(args)
	}

	s := &OpConP2PSidecar{
		name:       name,
		p:          t,
		listenHost: "127.0.0.1",
		listenTCP:  tcpPort,
		peerIDCh:   make(chan string, 1),
	}

	logOut := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "op-conp2p", "src", "stdout"), log.LevelInfo)
	logErr := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "op-conp2p", "src", "stderr"), log.LevelInfo)
	// The sidecar logs its own peer id once at startup (Info, on stderr). Tee the
	// raw stderr lines to capture it so GossipMultiaddr can hand this sidecar's
	// dialable address to a peer sidecar.
	stdErr := logpipe.LogCallback(func(line []byte) {
		if m := opConP2PPeerIDRe.FindSubmatch(ansiEscapeRe.ReplaceAll(line, nil)); m != nil {
			select {
			case s.peerIDCh <- string(m[1]):
			default:
			}
		}
		logErr(logpipe.ParseGoStructuredLogs(line))
	})
	sub := NewSubProcess(t,
		logpipe.LogCallback(func(line []byte) { logOut(logpipe.ParseGoStructuredLogs(line)) }),
		stdErr,
	)
	s.sub = sub

	t.Logger().Info("Starting op-conp2p sidecar", "name", name, "node_rpc", nodeUserRPC, "static_peer", staticPeerMultiaddr)
	t.Require().NoError(sub.Start(bin, args, nil), "must start op-conp2p sidecar")

	t.Cleanup(s.Stop)
	return s
}

// GossipMultiaddr returns this sidecar's own dialable gossip multiaddr
// (/ip4/<host>/tcp/<port>/p2p/<peerID>), blocking until the peer id has been
// observed in the startup log. Used to peer a publish sidecar directly to a
// receive sidecar, since neither op-con-node has a built-in gossip stack to
// dial and the sidecar exposes no opp2p RPC.
func (s *OpConP2PSidecar) GossipMultiaddr() string {
	ctx, cancel := context.WithTimeout(s.p.Ctx(), 30*time.Second)
	defer cancel()
	select {
	case peerID := <-s.peerIDCh:
		// Put it back so a second caller (or a reconnect) still sees it.
		select {
		case s.peerIDCh <- peerID:
		default:
		}
		return fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", s.listenHost, s.listenTCP, peerID)
	case <-ctx.Done():
		s.p.Require().Fail("op-conp2p sidecar never logged its peer id", "sidecar %s", s.name)
		return ""
	}
}

func (s *OpConP2PSidecar) Stop() {
	if s.sub == nil {
		return
	}
	if err := s.sub.Stop(true); err != nil {
		s.p.Logger().Warn("op-conp2p sidecar stop error", "err", err)
	}
	s.sub = nil
}

// sequencerGossipMultiaddr fetches the sequencer CL's gossip multiaddr (its
// opp2p_self address) for use as the sidecar's static peer, ensuring the
// /p2p/<peerID> suffix is present.
func SequencerGossipMultiaddr(t devtest.T, sequencerCL L2CLNode) string {
	p2pClient, err := GetP2PClient(t.Ctx(), t.Logger(), sequencerCL)
	t.Require().NoError(err, "p2p client for sequencer CL")
	self, err := GetPeerInfo(t.Ctx(), p2pClient)
	t.Require().NoError(err, "sequencer CL self peer info")
	addr := self.Addresses[0]
	if !strings.Contains(addr, "/p2p/") {
		addr = fmt.Sprintf("%s/p2p/%s", addr, self.PeerID.String())
	}
	return addr
}
