// proofbatch-submitter posts silhouette proof batches to L1 as blob transactions.
//
// A proof batch is what makes a silhouette chain's messages cross-safe on a verifier that does not
// derive it (see op-supernode/silhouette). This tool is the other end of that pipe: it takes an
// envelope — assembled here from a live chain's own outputs, or handed to it as a file — packs it
// into blobs and sends it from the designated submitter address to the designated inbox address.
//
// ATTESTED MODE is v1's whole producer side, and the send is the attestation: the L1 transaction is
// signed by the submitter key, acceptance rule 1 binds every accepted batch to that key, and so
// posting a batch IS the operator putting their name on it. Nothing is proven and nothing pretends to
// be — the proof slot goes out empty and a verifier in attested mode requires it to be. See
// /home/main/op/silhouette/TRUST-MODEL.md.
//
//	proofbatch-submitter --l1-eth-rpc … --private-key … --inbox 0x… --envelope batch.bin
//	proofbatch-submitter --l1-eth-rpc … --private-key … --inbox 0x… \
//	    --attested.rollup-rpc http://p-node:9545 --attested.l2-rpc http://p-reth:8545 \
//	    --attested.rollup-config-hash 0x… --attested.dep-set-hash 0x… \
//	    --attested.interval 10m --attested.cursor cursor.json
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

const envVarPrefix = "PROOFBATCH_SUBMITTER"

var (
	Version   = "v0.0.0"
	GitCommit = ""
	GitDate   = ""
)

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(envVarPrefix, name)
}

var (
	// l1EthRPCFlag is the transaction manager's L1 endpoint. txmgr defines the flag NAME and
	// reads it in ReadCLIConfig, but leaves registering it to the application (op-batcher and
	// op-proposer each declare their own), so a service that only appends txmgr.CLIFlags has an
	// unregistered flag and fails Check() with "must provide a L1 RPC url".
	l1EthRPCFlag = &cli.StringFlag{
		Name:     txmgr.L1RPCFlagName,
		Usage:    "HTTP provider URL for L1. Proof batches are sent here as blob transactions.",
		EnvVars:  prefixEnvVars("L1_ETH_RPC"),
		Required: true,
	}
	inboxFlag = &cli.StringFlag{
		Name:     "inbox",
		Usage:    "L1 address proof batches are sent to. A verifier only accepts envelopes addressed here from the submitter it is configured with.",
		EnvVars:  prefixEnvVars("INBOX"),
		Required: true,
	}
	envelopeFlag = &cli.PathFlag{
		Name:      "envelope",
		Usage:     "Submit the proof-batch envelope in this file once, then exit.",
		EnvVars:   prefixEnvVars("ENVELOPE"),
		TakesFile: true,
	}
	attestedRollupRPCFlag = &cli.StringFlag{
		Name:    "attested.rollup-rpc",
		Usage:   "Attested mode: the chain's op-node RPC, for safe head and output roots.",
		EnvVars: prefixEnvVars("ATTESTED_ROLLUP_RPC"),
	}
	attestedL2RPCFlag = &cli.StringFlag{
		Name:    "attested.l2-rpc",
		Usage:   "Attested mode: the chain's execution RPC, for block timestamps and logs.",
		EnvVars: prefixEnvVars("ATTESTED_L2_RPC"),
	}
	attestedRollupConfigHashFlag = &cli.StringFlag{
		Name:    "attested.rollup-config-hash",
		Usage:   "Attested mode: the rollup-config commitment to put on the wire. Must be the value the verifier is configured with.",
		EnvVars: prefixEnvVars("ATTESTED_ROLLUP_CONFIG_HASH"),
	}
	attestedDepSetHashFlag = &cli.StringFlag{
		Name:    "attested.dep-set-hash",
		Usage:   "Attested mode: the dependency-set commitment to put on the wire. Must be the value the verifier is configured with.",
		EnvVars: prefixEnvVars("ATTESTED_DEP_SET_HASH"),
	}
	attestedIntervalFlag = &cli.DurationFlag{
		Name:    "attested.interval",
		Usage:   "Attested mode: how often to build and submit a batch.",
		EnvVars: prefixEnvVars("ATTESTED_INTERVAL"),
		Value:   10 * time.Minute,
	}
	attestedMaxBlocksFlag = &cli.Uint64Flag{
		Name:    "attested.max-blocks",
		Usage:   "Attested mode: maximum blocks per batch (300 is one 10-minute cadence at 2s blocks).",
		EnvVars: prefixEnvVars("ATTESTED_MAX_BLOCKS"),
		Value:   300,
	}
	attestedL1LagFlag = &cli.Uint64Flag{
		Name:    "attested.l1-lag",
		Usage:   "Attested mode: how far below the L1 head a batch's claimed l1Head sits.",
		EnvVars: prefixEnvVars("ATTESTED_L1_LAG"),
		Value:   8,
	}
	attestedHeadFlag = &cli.StringFlag{
		Name: "attested.head",
		Usage: "Attested mode: which of the chain's own heads bounds a batch — " +
			"\"unsafe\" (silhouette: no batcher, so the safe head is whatever this tool last proved " +
			"and batching on it deadlocks) or \"safe\" (a chain that has a batcher).",
		EnvVars: prefixEnvVars("ATTESTED_HEAD"),
		Value:   HeadUnsafe.String(),
	}
	attestedCursorFlag = &cli.PathFlag{
		Name:    "attested.cursor",
		Usage:   "Attested mode: file holding where the last batch ended, so a restart chains onto its own history.",
		EnvVars: prefixEnvVars("ATTESTED_CURSOR"),
		Value:   "proofbatch-cursor.json",
	}
	// wireVersionFlag has NO DEFAULT, and that is the point.
	//
	// The version this tool posts is a public act: it decides whether every verifier in the dependency
	// set checks the chain's declared imports or trusts them (G7G D5), and a verifier accepts exactly
	// one version. A default would mean that rebuilding this binary after the codec's current version
	// moved would silently change what the live chain publishes — the rotation happening because
	// somebody recompiled. Requiring it makes the version a deployment decision, which is what it is.
	wireVersionFlag = &cli.UintFlag{
		Name: "wire-version",
		Usage: "REQUIRED. Proof-batch envelope version to post (2 = exports only; 3 = exports plus the " +
			"import list the cross-safety judge validates). Must match what every verifier is " +
			"configured to accept, and what the guest that produced the proofs commits to.",
		EnvVars: prefixEnvVars("WIRE_VERSION"),
	}
)

