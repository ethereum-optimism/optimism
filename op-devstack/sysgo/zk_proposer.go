package sysgo

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// zkProposerReadyMessage is the structured tracing message kona-sp1-proposer
// emits once config validation and provider construction have completed; the
// launcher matches it to detect startup.
const zkProposerReadyMessage = "kona-sp1-proposer started"

// startZKProposer launches the Rust kona-sp1-proposer binary against the ZK
// dispute game type. Modeled on the kona-node launcher (l2_cl_kona.go), minus
// the RPC proxy: the proposer serves no HTTP API. The proposer is configured
// exclusively through environment variables.
func startZKProposer(
	t devtest.T,
	keys devkeys.Keys,
	proposerChainID eth.ChainID,
	l1Net *L1Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	depSet depset.DependencySet,
	supernodeRPC string,
	l2Nets []*L2Network,
	l2ELs []L2ELNode,
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

	// The proposer checks and loads program artifacts at
	// PRESTATES_URL/<vkey>.agg.bin.gz and .range.bin.gz, mirroring
	// op-challenger's --prestates-url convention. The aggregation vkey embeds
	// the range program's vkey, so it keys both ELFs. Devstack publishes the
	// real ELFs, gzipped, when KONA_SP1_ELF_DIR is set; otherwise stub bytes
	// suffice, since the devstack deploys the mock verifier and the mock
	// proof provider never executes the ELFs and skips vkey verification.
	// Either way the file:// path is exercised.
	prestatesDir := t.TempDir()
	elfDir := os.Getenv("KONA_SP1_ELF_DIR")
	sources := map[string]func() (io.ReadCloser, error){
		".agg.bin.gz":   elfSource(elfDir, "super-aggregation-elf"),
		".range.bin.gz": elfSource(elfDir, "super-range-elf"),
	}
	for suffix, open := range sources {
		writePrestateArtifact(t, open, filepath.Join(prestatesDir, programVKey.Hex()+suffix))
	}

	// The defend path collects derivation witnesses through the interop host,
	// which needs the L1 beacon, every L2 EL, and the rollup/depset/L1 chain
	// configs. Materialize the config files with the same inline
	// marshal+WriteFile the kona-node launcher uses (startMixedKonaNode),
	// with per-chain rollup-<chainID>.json naming as in the shared
	// challenger's VM config.
	require.Len(l2ELs, len(l2Nets), "need matching L2 ELs for the ZK proposer")
	configDir := t.TempDir()
	l2RPCs := make([]string, len(l2ELs))
	rollupPaths := make([]string, len(l2Nets))
	for i, l2Net := range l2Nets {
		l2RPCs[i] = l2ELs[i].UserRPC()
		rollupData, err := json.Marshal(l2Net.rollupCfg)
		require.NoError(err, "must marshal rollup config")
		rollupPath := filepath.Join(configDir, fmt.Sprintf("rollup-%v.json", l2Net.ChainID()))
		require.NoError(os.WriteFile(rollupPath, rollupData, 0o640), "must write rollup config")
		rollupPaths[i] = rollupPath
	}

	l1CfgPath := filepath.Join(configDir, "l1-chain-config.json")
	l1CfgData, err := json.Marshal(l1Net.genesis.Config)
	require.NoError(err, "must marshal l1 chain config")
	require.NoError(os.WriteFile(l1CfgPath, l1CfgData, 0o640), "must write l1 chain config")

	depSetPath := filepath.Join(configDir, "interop-depset.json")
	depSetData, err := json.Marshal(depSet)
	require.NoError(err, "must marshal interop dependency set")
	require.NoError(os.WriteFile(depSetPath, depSetData, 0o640), "must write interop dependency set")

	env := []string{
		"L1_RPC=" + l1EL.UserRPC(),
		"SUPERNODE_RPC=" + supernodeRPC,
		"FACTORY_ADDRESS=" + factoryAddr.Hex(),
		"PRESTATES_URL=file://" + prestatesDir,
		"PRIVATE_KEY=" + hexutil.Encode(crypto.FromECDSA(proposerSecret)),
		// The mock provider skips SP1 proving but still runs the full witness
		// pipeline against these endpoints and configs.
		"PROOF_PROVIDER=mock",
		"L1_BEACON_RPC=" + l1CL.beaconHTTPAddr,
		// Order-irrelevant: kona-host keys L2 providers by queried eth_chainId.
		"L2_RPCS=" + strings.Join(l2RPCs, ","),
		"ROLLUP_CONFIG_PATHS=" + strings.Join(rollupPaths, ","),
		"L1_CONFIG_PATH=" + l1CfgPath,
		"DEPENDENCY_SET_PATH=" + depSetPath,
		// Short cadence for devstack: propose every 6s off the safe head so
		// acceptance tests observe games without waiting for finality.
		"PROPOSAL_INTERVAL_SECONDS=6",
		"PROPOSAL_SAFETY=safe",
		"FETCH_INTERVAL=2",
		"LOG_FORMAT=json",
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
	select {
	case <-startedChan:
	case <-sub.Exited():
		select {
		case <-startedChan:
		default:
			require.FailNow("kona-sp1-proposer exited before its startup line was emitted")
		}
	case <-t.Ctx().Done():
		require.NoError(t.Ctx().Err(), "need kona-sp1-proposer startup")
	}

	t.Logger().Info("kona-sp1-proposer is up", "chain", proposerChainID, "factory", factoryAddr)
}

// elfSource returns an opener for the program ELF in elfDir, or for
// deterministic stub bytes when elfDir is empty (no SP1 guest ELF build
// available).
func elfSource(elfDir, name string) func() (io.ReadCloser, error) {
	if elfDir == "" {
		return func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("stub " + name)), nil
		}
	}
	return func() (io.ReadCloser, error) {
		// os.Root confines the open to elfDir regardless of what the path
		// segments contain, making non-traversal a property of the code.
		root, err := os.OpenRoot(elfDir)
		if err != nil {
			return nil, err
		}
		defer root.Close()
		return root.Open(name)
	}
}

// writePrestateArtifact gzips the program ELF from open into dst, one of the
// `<vkey>.agg.bin.gz` / `<vkey>.range.bin.gz` artifacts the proposer's
// prestate check looks for.
func writePrestateArtifact(t devtest.T, open func() (io.ReadCloser, error), dst string) {
	require := t.Require()
	in, err := open()
	require.NoError(err, "open program ELF")
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	require.NoError(err, "create prestate artifact")
	defer out.Close()
	gz := gzip.NewWriter(out)
	_, err = io.Copy(gz, in)
	require.NoError(err, "gzip program ELF")
	require.NoError(gz.Close(), "finalize prestate artifact")
}
