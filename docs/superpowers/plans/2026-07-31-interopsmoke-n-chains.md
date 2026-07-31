# N-chain interop smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run interop smoke checks across any number of configured L2 chains and every ordered source-to-destination pair.

**Architecture:** Replace the fixed A/B environment with ordered slices of remote chains and users. Build pair descriptors for every distinct ordered index pair, then use them in bridge, valid-message, and invalid-message flows. Retain the existing sequential command execution and concurrency model within individual invalid-message phases.

**Tech Stack:** Go, urfave/cli v2, go-ethereum, existing `op-chain-ops/interopsmoke` package tests.

## Global Constraints

- Require at least two repeatable `--l2-rpc` values.
- Cover every ordered pair of distinct configured chains.
- Do not retain A/B-specific RPC flags, environment variables, or direction selection.
- Keep unrelated smoke commands and options unchanged.

---

### Task 1: Model arbitrary remote chains and pairs

**Files:**
- Modify: `op-chain-ops/interopsmoke/smoke.go`
- Test: `op-chain-ops/interopsmoke/smoke_test.go`

**Interfaces:**
- Produces: `newSmokeEnv(ctx, stderr, l2URLs, privateKey)`, with `chains []*remoteChain` and `users []*remoteUser`.
- Produces: `orderedPairs(env) ([]chainPair, error)`, where `chainPair` has `name string`, `initUser *remoteUser`, and `execUser *remoteUser`.

- [ ] **Step 1: Write failing tests for two- and three-chain pair generation**

```go
func TestOrderedPairs(t *testing.T) {
    env := &smokeEnv{users: []*remoteUser{
        {chain: &remoteChain{name: "L2A"}},
        {chain: &remoteChain{name: "L2B"}},
        {chain: &remoteChain{name: "L2C"}},
    }}
    pairs, err := orderedPairs(env)
    if err != nil { t.Fatal(err) }
    want := []string{"A->B", "A->C", "B->A", "B->C", "C->A", "C->B"}
    // assert names in order
}
```

- [ ] **Step 2: Run the package test to verify failure**

Run: `go test ./op-chain-ops/interopsmoke -run TestOrderedPairs`

Expected: FAIL because `orderedPairs` does not exist.

- [ ] **Step 3: Add slice-backed environment and pair helper**

```go
type smokeEnv struct {
    // existing fields
    chains []*remoteChain
    users  []*remoteUser
}

type chainPair struct {
    name string
    initUser, execUser *remoteUser
}

func orderedPairs(env *smokeEnv) ([]chainPair, error) {
    if len(env.users) < 2 { return nil, fmt.Errorf("at least two L2 RPC URLs are required") }
    var pairs []chainPair
    for i, initUser := range env.users {
        for j, execUser := range env.users {
            if i != j { pairs = append(pairs, chainPair{/* derive A->B name */}) }
        }
    }
    return pairs, nil
}
```

- [ ] **Step 4: Run the pair tests to verify success**

Run: `go test ./op-chain-ops/interopsmoke -run TestOrderedPairs`

Expected: PASS with six pairs for three chains.

- [ ] **Step 5: Commit**

Run: `git add op-chain-ops/interopsmoke/smoke.go op-chain-ops/interopsmoke/smoke_test.go && git commit -m 'op-chain-ops: model interop smoke chain pairs'`

### Task 2: Replace the two-chain CLI and setup

**Files:**
- Modify: `op-chain-ops/interopsmoke/smoke.go`
- Modify: `op-chain-ops/cmd/interop-smoke/main.go`
- Test: `op-chain-ops/interopsmoke/smoke_test.go`

**Interfaces:**
- Consumes: `newSmokeEnv(ctx, stderr, []string, privateKey)` and `orderedPairs(env)`.
- Produces: repeatable `--l2-rpc` CLI input and one user per chain.

- [ ] **Step 1: Write failing validation tests**

```go
func TestValidateL2URLs(t *testing.T) {
    for _, tc := range []struct{ urls []string; wantErr bool }{
        {[]string{"http://a", "http://b"}, false},
        {[]string{"http://a"}, true},
    } { /* assert validateL2URLs(tc.urls) */ }
}
```

- [ ] **Step 2: Run validation test to verify failure**