func main() {
	oplog.SetupDefaults()

	flags := []cli.Flag{l1EthRPCFlag, inboxFlag, envelopeFlag, attestedRollupRPCFlag, attestedL2RPCFlag,
		attestedRollupConfigHashFlag, attestedDepSetHashFlag, attestedIntervalFlag, attestedMaxBlocksFlag,
		attestedL1LagFlag, attestedHeadFlag, attestedCursorFlag, wireVersionFlag}
	flags = append(flags, txmgr.CLIFlags(envVarPrefix)...)
	flags = append(flags, oplog.CLIFlags(envVarPrefix)...)

	app := cli.NewApp()
	app.Flags = flags
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "proofbatch-submitter"
	app.Usage = "silhouette proof-batch submitter"
	app.Description = "Packs proof-batch envelopes into blobs and submits them to the L1 inbox."
	app.Action = run

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Crit("application failed", "err", err)
	}
}

func run(cliCtx *cli.Context) error {
	logger := oplog.NewLogger(oplog.AppOut(cliCtx), oplog.ReadCLIConfig(cliCtx))
	oplog.SetGlobalLogHandler(logger.Handler())
	ctx := cliCtx.Context

	inbox, err := parseAddress(cliCtx.String(inboxFlag.Name))
	if err != nil {
		return fmt.Errorf("invalid --inbox: %w", err)
	}
	txCfg := txmgr.ReadCLIConfig(cliCtx)
	if err := txCfg.Check(); err != nil {
		return fmt.Errorf("invalid transaction manager config: %w", err)
	}
	txMgr, err := txmgr.NewSimpleTxManager("proofbatch-submitter", logger, &txmetrics.NoopTxMetrics{}, txCfg)
	if err != nil {
		return fmt.Errorf("build transaction manager: %w", err)
	}
	defer txMgr.Close()

	if path := cliCtx.Path(envelopeFlag.Name); path != "" {
		payload, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read envelope %q: %w", path, err)
		}
		return submit(ctx, logger, txMgr, inbox, payload)
	}
	return runAttested(ctx, cliCtx, logger, txMgr, inbox, txCfg.L1RPCURL)
}

