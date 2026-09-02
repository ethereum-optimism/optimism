package interopsmoke

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// The private-pair profile: what this smoke means when chain B is the PRIVATE half of a
// private-interop pair rather than an ordinary public chain.
//
// The pair is one chain ID standing for two chains: a private sequenced chain, which is what the
// RPC in Config.L2BURL talks to, and its public rendering, which is what the supernode judges and
// what every counterparty means by chain B (op-private-interop/docs/DESIGN.md). Three properties of
// that arrangement change what a test here can honestly claim:
//
//   - MESSAGES INITIATED ON THE PRIVATE CHAIN ARE NAMED PUBLICLY. Their identifier is a position on
//     the rendering, not the position in the receipt the private chain returned. The correction is
//     made by a resolver the devstack registers process-globally when it builds the pair, which is
//     why this profile is only honoured in-process (errPrivatePairOutOfProcess).
//   - THE PRIVATE CHAIN HAS NO JUDGE. Nothing outside it can replace its blocks; a validity failure
//     pins the operator's trust frontier instead of reorging. Every check that waits for a block to
//     be replaced has to wait on the public counterparty.
//   - THE PRIVATE CHAIN IS CUSTOM-GAS-TOKEN. Its native unit is not ETH, and the ETH paths across
//     the bridge are closed on it.
//
// Nothing here is a silent skip. A test that cannot mean what it says against a pair either says so
// where its result would go and passes the run on (smokeSkip), or refuses outright when it was
// asked for by name.

// privatePairWaitTimeout is the head/balance wait budget on both chains of a pair.
//
// A leg initiated on the private chain has no identifier until the rendering has derived the block
// that carries it, which takes a claim cadence; the resolver absorbs that wait and bounds itself at
// five minutes (op-devstack/presets/private_interop_resolver.go). The waits around it must outlast
// that bound rather than cut it short and report a timeout about the wrong thing.
const privatePairWaitTimeout = 6 * time.Minute

// smokeSkip is a test that cannot apply to the topology it was pointed at. The run reports it and
// carries on: it is not a pass and not a failure, and the reason is always printed.
type smokeSkip struct {
	reason string
}

func (s *smokeSkip) Error() string {
	return "skipped: " + s.reason
}

// errBridgeOnPrivatePair is the bridge test against a pair.
//
// The pair runs the STOCK SuperchainETHBridge on both halves. On the custom-gas-token private chain
// that bridge moves the native asset against an ETHLiquidity the stock genesis funds with
// uint128-max, so a transfer "works" without conserving anything across the pair. The smoke does
// not exercise it: a pass would be read as a statement about supply conservation that the design
// explicitly does not make.
var errBridgeOnPrivatePair = &smokeSkip{
	reason: "SuperchainETHBridge on a CGT private pair is unbacked stock liquidity; supply conservation is not guaranteed and not exercised here",
}

// errChainedInvalidOnPrivatePair refuses the transitive-invalidation test against a pair.
//
// The cascade it measures begins with chain B's block being replaced for containing an invalid
// relay. A private chain's blocks are never replaced, so the first step of the test cannot happen;
// running it would spend the whole reorg budget to report a timeout about the wrong thing.
var errChainedInvalidOnPrivatePair = errors.New(
	"chained-invalid-message cannot run against a private-interop pair: it starts by having chain B's block replaced, " +
		"and the private chain has no judge -- its blocks are never replaced, and a validity failure pins the " +
		"operator's trust frontier instead")

// errPrivatePairOutOfProcess refuses the private-pair profile on the command line.
//
// The identifiers of messages initiated on the private chain are corrected by a resolver registered
// process-globally by the devstack code that builds the pair
// (op-devstack/presets/private_interop.go, txintent.RegisterPositionResolver). A separate process
// has no resolver registered, so it would quote raw private receipt positions: the executing
// messages would name logs that do not exist publicly, and the legs that are supposed to prove the
// naming works would fail or pass vacuously without ever exercising it.
//
// The in-process door is op-up: `op-up --private-interop --smoke`.
var errPrivatePairOutOfProcess = errors.New(
	"--" + privatePairBFlagName + " is not supported out of process: a message initiated on the private chain is named by " +
		"its position on the rendering, and that correction lives in a resolver the devstack registers in the process " +
		"that BUILT the pair. Run the smoke in that process instead: `op-up --private-interop --smoke`")

// defaultDirection is the invalid-message direction a suite runs when none was named.
func defaultDirection(privatePairB bool) string {
	if privatePairB {
		return directionBToA
	}
	return directionBoth
}

