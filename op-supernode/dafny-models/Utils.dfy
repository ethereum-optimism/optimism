// Utility lemmas and predicates over nat and generic maps.

module Utils {

  // Maximum element of a non-empty set of naturals.
  ghost function Max(s: set<nat>): nat
    requires |s| > 0
    ensures Max(s) in s
    ensures forall k :: k in s ==> k <= Max(s)
  {
    var x :| x in s;
    if s == {x} then x
    else
      var rest := s - {x};
      var maxRest := Max(rest);
      assert forall k :: k in s ==> (k == x || k in rest);
      if x >= maxRest then x else maxRest
  }

  // Maximum timestamp key in a non-empty map.
  // Generic in the value type so it can be shared across any nat-keyed map.
  ghost function MaxKey<V>(m: map<nat, V>): nat
    requires |m| > 0
    ensures MaxKey(m) in m
    ensures forall k :: k in m ==> k <= MaxKey(m)
  {
    Max(m.Keys)
  }

  // The committed timestamps form a contiguous integer range:
  // every key strictly below the maximum has an immediate successor in m.
  // Generic in the value type for the same reason as MaxKey.
  ghost predicate Sequential<V>(m: map<nat, V>)
  {
    |m| == 0 ||
    forall k :: k in m && k < MaxKey(m) ==> k + 1 in m
  }

  // Inserting a key greater than all existing keys makes it the new maximum.
  lemma MaxKeyInsertNewMax<V>(m: map<nat, V>, k: nat, v: V)
    requires |m| == 0 || k > MaxKey(m)
    ensures MaxKey(m[k := v]) == k
  {
    if |m| == 0 {
      assert MaxKey(m[k := v]) == k;
    } else {
      assert MaxKey(m[k := v]) == k;
    }
  }

  // The maximum key of a below-threshold filter is min(MaxKey(m), threshold - 1).
  // Requires the filtered map to be non-empty (i.e. some key lies below the threshold).
  lemma MaxKeyFilterBelow<V>(m: map<nat, V>, threshold: nat)
    requires |m| > 0
    requires Sequential(m)
    requires |map k | k in m && k < threshold :: m[k]| > 0
    ensures MaxKey(map k | k in m && k < threshold :: m[k]) ==
            if MaxKey(m) < threshold then MaxKey(m) else threshold - 1
  {
    var newMap := map k | k in m && k < threshold :: m[k];

    if MaxKey(m) < threshold {
      assert newMap == m;
    } else {
      var keyBelowThreshold :| keyBelowThreshold in m && keyBelowThreshold < threshold;
      SequentialContainsRange(m, keyBelowThreshold);
      assert threshold - 1 in newMap;
      assert MaxKey(newMap) == threshold - 1;
    }
  }

  // In a sequential map, every integer in [k0, MaxKey(m)] is a key of m.
  lemma SequentialContainsRange<V>(m : map<nat, V>, k0 : nat)
    requires |m| > 0
    requires Sequential(m)
    requires k0 in m
    requires k0 <= MaxKey(m)
    ensures forall k :: k0 <= k <= MaxKey(m) ==> k in m
  {
    var i := k0;

    while i < MaxKey(m)
      invariant forall k :: k0 <= k < i ==> k in m
    {
      if k0 < i {
        assert (i - 1) in m;
      }

      i := i + 1;
    }
  }

  // Filtering a map to keys below threshold strictly reduces its size iff some key is at or above the threshold.
  lemma FilterBelowSmallerIffKeyAbove<V>(m: map<nat, V>, threshold: nat)
    ensures (|map k | k in m && k < threshold :: m[k]| < |m|) <==>
            (exists k :: k in m && k >= threshold)
  {
    var newMap := map k | k in m && k < threshold :: m[k];

    if |newMap| < |m| {
      assert |m.Keys - newMap.Keys| > 0;
    } else {
      assert |m.Keys - newMap.Keys| == 0;
    }
  }

}
