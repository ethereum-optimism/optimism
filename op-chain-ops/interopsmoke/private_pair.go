package interopsmoke

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// The private-pair profile resolves chain B's private receipts to public projection
// positions. The devstack registers the resolver in-process; the standalone CLI
// constructs the same resolver from explicit projection execution and rollup RPCs.
// Private recovery is exercised by the dedicated acceptance suite. This smoke's
// invalid-message checks currently cover replacement on the public counterparty.
// Native ETH interop is disabled; funding uses L1 deposits.

// privatePairWaitTimeout outlasts the in-process resolver's five-minute bound.
// Remote runs increase this budget to match their configured position-resolution timeout.
const privatePairWaitTimeout = 6 * time.Minute

// smokeSkip is a test that cannot apply to the topology it was pointed at. The run reports it and
// carries on: it is not a pass and not a failure, and the reason is always printed.
type smokeSkip struct {
	reason string
}

func (s *smokeSkip) Error() string {
	return "skipped: " + s.reason
}

// Native bridge messages are unsupported by the private projection, including legacy CGT pairs.
var errBridgeOnPrivatePair = &smokeSkip{
	reason: "native ETH interop is disabled for private pairs; fund the private ETH chain through L1 deposits",
}

// Private-chain recovery and cascading invalidation require a dedicated scenario.
var errChainedInvalidOnPrivatePair = errors.New(
	"chained-invalid-message is not implemented for private recovery; use the dedicated private-interop acceptance tests")

// defaultDirection is the invalid-message direction a suite runs when none was named.
func defaultDirection(privatePairB bool) string {
	if privatePairB {
		return directionBToA
	}
	return directionBoth
}

// usePrivatePairDirection selects public-counterparty invalidation. Its private
// EventLogger origin is not exported, so this tests rejection of a fabricated
// import. The valid-message mirror leg separately verifies exported positions.
func (env *smokeEnv) usePrivatePairDirection() error {
	switch env.direction {
	case directionBToA:
		return nil
	case "":
		env.direction = directionBToA
		fmt.Fprintf(env.stderr, "    Direction %q: public-counterparty invalidation\n", directionBToA)
		return nil
	default:
		return fmt.Errorf("direction %q requires a private recovery scenario not implemented by this smoke; use %q for public-counterparty invalidation",
			env.direction, directionBToA)
	}
}

// privateMirrorLeg exports a messenger event from the private chain, resolves its
// public position, and verifies that execution on the counterparty survives.
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

	// Resolve the identifier from the canonical private receipts using the same
	// rendering as the filter, so execution can precede batch publication.
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
	fmt.Fprintf(env.stderr, "  This smoke checks invalidation on chain A; private recovery has dedicated acceptance tests.\n")
	fmt.Fprintf(env.stderr, "  Native ETH interop is disabled; the private ETH profile is funded through L1 deposits.\n\n")
}