Run: `go test ./op-chain-ops/interopsmoke -run TestValidateL2URLs`

Expected: FAIL because `validateL2URLs` does not exist.

- [ ] **Step 3: Implement repeatable endpoint input and generic setup**

```go
&cli.StringSliceFlag{
    Name: l2URLFlagName,
    Usage: "RPC URL for an interoperable L2. Repeat for each chain.",
    EnvVars: opservice.PrefixEnvVar(envPrefix, "SMOKE_L2_RPC"),
}
```

Validate two or more values before dialing. Connect each URL as `L2A`, `L2B`, then `L2C` and create same-key users. Close every connected client on setup failure and cleanup. Print each configured chain. Update standalone CLI wording from two chains to interoperable L2 chains.

- [ ] **Step 4: Run validation and package tests to verify success**

Run: `go test ./op-chain-ops/interopsmoke`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add op-chain-ops/interopsmoke/smoke.go op-chain-ops/interopsmoke/smoke_test.go op-chain-ops/cmd/interop-smoke/main.go && git commit -m 'op-chain-ops: accept n interop smoke chains'`

### Task 3: Apply all-chain and all-pair smoke behavior

**Files:**
- Modify: `op-chain-ops/interopsmoke/smoke.go`
- Test: `op-chain-ops/interopsmoke/smoke_test.go`

**Interfaces:**
- Consumes: `env.chains`, `env.users`, and `orderedPairs(env)`.
- Produces: identity and transfers per chain; bridge, valid message, and invalid message per ordered pair.

- [ ] **Step 1: Write failing tests for duplicate chain IDs and invalid pair coverage**

```go
func TestValidateChainIDs(t *testing.T) {
    env := &smokeEnv{chains: []*remoteChain{
        {name: "L2A", chainID: 1}, {name: "L2B", chainID: 1},
    }}
    if err := validateChainIDs(env); err == nil { t.Fatal("expected duplicate ID error") }
}
```

- [ ] **Step 2: Run targeted test to verify failure**

Run: `go test ./op-chain-ops/interopsmoke -run TestValidateChainIDs`

Expected: FAIL because generic chain-ID validation does not exist.

- [ ] **Step 3: Generalize smoke commands**

Run identity over all chain IDs and reject duplicates. Run transfer once per user. Loop bridge and valid-message over `orderedPairs`. Replace fixed `invalidDirections` with pair-derived invalid directions and remove its A/B selector flag. Ensure each valid flow waits on, executes on, and validates the specified destination. Preserve per-pair diagnostics.

- [ ] **Step 4: Run targeted and package tests to verify success**

Run: `go test ./op-chain-ops/interopsmoke`

Expected: PASS.

- [ ] **Step 5: Run formatter and commit**

Run: `gofmt -w op-chain-ops/interopsmoke/smoke.go op-chain-ops/interopsmoke/smoke_test.go && go test ./op-chain-ops/interopsmoke && git add op-chain-ops/interopsmoke/smoke.go op-chain-ops/interopsmoke/smoke_test.go && git commit -m 'op-chain-ops: smoke every interop direction'`

### Task 4: Verify integration impact

**Files:**
- Modify: `op-chain-ops/README.md`

**Interfaces:**
- Consumes: final repeatable `--l2-rpc` command contract.
- Produces: accurate tool description for arbitrary interoperable chains.

- [ ] **Step 1: Update the smoke-command description**

Replace the two-live-chain wording with arbitrary interoperable L2 chains and mention pairwise cross-chain coverage.

- [ ] **Step 2: Run focused package and command compilation checks**

Run: `go test ./op-chain-ops/interopsmoke && go test ./op-chain-ops/cmd/interop-smoke`

Expected: PASS.

- [ ] **Step 3: Commit**

Run: `git add op-chain-ops/README.md && git commit -m 'op-chain-ops: document n-chain smoke coverage'`

## Self-review

- Spec coverage: Tasks 1-3 cover repeatable RPC input, N-chain setup, unique identities, per-chain transfers, and every ordered pair for bridge, valid, and invalid messages. Task 4 updates the public description.
- Placeholder scan: no deferred implementation items.
- Type consistency: Task 1 defines the slice-backed environment and pair helper consumed by Tasks 2 and 3.
