# Withdrawal proof-record deletion

Experimental lifecycle component; not yet proved. It reuses the production
Portal proxy and registry fixture described in `withdrawal-verification.md`.
It does not establish authentic record creation, actual finalizer enforcement,
arbitrary histories, game correctness or end-to-end theft prevention.

For an independently selected hash, submitter, uint64 proof timestamp, valid game
status, blacklist Boolean and finalized Boolean, deletion must succeed exactly
when the timestamp is nonzero and the game either lost or is blacklisted. On
success the selected record is cleared; on failure it is unchanged. The finalized
word is preserved in both cases. An arbitrary observer slot, read after setup,
must remain unchanged unless it is the selected record slot. This frame property
quantifies Portal storage locations within the seeded fixture, not every initial
storage configuration. The unused four bytes in the record word start at zero.

Three proofs instantiate the valid game statuses: in progress, challenger wins,
and defender wins. Their union exhausts that enum domain; each retains all other
symbolic inputs and the same assertion helper. All three must pass. This divides
work across CI workers without removing the arbitrary observer-slot check.

After successful deletion, the actual `checkWithdrawal` call must reject. Its
first rejection may be `AlreadyFinalized` when that flag is true; otherwise the
cleared timestamp makes the record unproven. This checks the eligibility method,
not either finalizer. Both invalidation alternatives have a concrete acceptance
witness. Re-seeding between witness cases is test setup, not a protocol history.

The success equivalence requires normally returning, ABI-valid game getters.
The existing game reporter supplies these; the actual registry blacklist getter
executes over seeded storage. An arbitrary reverting status getter can block
deletion even when blacklisted, because status is evaluated first. Expected
results derive from independent inputs, not from returned getter values.
The proxy configuration and Portal caller are fixed by the fixture. The symbolic
caller of a proof method is not automatically the caller of its nested Portal
call. Kontrol's hash and storage-separation model remains a trust dependency.

The guards follow production `OptimismPortal2.deleteProvenWithdrawal`; record
removal and unrelated-storage preservation are explicit transition obligations.
They support a future provenance invariant but do not establish one here.

This research branch temporarily selects only the new lifecycle proofs in its
manual CI job, with a 15-minute global prover limit. The verified eligibility
branch remains preserved. Restore full-suite routing and its original time
budget before treating this branch as a merge candidate. All proof execution is
in CI, using the existing deployment build, pinned Kontrol, stack checks, Cancun
with gas erased, and no custom K lemmas or `--assume-defined`.
