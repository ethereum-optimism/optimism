// Dafny model for verified_db.go

include "Types.dfy"
include "Utils.dfy"

module VerifiedDB {
  import opened Types
  import opened Utils

  // ----- VerifiedDB class -------------------------------------------------

  class VerifiedDB {

    // Abstract state: map from timestamp to verified result.
    // Replaces the bbolt on-disk key-value store.
    var db: map<nat, VerifiedResult>

    // Cached most-recently-committed timestamp. Mirrors the Go field of the
    // same name; kept consistent with db by the class invariant below.
    var lastTimestamp: Option<nat>

    // Write-ahead log entry for crash recovery.
    // Replaces the bbolt pending_transition bucket.
    var pendingTransition: Option<PendingTransition>

    // Class invariant:
    //   (1) Committed timestamps form a contiguous range (Sequential).
    //   (2) Each entry's timestamp field matches its map key.
    //   (3) lastTimestamp is consistent with the contents of db.
    ghost predicate Valid()
      reads this
    {
      Sequential(db) &&
      (forall ts :: ts in db ==> db[ts].timestamp == ts) &&
      lastTimestamp == (if |db| == 0 then None else Some(MaxKey(db))) &&
      (forall t1, t2, cid ::
        t1 in db && t2 in db && t1 <= t2 &&
        cid in db[t1].l2Heads && cid in db[t2].l2Heads ==>
        db[t1].l2Heads[cid].number <= db[t2].l2Heads[cid].number)
    }

    // Initializes an empty VerifiedDB.
    constructor()
      ensures Valid()
      ensures db == map[]
      ensures pendingTransition == None
    {
      db := map[];
      lastTimestamp := None;
      pendingTransition := None;
    }

    // Stores a verified result at the next sequential timestamp.
    // Precondition replaces the ErrNonSequential / ErrAlreadyCommitted error returns:
    // the caller is responsible for providing the correct next timestamp.
    method Commit(result: VerifiedResult)
      modifies this
      requires Valid()
      requires |db| == 0 || result.timestamp == MaxKey(db) + 1
      requires LastTimestamp().Some? ==>
        result.l2Heads.Keys == db[LastTimestamp().value].l2Heads.Keys
      requires LastTimestamp().Some? ==>
        forall cid :: cid in result.l2Heads ==>
          db[LastTimestamp().value].l2Heads[cid].number <= result.l2Heads[cid].number
      ensures Valid()
      ensures db == old(db)[result.timestamp := result]
      ensures lastTimestamp == Some(result.timestamp)
      ensures pendingTransition == old(pendingTransition)
    {
      MaxKeyInsertNewMax(db, result.timestamp, result);
      db := db[result.timestamp := result];
      lastTimestamp := Some(result.timestamp);
    }

    // Returns the verified result at the given timestamp.
    // Precondition replaces the ErrNotFound error return.
    function Get(ts: nat): VerifiedResult
      reads this
      requires ts in db
    {
      db[ts]
    }

    // Returns whether a timestamp has a committed verified result.
    // Can equivalently be expressed inline as `ts in db`.
    function Has(ts: nat): bool
      reads this
    {
      ts in db
    }

    // Returns the most recently committed timestamp, or None if empty.
    function LastTimestamp(): Option<nat>
      reads this
    {
      lastTimestamp
    }

    // Removes all verified results at or after the given timestamp.
    // Returns true if any entries were deleted.
    method Rewind(timestamp: nat) returns (deleted: bool)
      modifies this
      requires Valid()
      ensures Valid()
      ensures db == map k | k in old(db) && k < timestamp :: old(db)[k]
      ensures deleted <==> exists k :: k in old(db) && k >= timestamp
      ensures pendingTransition == old(pendingTransition)
    {
      var newDb := map k | k in db && k < timestamp :: db[k];
      FilterBelowSmallerIffKeyAbove(db, timestamp);
      deleted := |newDb| < |db|;
      if |newDb| == 0 {
        lastTimestamp := None;
      } else {
        MaxKeyFilterBelow(db, timestamp);
        var oldMax := lastTimestamp.value;
        lastTimestamp := Some(if oldMax < timestamp then oldMax else timestamp - 1);
      }
      db := newDb;
    }

    // Persists a pending transition as the write-ahead log entry.
    method SetPendingTransition(pending: PendingTransition)
      modifies this
      requires Valid()
      ensures Valid()
      ensures pendingTransition == Some(pending)
      ensures db == old(db)
      ensures lastTimestamp == old(lastTimestamp)
    {
      pendingTransition := Some(pending);
    }

    // Returns the current pending transition, or None if none exists.
    function GetPendingTransition(): Option<PendingTransition>
      reads this
    {
      pendingTransition
    }

    // Clears the pending transition after it has been fully applied.
    method ClearPendingTransition()
      modifies this
      requires Valid()
      ensures Valid()
      ensures pendingTransition == None
      ensures db == old(db)
      ensures lastTimestamp == old(lastTimestamp)
    {
      pendingTransition := None;
    }

  }
}
