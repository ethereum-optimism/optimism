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

// ProofsHistoryInitReport is what a proofs-history init run produced.
type ProofsHistoryInitReport struct {
	// Duration is how long the subcommand ran, backfill included.
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
	InitProofsHistory(init ProofsHistoryInit) (ProofsHistoryInitReport, error)
}
