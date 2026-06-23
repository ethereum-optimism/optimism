package sysgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
}

// opConP2PSidecarBinaryEnv names the env var pointing at a prebuilt op-conp2p
// binary. op-conp2p is in-repo Go (built with `go build ./op-conp2p`), so unlike
// the rust binaries it is located by path rather than via rustbin.
const opConP2PSidecarBinaryEnv = "OP_CONP2P_BIN"

// startOpConP2PSidecar spawns op-conp2p, static-peered to the sequencer's gossip
// address, delegating verdicts to the verifier at nodeUserRPC.
//
//   - nodeUserRPC: the verifier's JSON-RPC endpoint (exposes admin_verifyUnsafePayload).
//   - rollupConfigPath: the standard op-node rollup.json (chain id + fork times
//     drive the gossip topic selection); the same file the verifier consumes.
//   - sequencerGossipMultiaddr: the sequencer CL's gossip multiaddr (from its
//     opp2p_self), used as a static peer since the sidecar runs no discovery.
func StartOpConP2PSidecar(
	t devtest.T,
	name string,
	nodeUserRPC string,
	rollupConfigPath string,
	sequencerGossipMultiaddr string,
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
		"--p2p.static", sequencerGossipMultiaddr,
		"--p2p.no-discovery=true",
		"--p2p.listen.ip", "127.0.0.1",
		"--p2p.listen.tcp", tcpPort,
		"--p2p.listen.udp", udpPort,
		"--p2p.priv.path", filepath.Join(dir, "p2p_priv.txt"),
		"--p2p.peerstore.path", "memory",
		"--p2p.discovery.path", "memory",
	}

	logOut := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "op-conp2p", "src", "stdout"), log.LevelInfo)
	logErr := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "op-conp2p", "src", "stderr"), log.LevelInfo)
	sub := NewSubProcess(t,
		logpipe.LogCallback(func(line []byte) { logOut(logpipe.ParseGoStructuredLogs(line)) }),
		logpipe.LogCallback(func(line []byte) { logErr(logpipe.ParseGoStructuredLogs(line)) }),
	)

	t.Logger().Info("Starting op-conp2p sidecar", "name", name, "node_rpc", nodeUserRPC, "static_peer", sequencerGossipMultiaddr)
	t.Require().NoError(sub.Start(bin, args, nil), "must start op-conp2p sidecar")

	s := &OpConP2PSidecar{name: name, p: t, sub: sub}
	t.Cleanup(s.Stop)
	return s
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
