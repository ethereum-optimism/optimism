package sysgo

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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
// emits once config validation and provider construction have completed; the
// launcher matches it to detect startup.
const (
	zkProposerReadyMessage = "kona-sp1-proposer started"
	konaSP1ELFDirEnv       = "KONA_SP1_ELF_DIR"
)

func loadZKProgramVKey(elfDir string) (common.Hash, error) {
	if elfDir == "" {
		return crypto.Keccak256Hash([]byte("kona-sp1-stub-super-aggregation-vkey")), nil
	}

	var vkeys map[string]string
	if _, err := toml.DecodeFile(filepath.Join(elfDir, "vkeys.toml"), &vkeys); err != nil {
		return common.Hash{}, fmt.Errorf("read Kona SP1 vkeys: %w", err)
	}
	raw, ok := vkeys["super-aggregation"]
	if !ok {
		return common.Hash{}, fmt.Errorf("vkeys.toml does not contain super-aggregation")
	}
	vkeyBytes, err := hexutil.Decode(raw)
	if err != nil {
		return common.Hash{}, fmt.Errorf("decode super-aggregation vkey: %w", err)
	}
	if len(vkeyBytes) != common.HashLength {
		return common.Hash{}, fmt.Errorf("super-aggregation vkey must encode exactly %d bytes", common.HashLength)
	}
	vkey := common.BytesToHash(vkeyBytes)
	if vkey == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("super-aggregation vkey must not be zero")
	}
	return vkey, nil
}

type zkProposerConfig struct {
	ProposalInterval *time.Duration
}

// ZKProposerOption configures the kona-sp1-proposer process started by
// devstack.
type ZKProposerOption func(cfg *zkProposerConfig)

// WithZKProposalInterval overrides the proposal interval passed to
// kona-sp1-proposer.
func WithZKProposalInterval(interval time.Duration) ZKProposerOption {
	return func(cfg *zkProposerConfig) {
		cfg.ProposalInterval = &interval
	}
}

func newZKProposerConfig(opts ...ZKProposerOption) (zkProposerConfig, error) {
	var cfg zkProposerConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.ProposalInterval != nil {
		if *cfg.ProposalInterval <= 0 {
			return zkProposerConfig{}, fmt.Errorf("ZK proposer interval must be positive")
		}
		if *cfg.ProposalInterval%time.Second != 0 {
			return zkProposerConfig{}, fmt.Errorf("ZK proposer interval must use whole seconds")
		}
	}
	return cfg, nil
}

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
	elfDir string,
	proposerOpts ...ZKProposerOption,
) {
	require := t.Require()
	cfg, err := newZKProposerConfig(proposerOpts...)
	require.NoError(err, "invalid ZK proposer config")

	proposerSecret, err := keys.Secret(devkeys.ProposerRole.Key(proposerChainID.ToBig()))
	require.NoError(err, "need proposer key for ZK proposer")

	execPath, err := rustbin.Spec{
		SrcDir:  "rust/kona",
		Package: "kona-sp1-proposer",
		Binary:  "kona-sp1-proposer",
	}.EnsureExists(t.Ctx(), t.Logger())
	require.NoError(err, "prepare kona-sp1-proposer binary")
	require.NotEmpty(execPath, "kona-sp1-proposer binary path resolved")

	// The proposer checks (and, with the defend path, loads) program
	// artifacts at PRESTATES_URL/<vkey>.agg.bin.gz and .range.bin.gz,
	// mirroring op-challenger's --prestates-url convention. The aggregation
	// vkey embeds the range program's vkey, so it keys both ELFs. Devstack
	// publishes the real ELFs, gzipped, when KONA_SP1_ELF_DIR is set;
	// otherwise stub bytes suffice, since the devstack deploys the mock
	// verifier and the create path only loads artifacts without verifying
	// them against the vkey. Either way the file:// path is exercised.
	prestatesDir := t.TempDir()
	sources := map[string]func() (io.ReadCloser, error){
		".agg.bin.gz":   elfSource(elfDir, "super-aggregation-elf"),
		".range.bin.gz": elfSource(elfDir, "super-range-elf"),
	}
	for suffix, open := range sources {
		writePrestateArtifact(t, open, filepath.Join(prestatesDir, programVKey.Hex()+suffix))
	}

	env := []string{
		"L1_RPC=" + l1EL.UserRPC(),
		"SUPERNODE_RPC=" + supernodeRPC,
		"FACTORY_ADDRESS=" + factoryAddr.Hex(),
		"PRESTATES_URL=file://" + prestatesDir,
		"PRIVATE_KEY=" + hexutil.Encode(crypto.FromECDSA(proposerSecret)),
		"PROPOSAL_SAFETY=safe",
		"FETCH_INTERVAL=2",
		"LOG_FORMAT=json",
	}
	if cfg.ProposalInterval != nil {
		env = append(env, "PROPOSAL_INTERVAL_SECONDS="+strconv.FormatUint(uint64(*cfg.ProposalInterval/time.Second), 10))
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
