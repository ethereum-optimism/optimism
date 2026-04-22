# Writing Acceptance Tests

Guidance for AI agents writing new acceptance tests in the Optimism monorepo. For building and running them, see [acceptance-tests.md](acceptance-tests.md).

## Philosophy

An acceptance test exists to describe a user-visible behaviour of the OP Stack and fail loudly when that behaviour breaks. A reader who is not a domain expert should be able to open any test in `op-acceptance-tests/tests/` and understand what the system is supposed to do, without running anything.

Tests express *requirements*. The DSL expresses *how those requirements are checked*. When the technical details change, the DSL is updated once and every test benefits — including tests added in the future. When a test is flaky, the fix usually belongs in the DSL (a better wait, a missing precondition), not in the test.

This guide covers both sides: how to write a test that reads as a requirement, and how to grow the DSL so future tests stay that way.

## Guiding Principles

### Keep Tests Simple, Even If That Makes the DSL Complex

Complexity in a test is duplicated across every test that follows. Complexity in the DSL is centralised and encapsulated. Push difficulty downward.

The DSL is not plain English and should not try to be. Its domain experts are test authors, not non-technical readers. A statement should clearly describe *what* it is doing without reading like a sentence.

### Consistency Over Cleverness

The "language" of the DSL emerges from consistent naming and structure. Follow established patterns even when a new one would be marginally nicer for your specific test — the cognitive cost of divergent patterns outweighs the win.

If the pattern you need does not exist, extend the DSL rather than invent a one-off in the test file.

### One Behaviour Per Test

A test sets up the minimum state, performs the minimum action, and asserts the minimum outcome required to verify *one* behaviour. If the name wants to say "and", split it.

```go
// Good: one behaviour, one failure mode
func TestTransferMovesFunds(gt *testing.T) { ... }
func TestTransferChargesGasToSender(gt *testing.T) { ... }

// Bad: ambiguous failure
func TestTransferMovesFundsAndChargesGasAndUpdatesNonce(gt *testing.T) { ... }
```

When two behaviours always change together, the second is part of the first behaviour's definition — fold it into an action or verification method on the DSL, not a second assertion in the test body.

### Plain-English Test Names

Test names describe the user-visible behaviour, not the implementation.

```go
// Good
func TestCrossChainWithdrawalFinalisesAfterProofWindow(gt *testing.T) { ... }
func TestSequencerHaltsWhenBatcherLagsBeyondSafeHead(gt *testing.T) { ... }

// Bad
func TestFlow1(gt *testing.T) { ... }
func Test_L2_CL_Adv_v2(gt *testing.T) { ... }
```

A name that requires reading the body to interpret is broken — rename it.

## Test Structure

Most acceptance tests follow the same shape:

```go
func TestSomething(gt *testing.T) {
    t := devtest.ParallelT(gt)
    sys := presets.NewMinimal(t)

    // 1. Arrange: seed state via DSL entry points (users, contracts, nodes)
    alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
    bob   := sys.Wallet.NewEOA(sys.L2EL)

    // 2. Act: call a single DSL action method
    alice.Transfer(bob.Address(), eth.OneHundredthEther)

    // 3. Assert: verify the user-visible outcome via DSL verification methods
    bob.VerifyBalance(eth.OneHundredthEther)
}
```

Keep the test body at this level. If you find yourself reaching into low-level clients, RPC calls, or raw receipts to build assertions, that is a signal the DSL needs a new method — not that the test needs more lines.

## DSL Patterns to Follow

### Action Methods Do Three Things

1. Check (and if needed, wait for) preconditions.
2. Perform the action and let the system fully process its effects.
3. Sanity-assert the action completed, so tests fail fast when something is clearly wrong. Options can expose more specific assertions.

As a test author, this means you should be able to call an action once and trust that what it says it did, it did. If that isn't true for the action you need, fix the DSL method.

### Verification Methods Include Waits

Verification methods in the DSL do the fetching, waiting, retrying, *and* asserting. Tests should never need to build their own wait/retry loops around a raw getter.

Use verification methods only to assert the behaviour the test is actually covering. Do **not** re-verify that setup worked — that belongs inside the setup action method. Extra verifications in the test body obscure intent and increase the number of places that need updating when behaviour changes.

### Prefer Verification Methods Over Getters

The system state is asynchronous; fetching raw values and asserting on them creates flakes. A verification method bundles the fetch with a bounded wait and an assertion.

```go
// Avoid: async state fetched and compared directly
bal := node.GetBalance(user)
require.Equal(t, expected, bal)

// Good
node.VerifyBalance(user, expected)

// Better: let the entry point decide where to verify
user.VerifyBalance(expected) // e.g. verifies across every node
```

Returning *objects* that represent entities in the system is fine — they expose further action and verification methods:

```go
claim   := game.RootClaim()
counter := claim.VerifyCountered()   // waits for op-challenger to counter
counter.VerifyClaimant(honestChallenger)
counter.Attack()
```

### Required vs Optional Arguments

Required inputs are normal parameters — the type system enforces presence. Optional inputs use a config struct plus a vararg of functions that mutate it. `With*` helpers exist only for the most common knobs; tests usually pass a one-off function that sets everything they need at once.

```go
alice.Transfer(bob.Address(), eth.OneEther,
    func(opts *TransferOpts) {
        opts.GasLimit = 100_000
        opts.AccessList = myAccessList
    },
)
```

## No Sleeps, No Retries, No Flakes

Reliability comes from waiting for the *right* condition, not from waiting longer.

