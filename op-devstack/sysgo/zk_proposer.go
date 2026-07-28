package sysgo

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// zkProposerReadyMessage is the structured tracing message kona-sp1-proposer
// emits once config validation, provider construction, and the metrics bind
// have completed. When metrics are enabled the same entry carries a
// `metrics_addr` field with the bound host:port.
const zkProposerReadyMessage = "kona-sp1-proposer started"

// startZKProposer launches the Rust kona-sp1-proposer binary against the ZK
// dispute game type. Modeled on the kona-node launcher (l2_cl_kona.go), minus
// the RPC proxy: the proposer serves no HTTP API. The proposer is configured
// exclusively through environment variables.
func startZKProposer(
	t devtest.T,
	keys devkeys.Keys,
	proposerChainID eth.ChainID,
	l1EL L1ELNode,
	supernodeRPC string,
	factoryAddr common.Address,
	programVKey common.Hash,
) {
	require := t.Require()

	proposerSecret, err := keys.Secret(devkeys.ProposerRole.Key(proposerChainID.ToBig()))
	require.NoError(err, "need proposer key for ZK proposer")

	execPath, err := rustbin.Spec{
		SrcDir:  "rust/kona",
		Package: "kona-sp1-proposer",
		Binary:  "kona-sp1-proposer",
	}.EnsureExists(t.Ctx(), t.Logger())
	require.NoError(err, "prepare kona-sp1-proposer binary")
	require.NotEmpty(execPath, "kona-sp1-proposer binary path resolved")

	// The proposer loads its known prestate set from a TOML document (the
	// vkeys.toml shape) via file:// or http(s)://. Devstack exercises the
	// file:// path with a document holding the deployed vkey.
	prestatesPath := filepath.Join(t.TempDir(), "prestates.toml")
	require.NoError(
		os.WriteFile(prestatesPath, fmt.Appendf(nil, "super-aggregation = %q\n", programVKey.Hex()), 0o644),
		"write prestates document",
	)

	env := []string{
		"L1_RPC=" + l1EL.UserRPC(),
		"SUPERNODE_RPC=" + supernodeRPC,
		"FACTORY_ADDRESS=" + factoryAddr.Hex(),
		"PRESTATES_URL=file://" + prestatesPath,
		"PRIVATE_KEY=" + hexutil.Encode(crypto.FromECDSA(proposerSecret)),
		// Short cadence for devstack: propose every 6s off the safe head so
		// acceptance tests observe games without waiting for finality.
		"PROPOSAL_INTERVAL_SECONDS=6",
		"PROPOSAL_SAFETY=safe",
		"FETCH_INTERVAL=2",
		"LOG_FORMAT=json",
	}
	// METRICS_PORT omitted (or 0) disables the metrics server entirely.
	if areMetricsEnabled() {
		env = append(env, "METRICS_PORT="+strconv.Itoa(freeTCPPort(t)))
	}

	logOut := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "kona-sp1-proposer", "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(t.Logger().New("component", "kona-sp1-proposer", "src", "stderr"), log.LevelWarn)
	startedChan := make(chan logpipe.LogEntry, 1)

	onLogEntry := func(e logpipe.LogEntry) {
		if strings.Contains(e.LogMessage(), zkProposerReadyMessage) {
			select {
			case startedChan <- e:
			default:
			}
		}
	}
	stdOutLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
		onLogEntry(e)
	})
	sub := NewSubProcess(t, stdOutLogs, stdErrLogs)

	require.NoError(sub.Start(execPath, nil, env), "must start kona-sp1-proposer")

	// Wait for the startup line, but fail fast if the process exits first
	// (e.g. a crash on boot) rather than blocking until the test times out.
	var started logpipe.LogEntry
	select {
	case started = <-startedChan:
	case <-sub.Exited():
		select {
		case started = <-startedChan:
		default:
			require.FailNow("kona-sp1-proposer exited before its startup line was emitted")
		}
	case <-t.Ctx().Done():
		require.NoError(t.Ctx().Err(), "need kona-sp1-proposer startup")
	}

	if areMetricsEnabled() {
		metricsAddr, ok := started.FieldValue("metrics_addr").(string)
		require.True(ok, "kona-sp1-proposer startup line must include metrics_addr when metrics are enabled")
		host, port, err := net.SplitHostPort(metricsAddr)
		require.NoError(err, "invalid metrics_addr in startup line: %s", metricsAddr)
		target := NewPrometheusMetricsTarget(host, port, false)
		t.Logger().Info("kona-sp1-proposer metrics endpoint ready", "target", target)
	}

	t.Logger().Info("kona-sp1-proposer is up", "chain", proposerChainID, "factory", factoryAddr)
}

// freeTCPPort reserves an OS-assigned TCP port and releases it for immediate
// reuse by the subprocess. The proposer takes its metrics port via env, so it
// cannot self-assign with port 0 (0 means metrics disabled).
func freeTCPPort(t devtest.T) int {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	t.Require().NoError(err, "reserve metrics port")
	port := lis.Addr().(*net.TCPAddr).Port
	t.Require().NoError(lis.Close())
	return port
}
