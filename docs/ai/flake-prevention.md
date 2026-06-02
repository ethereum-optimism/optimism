# Flake Prevention — op-acceptance-tests & op-devstack

Companion to [writing-acceptance-tests.md](writing-acceptance-tests.md). That doc tells you how to write a test cleanly; this doc names the *recurring* anti-patterns that have caused flakes in CI, the static checks that catch them, and the review questions that catch the ones a linter can't.

If you are reviewing a PR that touches `op-acceptance-tests/` or `op-devstack/`, run through the [Reviewer checklist](#reviewer-checklist) at the bottom.

## Why flakes keep landing

Education ("don't use short timeouts, don't sleep") has not closed the gap because the patterns that actually flake are *subtle in a diff*: they look like reasonable Go.

| Surface form | What flakes about it |
|---|---|
| `require.NoError(x.RPC(...))` inside `require.Eventually(...)` | A transient RPC error becomes a fatal `FailNow` *inside the callback*, killing the test on the first hiccup instead of letting the retry loop absorb it. |
| `WaitForStall(unsafe)` then `WaitForStall(safe)` | The two waits independently succeed but the *invariant `safe == unsafe`* downstream code assumes is not enforced. |
| `head := node.Head(); WaitForOther(head)` | TOCTOU — `head` is stale by the time the second call uses it. |
| `Sleep(2s); SequenceBlock()` | Looks like "give the system a moment"; actually a missing wait on a precondition. |
| `NewMultiNodeWithoutCheck(); WaitForBudget(60)` | Bypasses the preset's auto-detected sync budget. The preset would have used the correct 240s budget for ELSync; the test hard-codes 120s. |
| `defer cancel(); ... defer wg.Wait()` | LIFO: `Wait` runs first and blocks forever because `cancel` hasn't fired yet. |
| `go func(){ require.NoError(...) }()` | `FailNow` in a non-test goroutine only exits *the goroutine*; the test hangs until package timeout. |

If you have written one of these, you have probably also written it in another test. Lint catches the ones with a syntactic signature; the rest are reviewer responsibility.

## Anti-pattern catalogue

Each entry is grounded in an actual fix from the past month. The PR link is the canonical example of what the fix looks like.

### F1 — Fatal assertion inside a polling/retry callback

```go
// BAD — first transient error fails the test; sequencer's currentJob is left
// non-nil, so every subsequent retry hits ErrConflictingJob and the test hangs.
require.Eventually(t, func() bool {
    head := ts.New(parent)
    require.NoError(t, ts.Next(head))   // <-- this kills the test on first stall
    return condition(head)
}, 30*time.Second, 200*time.Millisecond)

// GOOD — let the loop absorb transient errors.
require.Eventually(t, func() bool {
    head := ts.New(parent)
    if err := ts.Next(head); err != nil {
        log.Warn("transient ts.Next error", "err", err)
        return false
    }
    return condition(head)
}, 30*time.Second, 200*time.Millisecond)
```

Same shape applies to `assert.NoError`, `t.Fatal`, `Require()` chains. Anything that calls `FailNow` inside a closure that the runtime expects to be re-called.

**Examples:** [#20651](https://github.com/ethereum-optimism/optimism/pull/20651), [#20255](https://github.com/ethereum-optimism/optimism/pull/20255), [#20209](https://github.com/ethereum-optimism/optimism/pull/20209), [#19976](https://github.com/ethereum-optimism/optimism/pull/19976). **Lint:** rule `flake-require-in-eventually` (catches this exactly).

### F2 — `require.*` inside a goroutine

```go
// BAD — FailNow only exits this goroutine; the main test hangs until package timeout.
go func() {
    for evt := range stream {
        require.NoError(t, evt.Err)
    }
}()

// GOOD — surface the error to the test via t.Errorf (non-fatal) or a channel.
errCh := make(chan error, 1)
go func() {
    for evt := range stream {
        if evt.Err != nil {
            errCh <- evt.Err
            return
        }
    }
}()
```

**Examples:** [#19976](https://github.com/ethereum-optimism/optimism/pull/19976), [#20255](https://github.com/ethereum-optimism/optimism/pull/20255). **Lint:** `flake-require-in-goroutine`.

### F3 — TOCTOU between two RPCs that depend on each other

```go
// BAD — head may advance between the two calls.
head, err := node.UnsafeHead(ctx)
require.NoError(t, err)
err = node.StartSequencer(ctx, head.Hash)
require.NoError(t, err)   // "block hash does not match"

// GOOD — retry the read+use pair on the documented "stale head" error.
retry.Do(ctx, func() error {
    head, err := node.UnsafeHead(ctx)
    if err != nil { return err }
    return node.StartSequencer(ctx, head.Hash)
})
```

**Examples:** [#20204](https://github.com/ethereum-optimism/optimism/pull/20204). **Lint:** partial (`flake-toctou-start-sequencer`).

### F4 — Wait on the wrong post-condition (or no post-condition at all)

```go
// BAD — ReorgTriggered guarantees the reorg-target block was rewritten, NOT that
// the sequencer has extended past the verifier's frozen height. Assertion races
// post-reorg block production.
chain.ReorgTriggered()
require.Greater(t, seq.Number(), ver.Number())

// GOOD — explicit wait on the actual condition being asserted.
frozen := ver.Head().Number
chain.ReorgTriggered()
seq.WaitForBlockNumber(frozen + 1)
require.Greater(t, seq.Number(), ver.Number())
```

```go
// BAD — proveWithdrawal reverts until block.timestamp > game.createdAt. The
// retry loop burns its 30s budget on guaranteed-revert estimateGas calls.
retryProve(ctx, 30*time.Second)

// GOOD — wait deterministically for the static precondition first.
require.Eventuallyf(t, func() bool {
    head, _ := l1.HeadBlock(ctx)
    return head.Time > gameCreatedAt
}, 60*time.Second, 1*time.Second, "L1 head past game createdAt")
retryProve(ctx, 30*time.Second)   // now scoped to transient submit/confirm errors only
```

```go
// BAD — CL/supervisor state reached the target, but the EL label/content is
// updated through an async forkchoice/reorg path. A synchronous EL read can
// still observe the previous safe label or old block contents.
cl.Reached(types.CrossSafe, target, 30)
safe := el.BlockRefByLabel(eth.Safe)
require.GreaterOrEqual(t, safe.Number, target)

// GOOD — wait on the component and observable state the assertion actually
// reads. If the assertion is about EL labels or block contents, include an
// EL-side wait before the synchronous assertion.
dsl.CheckAll(t,
    cl.ReachedFn(types.CrossSafe, target, 30),
    el.ReachedFn(eth.Safe, target, 30),
)
safe = el.BlockRefByLabel(eth.Safe)
require.GreaterOrEqual(t, safe.Number, target)
```

The same shape appears when a supervisor or CL validation wait is followed by
an EL block-content assertion. `AwaitValidatedTimestamp` and `Reached(CrossSafe)`
prove the control-plane view advanced; they do not automatically prove the EL
has exposed the replacement canonical block or historical proof state.

**Examples:** [#20199](https://github.com/ethereum-optimism/optimism/pull/20199), [#20677](https://github.com/ethereum-optimism/optimism/pull/20677), [#20482](https://github.com/ethereum-optimism/optimism/pull/20482), [#20852](https://github.com/ethereum-optimism/optimism/pull/20852), [#20782](https://github.com/ethereum-optimism/optimism/pull/20782). **Lint:** no (semantic — needs a reviewer).

### F5 — "Done" signal that doesn't entail "work happened"

```go
// BAD — AwaitBackfillCompleted resolves when runLogBackfill returns OR is
// skipped (e.g. nil-range when L1 head was briefly behind). Test asserts that
// blocks were sealed; gets `latest.Number == first.Number == 0`.
chain.RestartInterop(wipeLogsDBs=true)
chain.AwaitBackfillCompleted()
require.Greater(t, chain.LatestBlock(), chain.FirstBlock())

// GOOD — wait on the actual user-visible postcondition (sealed blocks),
// or strengthen AwaitBackfillCompleted to require non-empty backfill.
chain.RestartInterop(wipeLogsDBs=true)
chain.WaitForBackfillToSeal(types.MinBlocks(2))
```

**Examples:** [#20690](https://github.com/ethereum-optimism/optimism/issues/20690). **Lint:** no.

The same trap applies to `Stop`, `Reset`, and `Restart` helpers: the call
returning does not necessarily mean every background producer is quiesced or
that every hidden cache has been cleared. If the next assertion depends on
empty session state, drained queues, or no more payloads arriving, the helper
must either guarantee that stronger postcondition or the test must wait/assert
on it directly.

**Related:** [#20783](https://github.com/ethereum-optimism/optimism/pull/20783).

### F6 — Hand-rolled sync check that bypasses the preset's auto-budget

```go
// BAD — preset's NewSingleChainMultiNode auto-detects ELSync and applies a 4x
// budget. WithoutCheck plus a manual 120-attempt loop undoes that fix.
sys := presets.NewSingleChainMultiNodeWithoutCheck(t, opts)
require.NoError(t, dsl.MatchedFn(sys, types.CrossSafe, 60, 2*time.Second)(ctx))

// GOOD — use the auto-budgeted entry point.
sys := presets.NewSingleChainMultiNode(t, opts)
```

**Examples:** [#20454](https://github.com/ethereum-optimism/optimism/pull/20454), [#20343](https://github.com/ethereum-optimism/optimism/pull/20343). **Lint:** `flake-without-check-with-manual-budget`.

### F7 — Snapshot-once sync check that holds a stale target across reorgs

```go
// BAD — target is captured once outside the loop. If the reference node reorgs
// out that block, the predicate is stuck waiting for a hash that no longer exists.
target := ref.Head()
require.Eventually(t, func() bool {
    return base.Head().Number >= target.Number
}, 60*time.Second, 1*time.Second)

// GOOD — re-sample both sides every attempt, allow a small bounded gap, and
// verify hash agreement at the lower height.
require.Eventually(t, func() bool {
    a, b := ref.Head(), base.Head()
    if absDiff(a.Number, b.Number) > 5 { return false }
    return base.HashAt(min(a.Number, b.Number)) == ref.HashAt(min(a.Number, b.Number))
}, 60*time.Second, 1*time.Second)
```

**Examples:** [#20405](https://github.com/ethereum-optimism/optimism/pull/20405). **Lint:** partial.

### F8 — `time.Sleep` between async-driven operations

```go
// BAD — sequencing a block is async; sleeping 2s is "usually enough" until CI is loaded.
ts.SequenceBlock(parent)
time.Sleep(2 * time.Second)
ts.SequenceBlock(child)

// GOOD — wait on the block being visible at the consumer.
ts.SequenceBlock(parent)
chain.WaitForBlockNumber(parent.Number + 1)
ts.SequenceBlock(child)
```

The existing `writing-acceptance-tests.md` already bans `time.Sleep`. Lint enforces it now.

**Examples:** [#20198](https://github.com/ethereum-optimism/optimism/issues/20198). **Lint:** `flake-sleep-in-test`.

### F9 — Invariants between two parallel waits not enforced

```go
// BAD — sequencers and batchers stopped in parallel. WaitForStall returns when
// each independently quiesces, but `safe < unsafe` can persist. Downstream code
// that assumes `safe == unsafe` (timestamp arithmetic) computes nonsense.
stopAllSequencers()
stopAllBatchers()
waitForStall(LocalUnsafe)
waitForStall(LocalSafe)
// ... safe may still be < unsafe here

// GOOD — order the stops and *converge* the invariant before exit.
stopAllSequencers()
waitForStall(LocalUnsafe)
unsafeNumber := head(LocalUnsafe).Number
cl.Reached(LocalSafe, unsafeNumber, 30*time.Second)
stopAllBatchers()
require.Equal(t, head(LocalSafe).Number, head(LocalUnsafe).Number)
```

**Examples:** [#20580](https://github.com/ethereum-optimism/optimism/pull/20580). **Lint:** no.

### F10 — Defer ordering that deadlocks cleanup

```go
// BAD — source order: cancel deferred first, Wait deferred second.
// Defers are LIFO: the *later*-deferred Wait pops first, blocking on a
// goroutine that is itself waiting on the context — which cancel hasn't
// fired yet. Deadlock until package timeout.
ctx, cancel := context.WithCancel(t.Context())
defer cancel()       // registered first → runs LAST
wg.Add(1)
go collector(ctx, &wg)
defer wg.Wait()      // registered last → runs FIRST → blocks → deadlock

// GOOD — single defer, explicit order.
ctx, cancel := context.WithCancel(t.Context())
wg.Add(1)
go collector(ctx, &wg)
defer func() { cancel(); wg.Wait() }()
```

**Examples:** [#20600](https://github.com/ethereum-optimism/optimism/pull/20600). **Lint:** `flake-defer-cancel-before-wait`.

### F11 — Background test fixture races with the test on shared resources

The honest proposer and the test both try to create dispute games at the same factory timestamp. Game UUIDs are deterministic in `(gameType, rootClaim, extraData)`, so concurrent creation collides with `GameAlreadyExists`.

```go
// BAD — preset starts a real proposer that runs concurrently with the test's game creation.
sys := presets.NewSuperFaultProofs(t)
sys.DGF.CreateGame(...)   // races with proposer's autonomous game creation

// GOOD — opt out of the background actor the test doesn't need.
sys := presets.NewSuperFaultProofs(t, presets.WithoutHonestProposer())
sys.DGF.CreateGame(...)
```

The general lesson: **a preset should be a minimal universe**. If your test does not exercise a component, the preset should not run it. Add an opt-out, don't try to coexist with the racing background actor.

**Examples:** [#20575](https://github.com/ethereum-optimism/optimism/pull/20575), [#20574](https://github.com/ethereum-optimism/optimism/issues/20574). **Lint:** no.

### F12 — Speculative event treated as final

```go
// BAD — first flashblock containing tx may be superseded; tx lands in next block.
fb := stream.WaitFor(func(fb Flashblock) bool {
    return fb.Contains(txHash)
})
require.Equal(t, expectedBlock, fb.BlockNumber)

// GOOD — collect candidates, filter on confirmed inclusion block.
flashes := stream.Collect(20*time.Second)
receipt := tx.WaitForReceipt()
matching := filterByBlock(flashes, receipt.BlockNumber)
require.NotEmpty(t, matching)
```

**Examples:** [#20066](https://github.com/ethereum-optimism/optimism/pull/20066), [#19976](https://github.com/ethereum-optimism/optimism/pull/19976). **Lint:** no.

### F13 — Concurrent tx submissions sharing a nonce source

```go
// BAD — alice submits two txs concurrently without nonce coordination; second
// gets stale nonce 0 while first already used nonce 0.
go alice.Transfer(bob, amount)
go alice.Transfer(carol, amount)

// GOOD — serialize, or use distinct funded EOAs.
g, _ := errgroup.WithContext(t.Context())
g.Go(func() error { return sys.NewFundedEOA().Transfer(bob, amount) })
g.Go(func() error { return sys.NewFundedEOA().Transfer(carol, amount) })
require.NoError(t, g.Wait())
```

**Examples:** [#20345](https://github.com/ethereum-optimism/optimism/issues/20345). **Lint:** no (semantic).

### F14 — `MarkFlaky` as a band-aid instead of a fix

`MarkFlaky` is an *escalation*, not a resolution. Mark a test flaky **only** when:

1. A `C-flake` GitHub issue exists with a reproducible failure log.
2. The mark cites the issue number in a comment.
3. The team owning the test acknowledged ownership.

`MarkFlaky` without an owning issue is invisible debt. Lint flags `MarkFlaky` calls that don't reference an issue.

**Examples:** [#20200](https://github.com/ethereum-optimism/optimism/pull/20200). **Lint:** `flake-markflaky-without-issue`.

## Reviewer checklist

When reviewing a PR that touches `op-acceptance-tests/` or `op-devstack/`, ask:

- [ ] **F1**: Any `require.*` / `assert.*` calls inside a `Eventually`/`Until`/`retry.Do` callback? They should be `if err != nil { return false }` instead.
- [ ] **F2**: Any `require.*` inside `go func(){...}()`? Goroutine assertions hang the test on failure.
- [ ] **F3**: Any "read X, then use X in next RPC" pattern? Could X be stale?
- [ ] **F4**: Every assertion in the test has a preceding wait on its *exact* postcondition (not a proxy like "sequencer started", "CL reached CrossSafe", or "supervisor validated timestamp" when the assertion reads EL state).
- [ ] **F5**: `Await*`/`*Completed`/`*Done`/`Stop`/`Reset` methods — do they guarantee the *work* happened and hidden state is drained, or just that the signal/API call returned?
- [ ] **F6**: Manual sync checks: any `WithoutCheck` followed by a hand-rolled budget? Use the auto-budgeted preset.
- [ ] **F7**: Sync checks: is the reference value sampled *inside* the retry loop, or captured once outside?
- [ ] **F8**: Any `time.Sleep` in test code? Replace with `WaitFor*`.
- [ ] **F9**: Two independent waits — is the invariant between them asserted on exit?
- [ ] **F10**: Multiple defers — does the LIFO order produce a deadlock? Collapse to one closure.
- [ ] **F11**: Does the preset run components the test does not exercise? Consider an opt-out.
- [ ] **F12**: Any "first matching event" logic on a stream that can supersede entries?
- [ ] **F13**: Concurrent operations sharing nonces, ports, or other unique resources?
- [ ] **F14**: New `MarkFlaky` → linked C-flake issue and owner?

## When a flake report comes in

1. **File a `C-flake` issue** with the CircleCI link, the failing assertion, and the surrounding log context. Include CircleCI Insights flake-history if available.
2. **Search this doc** for the failure shape — most flakes are recurrences. If you find one, tag the issue with the F-number.
3. **If new shape**: after the fix lands, **add a new F-entry** to this doc with the PR number. The catalogue is the institutional memory.
4. **If recurring**: ask whether the existing lint rule should be tightened, or whether the catalogue entry needs a sharper example. The point of the catalogue is to make the next occurrence catchable.

## Static enforcement

Lint rules for the catchable patterns live in [`.semgrep/rules/go-acceptance-test-flakes.yaml`](../../.semgrep/rules/go-acceptance-test-flakes.yaml) and run in CI under the `semgrep-scan-local` job. Failing rules block merge.

The rules are *deliberately scoped* to `op-acceptance-tests/` and `op-devstack/` paths — they encode test-specific conventions, not generic Go style.