**Banned:**
- `time.Sleep(...)`, fixed delays, "give it a second" waits.
- Hand-rolled retry loops in tests.
- Using `MarkFlaky()` as a substitute for fixing the underlying missing wait.

**Required:**
- Every wait targets a specific post-condition (a balance, a block number, an event appearing, a head advancing).
- Every wait has a bounded timeout.
- A failed wait produces an actionable message describing what was expected.

```go
// Bad: timing-based hope
alice.Transfer(bobAddr, amount)
time.Sleep(5 * time.Second)
require.Equal(t, expected, bob.GetBalance())

// Good: wait for the real post-condition
alice.Transfer(bobAddr, amount)
bob.WaitForBalance(expected)
```

If the DSL lacks a wait for the condition you care about, add one — don't sprinkle `time.Sleep` into the test. If a test only passes on rerun, it is broken, not "flaky"; treat the retry as a missing post-condition somewhere upstream and find it.

## No Test-Only Branches in Production Code

Production code paths and test code paths must be the same paths. The only acceptable variation points are explicit, observable seams — preset selection, DSL-injected doubles, launch flags read once at startup.

**Banned:**
- Booleans on production types that change behaviour when set by a test (`isTest`, `skipChecksInTests`).
- Test-only public methods on production structs.
- `if os.Getenv("UNDER_TEST") != ""` branches scattered through business logic.

If you find yourself wanting one of these to make a test pass, the real fix is at the DSL or preset layer — seed different state, use a different preset, inject a different component via the existing provider. Once production code branches on "am I being tested?", the test stops covering the production path and bugs hide in the gap.

## Logging

DSL methods (including ones you add) should log what they are doing. Waiters should log what they are waiting for and the current state of the system on every poll cycle. When an acceptance test fails in CI, the logs are often the only evidence — make them speak.

Inside a test body, prefer expressive DSL calls over scattered `t.Log` statements. If you need a comment or log line to explain what a block of test code is doing, the DSL method is either poorly named or missing.

## Self-Sufficient Failures

A failing test must give the reader enough information to diagnose without re-running.

- **Actionable assertion messages.** Name the expectation: `"expected L2 balance to reflect transfer minus gas"` beats `""`.
- **Deterministic fixtures.** Fixed seeds, fixed addresses where possible, reproducible block/chain setup. A failure should reproduce on the next run unless production code changed.
- **Persistent artefacts.** `just acceptance-test` already writes logs to `op-acceptance-tests/logs/testrun-<timestamp>/` (see [acceptance-tests.md](acceptance-tests.md#log-output-acceptance-test-only)) — make sure the log output your test produces via the DSL is enough to understand a failure from those files alone.

Re-running to "see what happened" is a sign the failure artefact is missing.

## Test Smells

Smells are signals that the DSL is at the wrong level of abstraction for this test. They are not hard rules — they are invitations to stop and consider whether the DSL should grow.

### Comment + Code Block

A comment explaining what a block of test code is doing usually means the DSL method is either misnamed or too low-level.

```go
// Smelly: test explains low-level mechanics in prose
// Deploy test contract
storeProgram := program.New().Sstore(0, 0xbeef).Bytes()
walletv2, err := system.NewWalletV2FromWalletAndChain(ctx, wallet, l2Chain)
require.NoError(t, err)
storeAddr, err := DeployProgram(ctx, walletv2, storeProgram)
require.NoError(t, err)
code, err := l2Client.CodeAt(ctx, storeAddr, nil)
require.NoError(t, err)
require.NotEmpty(t, code, "Store contract not deployed")
require.Equal(t, code, storeProgram, "Store contract code incorrect")

// Good: a DSL method captures the intent
contract := contracts.SStoreContract.Deploy(l2Node, 0xbeef)
```

Not every comment is bad — explaining *why* a specific number was chosen is useful:

```go
// operatorFeeCharged = gasUsed * operatorFeeScalar == 1000 * 5 == 5000
tx.VerifyOperatorFeeCharged(5000)
```

### Reaching Past the DSL

Raw RPC clients, direct receipt manipulation, or imports of internal packages from a test file usually indicate a missing DSL method. Add it, then use it.

### Re-Asserting Setup

A test that verifies its own setup completed ("did the deposit arrive?") before running its real assertions is duplicating work that belongs inside the setup action method. Move it.

### Sharing Setup Across Tests By Copy-Paste

If two tests open with the same ten lines of boilerplate, that boilerplate belongs in a helper or preset — not pasted.

## When to Extend the DSL

Write a new DSL method when:

- A comment is needed to explain what a block of test code is doing.
- Two tests need the same setup or wait pattern.
- A test hand-rolls a `require.Eventually` or retry loop.
- A test reaches past the DSL into low-level clients, RPCs, or internal packages.
- Adding the method would let the test read at the level "describe a behaviour" rather than "drive the implementation".

When extending the DSL, apply the same patterns this guide prescribes for tests: action methods check/act/assert, verification methods wait internally, optional args use the opts-struct vararg pattern, and no method should require the caller to add their own wait or retry loop.

## Checklist (Every Acceptance Test)

- [ ] Test name describes the user-visible behaviour in plain English.
- [ ] Test asserts exactly one behaviour.
- [ ] No `time.Sleep`, no hand-rolled retry loops, no `MarkFlaky` band-aid.
- [ ] No test-only branches added to production code to make this test pass.
- [ ] Setup is not re-verified in the test body.
- [ ] Fixtures are deterministic.
- [ ] Waits and assertions go through DSL methods, not raw clients.
- [ ] If a new pattern was needed, it was added to the DSL, not inlined in the test.
- [ ] Assertion and log messages are actionable — a CI failure can be diagnosed from logs alone.
