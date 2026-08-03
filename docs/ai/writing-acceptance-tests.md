# Writing Acceptance Tests

Guidance for AI agents writing new acceptance tests in the Optimism monorepo. For building and running them, see [acceptance-tests.md](acceptance-tests.md).

## Philosophy

An acceptance test exists to describe a user-visible behaviour of the OP Stack and fail loudly when that behaviour breaks. A reader who is not a domain expert should be able to open any test in `op-acceptance-tests/tests/` and understand what the system is supposed to do, without running anything.

Tests express *requirements*. The DSL exposes reusable domain operations and hides transport details, asynchronous waits, and other mechanics. Tests retain scenario-specific policy and assertions so a reader can see what behaviour is required without opening the DSL. When shared mechanics change, the DSL is updated once and every test benefits. When a test is flaky, the fix usually belongs in a reusable wait or precondition rather than in timing code in the test.

This guide covers both sides: how to write a test that reads as a requirement, and how to grow the DSL so future tests stay that way.

## Guiding Principles

### Keep Tests Simple Without Hiding the Requirement

Complex reusable mechanics in a test are duplicated across every test that follows. Centralise those mechanics in the DSL and push transport details, retries, and stable domain operations downward.

Do not move an entire one-off verification into the DSL merely to shorten a test. A method that chooses the inputs, defines the expected policy, and performs every assertion can hide the requirement it is meant to clarify. The test should still make the action and expected outcome visible.

The DSL is not plain English and should not try to be. Its domain experts are test authors, not non-technical readers. A statement should clearly describe *what* it is doing without reading like a sentence.

### Consistency Over Cleverness

The "language" of the DSL emerges from consistent naming and structure. Follow established patterns even when a new one would be marginally nicer for your specific test — the cognitive cost of divergent patterns outweighs the win.

If a reusable operation or wait is missing, extend the DSL rather than reaching into raw clients. If only a verification policy is shared by one or a few tests in the same package, prefer a package-local test helper over adding it to the repository-wide DSL.

### One Behaviour Per Test

A test sets up the minimum state, performs the minimum action, and asserts the minimum outcome required to verify *one* behaviour. If the name wants to say "and", split it.

```go
// Good: one behaviour, one failure mode
func TestTransferMovesFunds(gt *testing.T) { ... }
func TestTransferChargesGasToSender(gt *testing.T) { ... }

// Bad: ambiguous failure
func TestTransferMovesFundsAndChargesGasAndUpdatesNonce(gt *testing.T) { ... }
```

When two behaviours always change together, the second is part of the first behaviour's definition — keep them in the same assertion flow. If that flow is reused, place it at the narrowest useful scope rather than automatically adding it to the DSL.

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

Keep the test body at this level. If you find yourself reaching into low-level clients, RPC calls, or raw receipts, that is a signal the DSL is missing an operation or typed API. The resulting value or error may still be asserted in the test when that assertion is the behaviour under test.

## DSL Patterns to Follow

### Action Methods Do Three Things

1. Check (and if needed, wait for) preconditions.
2. Perform the action and let the system fully process its effects.
3. Sanity-assert the action completed, so tests fail fast when something is clearly wrong. Options can expose more specific assertions.

As a test author, this means you should be able to call an action once and trust that what it says it did, it did. If that isn't true for the action you need, fix the DSL method.

### Verification Methods Include Waits

When a verification belongs in the DSL, it does the fetching, waiting, retrying, *and* asserting. Tests should never need to build their own wait/retry loops around a raw getter. This does not mean every assertion needs a DSL verification method: scenario-specific assertions can remain in the test or a package-local helper while using DSL operations and wait primitives.

Use verification methods only to assert the behaviour the test is actually covering. Do **not** re-verify that setup worked — that belongs inside the setup action method. Extra verifications in the test body obscure intent and increase the number of places that need updating when behaviour changes.

### Put Verification at the Narrowest Useful Scope

Choose where a verification lives based on how broadly it is useful:

- Used by one test: keep the assertion in that test, or in a small file-local helper if needed for readability.
- Used by a few tests in one package: share it in a package-local `_test.go` helper.
- Used across packages, or representing a stable domain invariant: make it a DSL verification method.

Even repeated verification often belongs at package scope because acceptance-test policy is usually shared only by closely related tests. The DSL should provide the thin capability underneath it. For example, expose typed methods for calling a proxied sequencer API, but let the proxy acceptance test choose which API families to probe and assert which calls must succeed or fail.

