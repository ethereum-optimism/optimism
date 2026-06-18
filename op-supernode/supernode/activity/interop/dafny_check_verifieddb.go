package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	bolt "go.etcd.io/bbolt"
)

// allVerified snapshots the verified bucket as the model state
// `db: map<nat, VerifiedResult>` of VerifiedDB.dfy, via a read-only bbolt
// View. An entry that does not map to the model (key not an 8-byte
// big-endian timestamp, value not a JSON VerifiedResult) is an error.
func (v *VerifiedDB) allVerified() (map[uint64]VerifiedResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	out := make(map[uint64]VerifiedResult)
	err := v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, val []byte) error {
			if len(k) != u64Len {
				return fmt.Errorf("key %x is not an %d-byte big-endian timestamp", k, u64Len)
			}
			var result VerifiedResult
			if err := json.Unmarshal(val, &result); err != nil {
				return fmt.Errorf("value at key %d is not a VerifiedResult: %w", binary.BigEndian.Uint64(k), err)
			}
			out[binary.BigEndian.Uint64(k)] = result
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// checkSequential mirrors Sequential(m) in op-supernode/dafny-models/Utils.dfy
// over the key set of a nat-keyed map: nil iff keys is empty or every key
// strictly below the maximum has an immediate successor. keys must be the
// distinct keys of a map, in any order.
func checkSequential(keys []uint64) error {
	if len(keys) == 0 {
		return nil
	}
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1]+1 {
			return fmt.Errorf("gap at %d (next committed key is %d)", sorted[i-1]+1, sorted[i])
		}
	}
	return nil
}

// CheckVerifiedDBValid mirrors the class invariant VerifiedDB.Valid() in
// op-supernode/dafny-models/VerifiedDB.dfy, with the model `db` enumerated
// from the bbolt verified bucket (allVerified) rather than the cached
// timestamp fields, and `lastTimestamp: Option<nat>` mapped to the Go
// (lastTimestamp, initialized) pair. Conjuncts:
//
//	(0) every stored entry maps to the model db: map<nat, VerifiedResult>
//	    and the store is reachable (mapping requirement; the remaining
//	    conjuncts assume it)
//	(1) Sequential(db): committed timestamps form a contiguous range
//	(2) forall ts in db: db[ts].timestamp == ts
//	(3) lastTimestamp == (if |db| == 0 then None else Some(MaxKey(db)))
//	(4) forall t1 <= t2 in db, cid in both l2Heads:
//	    db[t1].l2Heads[cid].number <= db[t2].l2Heads[cid].number
func CheckVerifiedDBValid(v *VerifiedDB) error {
	const pred = "VerifiedDB.dfy Valid()"
	if v == nil || v.db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}
	db, err := v.allVerified()
	if err != nil {
		return violation(pred, "0", "verified bucket does not map to db: map<nat, VerifiedResult>: %v", err)
	}
	last, initialized := v.LastTimestamp()

	var errs []error
	keys := slices.Sorted(maps.Keys(db))

	if err := checkSequential(keys); err != nil {
		errs = append(errs, violation(pred, "1", "committed timestamps not sequential: %v", err))
	}

	for _, ts := range keys {
		if db[ts].Timestamp != ts {
			errs = append(errs, violation(pred, "2",
				"entry at key %d has timestamp field %d", ts, db[ts].Timestamp))
		}
	}

	switch {
	case len(db) == 0 && initialized:
		errs = append(errs, violation(pred, "3",
			"db is empty but lastTimestamp is Some(%d)", last))
	case len(db) > 0 && !initialized:
		errs = append(errs, violation(pred, "3",
			"db has %d entries but lastTimestamp is None", len(db)))
	case len(db) > 0 && last != keys[len(keys)-1]:
		errs = append(errs, violation(pred, "3",
			"lastTimestamp %d != MaxKey(db) %d", last, keys[len(keys)-1]))
	}

	// Consecutive-occurrence comparisons over ascending timestamps cover all
	// t1 <= t2 pairs of conjunct (4) by transitivity of <=.
	type seen struct {
		number uint64
		ts     uint64
	}
	lastSeen := make(map[eth.ChainID]seen)
	for _, ts := range keys {
		heads := db[ts].L2Heads
		for _, cid := range slices.SortedFunc(maps.Keys(heads), eth.ChainID.Cmp) {
			if prev, ok := lastSeen[cid]; ok && heads[cid].Number < prev.number {
				errs = append(errs, violation(pred, "4",
					"chain %s l2Heads number decreases from %d at ts %d to %d at ts %d",
					cid, prev.number, prev.ts, heads[cid].Number, ts))
			}
			lastSeen[cid] = seen{number: heads[cid].Number, ts: ts}
		}
	}

	return errors.Join(errs...)
}

// AssertVerifiedDBValid fails t when CheckVerifiedDBValid reports violations.
func AssertVerifiedDBValid(t dafnyT, v *VerifiedDB) {
	t.Helper()
	failOnViolation(t, CheckVerifiedDBValid(v))
}
