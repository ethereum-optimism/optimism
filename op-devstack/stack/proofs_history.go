package stack

import "time"

// ProofsHistoryInit is one run of `op-reth proofs init` against a node's datadir: it anchors a
// proofs-history store at the datadir's tip, backfills the store Window blocks behind the anchor,
// and makes the node run with proofs history from its next Start on. The zero value is the
// operator's plain invocation.
type ProofsHistoryInit struct {
	// PlainBackfill reads every backfilled block's changes from reth's database instead of going
	// through the trie-state snapshot op-reth builds by default.
	PlainBackfill bool
	// Window is the number of blocks the backfill extends behind the anchor and the window the
	// node keeps once proofs history is on. Zero keeps the window the node was configured with.
	Window uint64
}

// ProofsHistoryUnwind is one run of `op-reth proofs unwind` against a node's datadir: it removes
// every block above Target from the node's proofs-history store, so Target becomes the store's
// latest block. A Target at or above the store's latest block leaves the store unchanged, and one
// below its earliest block fails.
type ProofsHistoryUnwind struct {
	// Target is the block the store keeps as its latest block.
	Target uint64
}

// ProofsHistorySubcommandReport is what a run of an `op-reth proofs` subcommand produced.
type ProofsHistorySubcommandReport struct {
	// Duration is how long the subcommand ran.
	Duration time.Duration
	// Output is the subcommand's combined stdout and stderr.
	Output string
}

// ProofsHistoryInitializer is implemented by execution-layer nodes whose proofs-history store can
// be seeded from their datadir after they were built, the way an operator seeds one for a node
// that has synced without proofs history.
type ProofsHistoryInitializer interface {
	// InitProofsHistory runs the init against the node's datadir, whether or not the node is
	// running, and enables proofs history on the node's next Start when it succeeds. The error of
	// a failed run carries the subcommand's output.
	InitProofsHistory(init ProofsHistoryInit) (ProofsHistorySubcommandReport, error)
}

// ProofsHistoryUnwinder is implemented by execution-layer nodes whose proofs-history store can be
// unwound to a block of their datadir, the way an operator recovers a store that got ahead of
// reth's datadir.
type ProofsHistoryUnwinder interface {
	// UnwindProofsHistory runs the unwind against the node's datadir, whether or not the node is
	// running. The error of a failed run carries the subcommand's output.
	UnwindProofsHistory(unwind ProofsHistoryUnwind) (ProofsHistorySubcommandReport, error)
}