// submit packs an envelope into blobs and sends it. The envelope is decoded first: a submitter
// that posts a payload no verifier can decode has spent a blob to say nothing.
//
// DecodeAny rather than a configured version, because this is a pre-flight check on bytes the caller
// framed — a file, or the attested builder's own output — and the check being performed is
// "is this readable and structurally sound", not "would MY verifier accept it". The version is logged
// so the operator sees what went out.
func submit(ctx context.Context, logger log.Logger, txMgr txmgr.TxManager, inbox common.Address, payload []byte) error {
	env, err := proofbatch.DecodeAny(payload)
	if err != nil {
		return fmt.Errorf("refusing to submit an undecodable envelope: %w", err)
	}
	if err := env.Batch.CheckStructure(); err != nil {
		return fmt.Errorf("refusing to submit an invalid batch: %w", err)
	}
	// Also the one acceptance rule a submitter can check for itself. A batch that every verifier will
	// refuse is a blob spent to say nothing, which is the same reason the decode above exists.
	if err := env.Batch.CheckNoSameTimestampImports(); err != nil {
		return fmt.Errorf("refusing to submit a batch every verifier will refuse: %w", err)
	}
	blobs, err := proofbatch.ToBlobs(payload)
	if err != nil {
		return fmt.Errorf("pack blobs: %w", err)
	}
	logger.Info("submitting proof batch", "inbox", inbox, "wire_version", env.Version,
		"bytes", len(payload), "blobs", len(blobs),
		"blocks", len(env.Batch.Blocks), "first", env.Batch.Blocks[0].Number,
		"last", env.Batch.Blocks[len(env.Batch.Blocks)-1].Number,
		"output_root", env.Batch.NewOutputRoot, "proof_bytes", len(env.Proof))

	receipt, err := txMgr.Send(ctx, txmgr.TxCandidate{To: &inbox, Blobs: blobs})
	if err != nil {
		return fmt.Errorf("submit proof batch: %w", err)
	}
	logger.Info("proof batch landed", "tx", receipt.TxHash, "l1_block", receipt.BlockNumber)
	return nil
}

