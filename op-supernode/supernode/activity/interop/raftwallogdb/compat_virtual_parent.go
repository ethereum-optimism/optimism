// COMPATIBILITY SHIM — safe to delete once no live databases predate
// ethereum-optimism/optimism#20726.
//
// Pre-#20726, the supernode and interop filter sealed a synthetic "virtual
// parent" entry as the first block in a fresh logsDB: number = activation-1,
// parent hash = zero, no logs, no exec messages. #20726 removed that
// indirection and seals the activation block directly. Databases written
// before the change still carry the vestigial entry.
//
// hidePreVirtualParentLayout detects the entry on Open and bumps d.firstBlock
// past it. The raft-wal record stays on disk (one wasted entry) but is
// unreachable through the public API. Internal write paths maintain
// firstBlock from this point forward, so the detection only runs once per
// process lifetime.
//
// Removal: delete this file and the single call from refreshCache. Behaviour
// for post-#20726 databases is unchanged because the detection is a no-op
// when the first entry has a non-zero parent hash.

package raftwallogdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// hidePreVirtualParentLayout hides a pre-#20726 virtual-parent entry, if
// present, by advancing d.firstBlock past it. Must be called with the cache
// already populated (hasBlocks/firstBlock/latest set).
func (d *DB) hidePreVirtualParentLayout() error {
	// No data, genesis-anchored DB, or only the (possibly virtual) entry
	// exists: nothing to hide. Requiring a second entry means we never bump
	// firstBlock past latest, and avoids touching DBs that never wrote an
	// activation block.
	if !d.hasBlocks || d.firstBlock == 0 || d.firstBlock >= d.latest.Number {
		return nil
	}

	rec, err := d.readBlockAt(indexFor(d.firstBlock))
	if err != nil {
		return fmt.Errorf("read first entry for virtual-parent detection: %w", err)
	}

	// The virtual parent was sealed via SealBlock(common.Hash{}, ...) with no
	// AddLog calls preceding it. Any of these three deviations rules it out;
	// require all three for belt-and-braces (the supernode/filter write paths
	// never produce a non-genesis entry with a zero parent hash).
	if rec.ParentHash != (common.Hash{}) || rec.LogCount != 0 || rec.ExecMsgCount != 0 {
		return nil
	}

	d.firstBlock++
	return nil
}