Avoid DSL methods that encode a whole test in a name such as `VerifyAllProxyBehaviour`. They obscure which calls define coverage and make unrelated tests depend on scenario-specific policy.

```go
// Avoid: the test's coverage policy is hidden inside a repository-wide helper.
leader.VerifyProxyServesAllRequiredAPIs()

// Better: the DSL exposes the capability; the test shows the requirement.
api := leader.SequencerAPI()
_, err := api.SyncStatus()
require.NoError(t, err)
```

### Prefer Waiting Operations Over Snapshot Getters

The system state is asynchronous; fetching raw values and immediately asserting on them creates flakes. A reusable verification method can bundle the fetch with a bounded wait and an assertion; a package-local verification helper can compose the same DSL wait primitives.

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

For the catalogue of recurring flake patterns observed in CI — with the static lint rules that catch them and the reviewer checklist for the ones that lint can't — see [flake-prevention.md](flake-prevention.md).

## No Test-Only Branches in Production Code

Production code paths and test code paths must be the same paths. The only acceptable variation points are explicit, observable seams — preset selection, DSL-injected doubles, launch flags read once at startup.

**Banned:**
- Booleans on production types that change behaviour when set by a test (`isTest`, `skipChecksInTests`).
- Test-only public methods on production structs.
- `if os.Getenv("UNDER_TEST") != ""` branches scattered through business logic.

If you find yourself wanting one of these to make a test pass, the real fix is at the DSL or preset layer — seed different state, use a different preset, inject a different component via the existing provider. Once production code branches on "am I being tested?", the test stops covering the production path and bugs hide in the gap.

## Logging

DSL methods (including ones you add) should log what they are doing. Waiters should log what they are waiting for and the current state of the system on every poll cycle. When an acceptance test fails in CI, the logs are often the only evidence — make them speak.

Inside a test body, prefer expressive DSL calls over scattered `t.Log` statements. If you need a comment or log line to explain mechanics, the underlying DSL capability or a package-local helper may be poorly named or missing. Comments that explain why an outcome matters should remain in the test.

## Self-Sufficient Failures

A failing test must give the reader enough information to diagnose without re-running.

- **Actionable assertion messages.** Name the expectation: `"expected L2 balance to reflect transfer minus gas"` beats `""`.
- **Deterministic fixtures.** Fixed seeds, fixed addresses where possible, reproducible block/chain setup. A failure should reproduce on the next run unless production code changed.
- **Persistent artefacts.** `just acceptance-test` already writes logs to `op-acceptance-tests/logs/testrun-<timestamp>/` (see [acceptance-tests.md](acceptance-tests.md#log-output-acceptance-test-only)) — make sure the log output your test produces via the DSL is enough to understand a failure from those files alone.

Re-running to "see what happened" is a sign the failure artefact is missing.

## Test Smells

Smells are signals that test support is at the wrong level of abstraction. They are not hard rules — they are invitations to consider whether a reusable capability belongs in the DSL or scenario-specific composition belongs in a package-local helper.

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

- A reusable domain operation or typed API is missing.
- Multiple packages need the same setup, wait, or stable domain invariant.
- A test hand-rolls a `require.Eventually` or retry loop that belongs in a reusable wait primitive.
- A test reaches past the DSL into low-level clients, RPCs, or internal packages.
- Adding the method would let the test read at the level "describe a behaviour" rather than "drive the implementation" without hiding the expected outcome.

Do not add a DSL method solely because one or two tests in the same package share an assertion. Prefer a package-local test helper for that verification and add only the underlying reusable capability to the DSL.

When extending the DSL, apply the same patterns this guide prescribes for tests: action methods check/act/assert, verification methods wait internally, optional args use the opts-struct vararg pattern, and no method should require the caller to add their own wait or retry loop.

## Checklist (Every Acceptance Test)

- [ ] Test name describes the user-visible behaviour in plain English.
- [ ] Test asserts exactly one behaviour.
- [ ] No `time.Sleep`, no hand-rolled retry loops, no `MarkFlaky` band-aid.
- [ ] No test-only branches added to production code to make this test pass.
- [ ] Setup is not re-verified in the test body.
- [ ] Fixtures are deterministic.
- [ ] Tests use DSL operations and wait primitives rather than raw clients or hand-rolled retries.
- [ ] Scenario-specific assertions remain visible in the test or a package-local helper.
- [ ] Reusable capabilities live in the DSL; package-local verification policy does not leak into it.
- [ ] Assertion and log messages are actionable — a CI failure can be diagnosed from logs alone.
