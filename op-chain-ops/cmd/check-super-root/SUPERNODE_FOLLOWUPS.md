# Future SuperNode Follow-Ups for `check-super-root`

## Scope split

For this PR:

- `check-super-root` conforms to the SuperNode behavior that already exists today.
- `op-supernode` has no diff in this PR.

For future work:

- we can improve SuperNode where the missing functionality is clear and independently useful.

## What this PR does

`check-super-root` follows the existing SuperNode contract:

- the command treats `resp.Data == nil` as "verified super root unavailable"
- the command does not depend on optimistic or partial per-chain data
- the script tests validate the command against the real SuperNode behavior that already exists

## Future SuperNode work

These are the follow-up changes that may still be worth doing, but not in this PR.

### 1. Add an explicit way to query status / latest finalized timestamp

Today, `check-super-root` needs to infer a finalized timestamp from the existing `SuperRootAtTimestamp` response shape.

Future improvement:

- add a dedicated SuperNode status RPC, or
- formally define a status-only `SuperRootAtTimestamp` mode

Why this is useful:

- it makes the API contract explicit
- it removes the need for callers to infer status through a timestamp query

### 2. Normalize missing-data errors

Today, lower-level not-found cases may surface through different error types or wrappers.

Future improvement:

- normalize "not yet available / not found" cases into a stable API-facing shape

Why this is useful:

- callers can distinguish unavailable verified data from hard failures more reliably
- tests become less dependent on internal error plumbing

### 3. Clarify interop-at-genesis semantics

Some test environments use `InteropTime == 0` to mean "activate at genesis", which is not a clear contract boundary by itself.

Future improvement:

- document that convention explicitly, or
- replace it with a less ambiguous configuration path

Why this is useful:

- helper code does not need to infer intent from `0`
- SuperNode and test harnesses can agree on one activation-time model

## Out of scope for this PR

- changing SuperNode so `check-super-root` can succeed on non-verified timestamps
- changing SuperNode just to satisfy optimistic-data assertions in tests
- expanding the SuperNode contract without a separately justified API need
