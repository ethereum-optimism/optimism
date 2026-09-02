// Dafny model for the LogsDB interface from logdb.go.
// Models the subset of the interface used by the Interop activity.

include "VerifiedDB.dfy"

module LogsDB {
  import opened Types
  import opened VerifiedDB

  class LogsDB {

    constructor()
      ensures {:axiom} LatestSealedBlock() == None

    // Removes all data from the database.
    // Pure I/O with no modeled state changes.
    // Corresponds to LogsDB.Clear in logdb.go.
    method Clear()
      modifies this
      ensures {:axiom} LatestSealedBlock() == None

    // Removes all blocks after newHead from the database.
    // Pure I/O with no modeled state changes.
    // Corresponds to LogsDB.Rewind in logdb.go (the LatestSealedBlock guard is
    // abstracted into the stub).
    method Rewind(newHead: BlockID)
      modifies this
      requires FindSealedBlock(newHead.number).Some?
      requires FindSealedBlock(newHead.number).value.id == newHead
      ensures {:axiom} LatestSealedBlock() == Some(newHead)
      ensures {:axiom} forall number :: 0 <= number <= newHead.number ==>
        FindSealedBlock(number) == old(FindSealedBlock(number))
      ensures {:axiom} forall number :: 0 <= number <= newHead.number && FindSealedBlock(number).Some? ==>
        BlockLogs(number) == old(BlockLogs(number))

    // Returns the first sealed block ID, or None if no blocks are sealed.
    // Corresponds to LogsDB.LatestSealedBlock in logdb.go.
    function FirstSealedBlock(): Option<BlockID>
      reads this
      ensures {:axiom} match FirstSealedBlock() {
        case None => forall number :: FindSealedBlock(number) == None
        case Some(block) =>
          FindSealedBlock(block.number).Some? &&
          FindSealedBlock(block.number).value.id == block &&
          forall number :: 0 <= number < block.number ==> FindSealedBlock(number) == None
      }

    // Returns the latest sealed block ID, or None if no blocks are sealed.
    // Corresponds to LogsDB.LatestSealedBlock in logdb.go.
    function LatestSealedBlock(): Option<BlockID>
      reads this
      ensures {:axiom} match LatestSealedBlock() {
        case None => forall number :: FindSealedBlock(number) == None
        case Some(block) =>
          FindSealedBlock(block.number).Some? &&
          FindSealedBlock(block.number).value.id == block &&
          forall number :: block.number < number ==> FindSealedBlock(number) == None
      }

    // Returns the sealed block at the given block number, or None if not found.
    // Corresponds to LogsDB.FindSealedBlock in logdb.go.
    function FindSealedBlock(number: nat): Option<BlockSeal>
      reads this
      ensures {:axiom} FindSealedBlock(number).Some? ==>
        FindSealedBlock(number).value.id.number == number
      ensures {:axiom} forall number' ::
        0 <= number' < number && FindSealedBlock(number').Some? && FindSealedBlock(number).Some? ==>
          FindSealedBlock(number').value.timestamp < FindSealedBlock(number).value.timestamp

    ghost function BlockLogs(blockNum: nat) : BlockLogs
      reads this
      requires FindSealedBlock(blockNum).Some?
      ensures {:axiom} |BlockLogs(FirstSealedBlock().value.number).execMsgs| == 0

    // Opens a sealed block and returns its block seal plus the map of
    // logIdx -> ExecutingMessage for all executing messages in that block.
    // In Go, OpenBlock fails with ErrSkipped for the first block in the DB
    // because there is no parent to verify against. In Dafny, callers handle
    // the first block separately using FirstSealedBlock() before calling OpenBlock.
    // Corresponds to LogsDB.OpenBlock in logdb.go.
    method OpenBlock(blockNum: nat) returns (ref: BlockSeal, execMsgs: map<nat, ExecutingMessage>)
      requires FindSealedBlock(blockNum).Some?
      requires FirstSealedBlock().Some? && FirstSealedBlock().value.number < blockNum
      ensures {:axiom} ref == FindSealedBlock(blockNum).value
      ensures {:axiom} execMsgs == BlockLogs(blockNum).execMsgs

    // Checks whether an initiating message matching the query exists in the DB.
    // Returns true if found. Simplified from Go's Contains, which returns (BlockSeal, error):
    // both ErrConflict (message absent) and ErrFuture (not yet indexed) map to false here.
    // Corresponds to LogsDB.Contains in logdb.go.
    predicate Contains(query: ContainsQuery)
      reads this
      ensures {:axiom} Contains(query) ==> FindSealedBlock(query.blockNum).Some?
      ensures {:axiom} Contains(query) ==> FindSealedBlock(query.blockNum).value.timestamp == query.timestamp
      ensures {:axiom} Contains(query) ==> query.logIdx < |BlockLogs(query.blockNum).fullLogs|
      ensures {:axiom} Contains(query) ==> BlockLogs(query.blockNum).fullLogs[query.logIdx].checksum == query.checksum

    // Monotonicity: if Contains held for a query and FindSealedBlock for the queried
    // block number is unchanged, Contains still holds. This captures that the LogsDB
    // only grows — adding blocks does not modify or remove existing entries.
    twostate lemma ContainsMonotone(query: ContainsQuery)
      requires old(Contains(query))
      requires FindSealedBlock(query.blockNum) == old(FindSealedBlock(query.blockNum))
      ensures {:axiom} Contains(query)
  }
}
