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
  }
}