// runAttested builds and submits batches on a cadence from the chain's own outputs, with an empty
// proof slot. This is not a stand-in for a prover: it is v1's ENTIRE producer side, and the send IS
// the attestation — the transaction is signed by the submitter key, acceptance rule 1 binds every
// accepted batch to that key, so posting a batch is the operator putting their name on it.
//
// It is also the shape a prover would step into: same wire, same slot, filled instead of empty.
func runAttested(ctx context.Context, cliCtx *cli.Context, logger log.Logger, txMgr txmgr.TxManager, inbox common.Address, l1RPCURL string) error {
	rollupURL := cliCtx.String(attestedRollupRPCFlag.Name)
	l2URL := cliCtx.String(attestedL2RPCFlag.Name)
	if rollupURL == "" || l2URL == "" {
		return errors.New("provide --envelope, or --attested.rollup-rpc and --attested.l2-rpc for attested mode")
	}
	rollupConfigHash, err := parseHash(cliCtx.String(attestedRollupConfigHashFlag.Name))
	if err != nil {
		return fmt.Errorf("invalid --attested.rollup-config-hash: %w", err)
	}
	depSetHash, err := parseHash(cliCtx.String(attestedDepSetHashFlag.Name))
	if err != nil {
		return fmt.Errorf("invalid --attested.dep-set-hash: %w", err)
	}

	head, err := ParseHeadSource(cliCtx.String(attestedHeadFlag.Name))
	if err != nil {
		return fmt.Errorf("invalid --attested.head: %w", err)
	}

	wireVersion, err := requiredWireVersion(cliCtx)
	if err != nil {
		return err
	}
	logger.Info("attested-cadence submitter will post at an explicit wire version",
		"wire_version", wireVersion,
		"declares_imports", proofbatch.VersionHasExecMsgs(wireVersion))

	l1RPC, err := client.NewRPC(ctx, logger, l1RPCURL)
	if err != nil {
		return fmt.Errorf("dial L1 %q: %w", l1RPCURL, err)
	}
	defer l1RPC.Close()
	l2RPC, err := client.NewRPC(ctx, logger, l2URL)
	if err != nil {
		return fmt.Errorf("dial L2 %q: %w", l2URL, err)
	}
	defer l2RPC.Close()
	rollupRPC, err := client.NewRPC(ctx, logger, rollupURL)
	if err != nil {
		return fmt.Errorf("dial rollup node %q: %w", rollupURL, err)
	}
	defer rollupRPC.Close()

	b, err := newBuilder(builderConfig{
		RollupConfigHash: rollupConfigHash,
		DepSetHash:       depSetHash,
		MaxBlocks:        cliCtx.Uint64(attestedMaxBlocksFlag.Name),
		L1Lag:            cliCtx.Uint64(attestedL1LagFlag.Name),
		CursorPath:       cliCtx.Path(attestedCursorFlag.Name),
		Head:             head,
	}, l1RPC, l2RPC, rollupRPC)
	if err != nil {
		return err
	}

	interval := cliCtx.Duration(attestedIntervalFlag.Name)
	logger.Info("attested mode", "interval", interval, "max_blocks", cliCtx.Uint64(attestedMaxBlocksFlag.Name), "head", head, "inbox", inbox)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := buildAndSubmit(ctx, logger, b, txMgr, inbox, wireVersion); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logger.Error("proof batch cycle failed, will retry", "err", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func buildAndSubmit(ctx context.Context, logger log.Logger, b *builder, txMgr txmgr.TxManager, inbox common.Address, wireVersion uint8) error {
	batch, err := b.next(ctx)
	if err != nil {
		return err
	}
	if batch == nil {
		logger.Info("no new blocks to batch")
		return nil
	}
	payload, err := proofbatch.EncodeAs(batch, nil, wireVersion)
	if err != nil {
		return fmt.Errorf("encode envelope at wire version %d: %w", wireVersion, err)
	}
	if err := submit(ctx, logger, txMgr, inbox, payload); err != nil {
		return err
	}
	return b.commit(batch)
}

// requiredWireVersion reads --wire-version and refuses to guess.
//
// It is checked here, before any RPC is dialled and long before a blob is paid for, and it refuses a
// version this codec cannot produce as well as an absent one. The reasoning for having no default is
// on wireVersionFlag.
func requiredWireVersion(cliCtx *cli.Context) (uint8, error) {
	if !cliCtx.IsSet(wireVersionFlag.Name) {
		return 0, fmt.Errorf("--%s is required: the version this submitter posts decides whether every "+
			"verifier checks the chain's declared imports or trusts them, and it must not be inherited "+
			"from whichever codec version the binary happened to be built with", wireVersionFlag.Name)
	}
	v := cliCtx.Uint(wireVersionFlag.Name)
	if v > 255 {
		return 0, fmt.Errorf("--%s %d is not a version byte", wireVersionFlag.Name, v)
	}
	version := uint8(v)
	if err := proofbatch.CheckVersion(version); err != nil {
		return 0, fmt.Errorf("--%s: %w", wireVersionFlag.Name, err)
	}
	return version, nil
}

func parseAddress(s string) (common.Address, error) {
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("%q is not an address", s)
	}
	return common.HexToAddress(s), nil
}

// parseHash requires a full 32 bytes: a short value would silently left-pad into a commitment no
// verifier will ever match, which is a confusing way to discover a typo.
func parseHash(s string) (common.Hash, error) {
	raw, err := hexutil.Decode(s)
	if err != nil {
		return common.Hash{}, err
	}
	if len(raw) != common.HashLength {
		return common.Hash{}, fmt.Errorf("%q is %d bytes, want %d", s, len(raw), common.HashLength)
	}
	return common.BytesToHash(raw), nil
}