// usePrivatePairDirection restricts the invalid-message test to the one direction that means
// anything against a pair.
//
// An invalid executing message landed ON the private chain would sit in a canonical block forever:
// there is no judge to replace it. The other direction -- initiated on the private chain, executed
// on the public counterparty -- is the real check.
//
// Read what it proves precisely. Its initiating message is an EventLogger log, which the export
// policy does not publish, so against a pair the executing message is already naming something with
// no public existence before its log index is bumped. What the leg establishes is that the
// COUNTERPARTY rejects a message the private chain's public presence does not carry -- the
// fabricated-import path -- rather than that a corrupted rendering position is caught. The
// resolver's own correctness is what the valid-message mirror leg is for.
func (env *smokeEnv) usePrivatePairDirection() error {
	switch env.direction {
	case directionBToA:
		return nil
	case "":
		env.direction = directionBToA
		fmt.Fprintf(env.stderr, "    Direction %q: the only one a private pair can be held to, see below\n", directionBToA)
		return nil
	default:
		return fmt.Errorf("direction %q cannot run against a private-interop pair: it would land an invalid executing "+
			"message on the private chain, whose blocks are never replaced -- it has no judge, and a validity failure "+
			"pins the operator's trust frontier instead. Use %q, which executes on the public counterparty",
			env.direction, directionBToA)
	}
}

// privateMirrorLeg initiates a message on the PRIVATE chain and executes it on the public
// counterparty. It is the reason the smoke runs in-process against a pair at all.
//
// The identifier the executing message quotes is a position on the RENDERING -- a different log
// index from the private receipt's, and a different origin for a log the rendering republishes
// through its generic replayer. That correction is the resolver's, and the resolver exists only in
// the process that built the pair. Without this leg a pair passes valid-message without a private
// message ever being named, which is exactly the vacuous pass an out-of-process run would give.
//
// It goes through the L2ToL2CrossDomainMessenger, not through an EventLogger like the leg above,
// because the export policy is not a per-message choice: a private chain publishes its messenger's
// SentMessage and its inbox's ExecutingMessage logs, and nothing else unless its genesis configures
// extra emitters (op-private-interop/render, EmitterSet.Renders). A log the rendering does not
// carry has no public position at all -- its identifier stays an honest private receipt position,
// which a judge correctly rejects -- so an EventLogger here would be a fabricated-import test
// wearing a valid-message name.
func (env *smokeEnv) privateMirrorLeg() error {
	initUser, execUser := env.userB, env.userA
	fmt.Fprintf(env.stderr, "    Mirror leg: initiated on %s (private), executed on %s\n", initUser.chain.name, execUser.chain.name)

	// The message is never relayed: what this leg is about is the SentMessage log the send emits,
	// which is what the rendering republishes and what the counterparty executes against.
	sent, err := initUser.sendMessage(env.ctx, execUser.chain.chainID, randomAddress(), []byte{})
	if err != nil {
		return fmt.Errorf("send messenger message on %s: %w", initUser.chain.name, err)
	}
	fmt.Fprintf(env.stderr, "    Message sent through the messenger on %s (block %d)\n",
		initUser.chain.name, bigs.Uint64Strict(sent.Receipt.BlockNumber))

	// Evaluating the result is where the identifier is resolved, and where this leg waits: a
	// message has no public position until the rendering has derived the block that carries it,
	// which takes a claim cadence.
	out, err := sent.Tx.Result.Eval(env.ctx)
	if err != nil {
		return fmt.Errorf("resolve the message's position on the rendering: %w", err)
	}
	entryIdx := firstLogFrom(sent.Receipt.Logs, predeploys.L2toL2CrossDomainMessengerAddr)
	if entryIdx < 0 || entryIdx >= len(out.Entries) {
		return fmt.Errorf("the send produced no messenger entry")
	}
	entry := out.Entries[entryIdx]
	fmt.Fprintf(env.stderr, "    Public position: origin %s, block %d, log index %d (private log index %d)\n",
		entry.Identifier.Origin, entry.Identifier.BlockNumber, entry.Identifier.LogIndex, sent.Receipt.Logs[entryIdx].Index)

	if _, err := waitForNextBlock(env.ctx, execUser.chain); err != nil {
		return err
	}
	receipt, err := execUser.execEntry(env.ctx, entry)
	if err != nil {
		return fmt.Errorf("execute the private chain's message on %s: %w", execUser.chain.name, err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("exec tx reverted on %s", execUser.chain.name)
	}
	blockNum := bigs.Uint64Strict(receipt.BlockNumber)
	fmt.Fprintf(env.stderr, "    Exec message sent on %s (block %d)\n", execUser.chain.name, blockNum)

	// If the position were wrong the message would be a fabricated import, and this is where the
	// counterparty's judge would say so by replacing the block.
	return assertBlockSurvives(env, execUser.chain, blockNum, receipt.BlockHash, receipt.TxHash)
}

// printPrivatePairProfile states, before anything runs, what this run does and does not check.
func printPrivatePairProfile(env *smokeEnv) {
	fmt.Fprintf(env.stderr, "Chain B is the PRIVATE half of a private-interop pair.\n")
	fmt.Fprintf(env.stderr, "  Its messages are named by their positions on its public rendering; this process resolves them.\n")
	fmt.Fprintf(env.stderr, "  Its blocks are never replaced: it has no judge, so every reorg check runs on chain A.\n")
	fmt.Fprintf(env.stderr, "  It is custom-gas-token: its native unit is not ETH, and the ETH bridge path is closed.\n\n")
}
