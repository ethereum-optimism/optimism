# Deletion Review

Method for reviewing a diff that **deletes** externally observable names or state
writes. Pairs with the `deletion-reviewer` agent (`.claude/agents/deletion-reviewer.md`);
under a harness without that agent, work through this guide directly.

## When this review applies

The diff removes any of:

- a public symbol: type, function, event, enum variant, interface method, parameter
- a wire name: RPC method, JSON response field, WS subscription
- a metric name or a metric **label value**
- a config key, CLI flag, or env var
- a test or subtest name

Deletions fail differently from additions: the compiler proves that no *code* still
needs the removed thing, and generic code review then stops there. The two checks below
cover what neither proves.

## Check 1: the reference sweep goes beyond code

Build a deletion inventory in both code form and **string form** (JSON keys, metric
label values, method names, subtest names), then sweep the whole tree. Reference sites
that no compiler sees, in rough order of how often they are missed:

1. **Public docs** (`docs/public-docs/`): field lists *and* example payloads — a JSON
   example embeds a wire field a second time, inside a code fence.
2. **Grafana dashboards** and monitoring config (`**/grafana/**/*.json`): a removed
   metric or label value leaves a panel charting a permanently empty series. When
   dashboards exist in duplicated variants, keep them byte-identical.
3. **CI config, justfiles, workflows**: test names, binary names, package lists —
   these are enumerated strings that silently skip or break when a name changes.
4. READMEs, compose files, scripts.

Classify every hit into exactly one of:

- **must-update** — fix in the same PR;
- **deliberate survivor** — e.g. a shared Go type another service still populates, or
  an enum value kept for wire compatibility; state the reason in the PR;
- **same-name, different concept** — verify the boundary and leave it alone. Names are
  overloaded: the same word can label a chain head in one subsystem and a per-message
  validation threshold in another. The sweep must not overreach into live functionality
  that shares the name; when the boundary is subtle, record it in the PR description.

## Check 2: deleted writes — prove *when* the survivors fire, not *whether* they exist

The subtlest deletion bug: removed code was one of several writers to state that
survives (a status field, tracker, metric, head label), and review "confirms" safety by
observing that other writers exist. Existence is not coverage.

For each deleted write:

1. Enumerate the surviving writers of the same state.
2. For each, establish the exact firing conditions — triggering event, guards, mode.
3. Prove the union covers every window the deleted writer covered. The windows that get
   missed: **startup/initialization**, **sync modes** (EL/snap sync, before derivation
   runs and forkchoice updates are gated), **resets**, **reorgs** (writers that must
   move a value *backward*), and **error/halt paths**.

An uncovered window means the value goes silently stale exactly when operators or
downstream services (health monitors, dashboards) are watching it. If the old coupling
was incidental, make the new coupling explicit rather than restoring the deleted path.

Tests for the fix must pin the semantics, not just the happy case: if the write must
also move a value backward, assert that, or a later "hardening" to advances-only
reintroduces the staleness.

## Consequential cleanups

- **Now-unproducible code**: error variants or branches whose only producer was
  deleted. Delete them with the producer.
- **Dead parameters**: values threaded through interfaces that nothing reads after the
  removal — shrink the signatures in the same PR.
- **Vacuous tests**: an assertion on a removed field can start comparing zero-to-zero
  and pass forever. Prefer making the removed concept an explicit error over returning
  a zero value.
- **Wire compatibility**: removed RPC/JSON fields parse as zero values in lenient
  clients — trace what each in-repo consumer does with that zero, and disclose the
  removal (breaking-change marker + migration note) for out-of-repo readers.

## False-positive traps

- **Shared types**: a field removed from one implementation's output may legitimately
  remain in a shared struct that another service still populates. Removing it there is
  a separate, wider change — do not flag the survivor as a leftover.
- **Overloaded names**: see "same-name, different concept" above; a grep hit is a
  question, not a finding.
- **Duplicated artifacts**: paired dashboards or mirrored configs must be updated
  together; flagging only one file is an incomplete finding.
