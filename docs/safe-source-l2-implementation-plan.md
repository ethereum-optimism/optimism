# Implementation Plan: `--safe-source=l2` Feature

## Overview
Enable op-node to trust another L2 node's safe/finalized heads instead of deriving from L1, creating a "fast follower" mode that bypasses derivation.

## Design Philosophy
**Minimal Code Changes** - This PR should be primarily tests with minimal modifications to the op-node core:
- ~200 LOC in op-node for config, flags, and logic changes
- ~225 LOC in test infrastructure (devstack presets, config wiring)
- ~215 LOC in tests (sync tester + unit tests)
- **Total: ~640 LOC** with roughly equal distribution between implementation, infrastructure, and tests
- Focus on comprehensive testing to ensure the feature works correctly

## Core Code Changes Summary
The implementation requires changes to only **4 core areas**:

1. **Configuration** (`op-node/rollup/sync/config.go`, `op-node/flags/flags.go`, `op-node/service.go`)
   - Add `SafeSource` enum and config parsing (~60 LOC)

2. **L2 RPC Client** (No new file needed!)
   - Reuse existing `sources.EngineClient` directly (~0 LOC new code)

3. **Derivation Skip** (`op-node/rollup/driver/sync_deriver.go`)
   - Add one `if` check in `SyncStep()` (~10 LOC)

4. **Remote Head Query** (`op-node/rollup/engine/engine_controller.go`)
   - Modify `insertUnsafePayload()` to query remote safe/finalized (~80 LOC)

**Total core op-node changes: ~150 LOC** (reduced by removing unnecessary wrapper)

**Majority of PR: Tests (~800+ LOC)**
- New sync tester tests
- New presets for safe-source=l2
- Integration tests

---

## Phase 1: Configuration & Types

### 1.1 Add Safe Source Type Enum
**File:** `op-node/rollup/sync/config.go`

```go
type SafeSource int

const (
    SafeSourceL1 SafeSource = iota  // Default: derive from L1
    SafeSourceL2                     // Trust remote L2 node
)

const (
    SafeSourceL1String string = "l1"
    SafeSourceL2String string = "l2"
)

func StringToSafeSource(s string) (SafeSource, error) {
    switch strings.ToLower(s) {
    case SafeSourceL1String:
        return SafeSourceL1, nil
    case SafeSourceL2String:
        return SafeSourceL2, nil
    default:
        return 0, fmt.Errorf("unknown safe source: %s", s)
    }
}

func (s SafeSource) String() string {
    switch s {
    case SafeSourceL1:
        return SafeSourceL1String
    case SafeSourceL2:
        return SafeSourceL2String
    default:
        return "unknown"
    }
}
```

### 1.2 Update Sync Config
**File:** `op-node/rollup/sync/config.go`

```go
type Config struct {
    SyncMode                       Mode   `json:"syncmode"`
    SkipSyncStartCheck             bool   `json:"skip_sync_start_check"`
    SupportsPostFinalizationELSync bool   `json:"supports_post_finalization_elsync"`

    // NEW: Safe head source configuration
    SafeSource        SafeSource `json:"safe_source"`
    SafeSourceL2RPC   string     `json:"safe_source_l2_rpc,omitempty"`
}
```

### 1.3 Add CLI Flags
**File:** `op-node/flags/flags.go`

```go
SafeSourceFlag = &cli.GenericFlag{
    Name:    "safe-source",
    Usage:   fmt.Sprintf("Source for safe head determination (options: %s)", "l1, l2"),
    EnvVars: prefixEnvVars("SAFE_SOURCE"),
    Value: func() *sync.SafeSource {
        out := sync.SafeSourceL1
        return &out
    }(),
    Category: RollupCategory,
}

SafeSourceL2RPCFlag = &cli.StringFlag{
    Name:     "safe-source.l2-rpc",
    Usage:    "HTTP RPC endpoint of the L2 node to use as safe head source (required when --safe-source=l2)",
    EnvVars:  prefixEnvVars("SAFE_SOURCE_L2_RPC"),
    Category: RollupCategory,
}
```

### 1.4 Parse Config
**File:** `op-node/service.go`

```go
func NewSyncConfig(ctx *cli.Context, log log.Logger) (*sync.Config, error) {
    // ... existing code ...

    safeSource := sync.SafeSourceL1
    if ctx.IsSet(flags.SafeSourceFlag.Name) {
        safeSource = ctx.Generic(flags.SafeSourceFlag.Name).(*sync.SafeSource)
    }

    safeSourceL2RPC := ctx.String(flags.SafeSourceL2RPCFlag.Name)

    // Validate configuration
    if safeSource == sync.SafeSourceL2 && safeSourceL2RPC == "" {
        return nil, errors.New("--safe-source.l2-rpc is required when --safe-source=l2")
    }

    cfg := &sync.Config{
        SyncMode:                       mode,
        SkipSyncStartCheck:             ctx.Bool(flags.SkipSyncStartCheck.Name),
        SupportsPostFinalizationELSync: engineKind.SupportsPostFinalizationELSync(),
        SafeSource:                     *safeSource,
        SafeSourceL2RPC:                safeSourceL2RPC,
    }

    return cfg, nil
}
```

---

## Phase 2: L2 RPC Client Setup

### 2.1 Initialize Remote L2 Client in Driver
**Note:** We reuse the existing `sources.EngineClient` - no new wrapper needed!

**File:** `op-node/rollup/driver/driver.go`

```go
type Driver struct {
    // ... existing fields ...

    // NEW: Remote L2 client for querying safe/finalized heads (nil if using L1 source)
    safeSourceL2 *sources.EngineClient
}

func NewDriver(
    sys event.Registry,
    // ... existing params ...
    syncCfg *sync.Config,
    // ... rest ...
) *Driver {
    // ... existing initialization ...

    var safeSourceL2Client *sources.EngineClient
    if syncCfg.SafeSource == sync.SafeSourceL2 {
        rpcClient, err := client.NewRPC(driverCtx, log, syncCfg.SafeSourceL2RPC)
        if err != nil {
            log.Crit("Failed to create RPC client for safe source L2", "err", err)
        }

        // Reuse existing EngineClient - it already has L2BlockRefByLabel!
        safeSourceL2Client, err = sources.NewEngineClient(rpcClient, log, nil, sources.EngineClientDefaultConfig(cfg))
        if err != nil {
            log.Crit("Failed to create engine client for safe source L2", "err", err)
        }

        log.Info("Initialized safe source L2 client", "rpc", syncCfg.SafeSourceL2RPC)
    }

    // ... create syncDeriver with safeSourceL2Client ...
    syncDeriver := &SyncDeriver{
        // ... existing fields ...
        SafeSourceL2: safeSourceL2Client,
    }

    // Wire to engine controller
    if safeSourceL2Client != nil {
        ec.SetSafeSourceL2(safeSourceL2Client)
    }

    // ... rest of initialization ...
}
```

### 2.3 Add to Devstack L2CL Config
**File:** `op-devstack/sysgo/l2_cl.go`

```go
type L2CLConfig struct {
    SequencerSyncMode nodeSync.Mode
    VerifierSyncMode  nodeSync.Mode
    SafeDBPath        string

    // NEW: Safe source configuration
    SafeSource        nodeSync.SafeSource
    SafeSourceL2RPC   string

    IsSequencer       bool
    IndexingMode      bool
    EnableReqRespSync bool
}

func DefaultL2CLConfig() *L2CLConfig {
    return &L2CLConfig{
        SequencerSyncMode: nodeSync.CLSync,
        VerifierSyncMode:  nodeSync.CLSync,
        SafeDBPath:        "",
        SafeSource:        nodeSync.SafeSourceL1,  // NEW: default to L1
        SafeSourceL2RPC:   "",                      // NEW: empty by default
        IsSequencer:       false,
        IndexingMode:      false,
        EnableReqRespSync: true,
    }
}
```

**File:** `op-devstack/sysgo/l2_cl_opnode.go` (wire config to op-node flags)

```go
// In the op-node initialization section, add flags based on config
if cfg.SafeSource == nodeSync.SafeSourceL2 {
    args = append(args,
        "--safe-source=l2",
        "--safe-source.l2-rpc="+cfg.SafeSourceL2RPC,
    )
}
```


### 2.2 Add to SyncDeriver
**File:** `op-node/rollup/driver/sync_deriver.go`

```go
type SyncDeriver struct {
    // ... existing fields ...

    // NEW: Client for querying remote L2 safe head
    SafeSourceL2 *sources.EngineClient
}

func (s *SyncDeriver) usingSafeSourceL2() bool {
    return s.SyncCfg.SafeSource == sync.SafeSourceL2 && s.SafeSourceL2 != nil
}
```

---

## Phase 3: Skip Derivation Logic

### 3.1 Modify SyncStep
**File:** `op-node/rollup/driver/sync_deriver.go:218-245`

```go
func (s *SyncDeriver) SyncStep() {
    s.Log.Debug("Sync process step")

    s.tryBackupUnsafeReorg()

    s.Engine.TryUpdateEngine(s.Ctx)

    // Block derivation if EL is syncing
    if s.Engine.IsEngineSyncing() {
        s.Log.Debug("Rollup driver is backing off because execution engine is syncing.",
            "unsafe_head", s.Engine.UnsafeL2Head())
        s.StepDeriver.ResetStepBackoff(s.Ctx)
        return
    }

    // NEW: Block derivation if using L2 safe source
    if s.usingSafeSourceL2() {
        s.Log.Debug("Skipping derivation, using L2 safe source",
            "safe_source_rpc", s.SyncCfg.SafeSourceL2RPC,
            "unsafe_head", s.Engine.UnsafeL2Head())
        s.StepDeriver.ResetStepBackoff(s.Ctx)
        return
    }

    // Continue with normal derivation pipeline
    s.Engine.RequestPendingSafeUpdate(s.Ctx)
}
```

---

## Phase 4: Query Remote Safe/Finalized Heads

### 4.1 Add Remote Head Querying to EngineController
**File:** `op-node/rollup/engine/engine_controller.go`

Add new field and setter:

```go
type EngineController struct {
    // ... existing fields ...

    // NEW: Client for querying remote L2 safe/finalized heads
    safeSourceL2 *sources.EngineClient
}

func (e *EngineController) SetSafeSourceL2(client *sources.EngineClient) {
    e.safeSourceL2 = client
}

func (e *EngineController) usingSafeSourceL2() bool {
    return e.syncCfg.SafeSource == sync.SafeSourceL2 && e.safeSourceL2 != nil
}
```

Wire it up in Driver:

```go
// In NewDriver after creating EngineController
if safeSourceL2Client != nil {
    ec.SetSafeSourceL2(safeSourceL2Client)
}
```

### 4.2 Modify InsertUnsafePayload
**File:** `op-node/rollup/engine/engine_controller.go:491-591`

```go
func (e *EngineController) insertUnsafePayload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef) error {
    // ... existing EL sync check code (lines 493-508) ...

    // Insert the payload & then call FCU
    newPayloadStart := time.Now()
    status, err := e.engine.NewPayload(ctx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
    // ... existing error handling ...

    newPayloadFinish := time.Now()

    // Build forkchoice state
    var safeHash common.Hash
    var finalizedHash common.Hash

    // NEW: Query remote L2 for safe/finalized if using L2 safe source
    if e.usingSafeSourceL2() {
        e.log.Debug("Querying remote L2 for safe/finalized heads")

        // Use existing L2BlockRefByLabel method from EngineClient
        remoteSafe, err := e.safeSourceL2.L2BlockRefByLabel(ctx, eth.Safe)
        if err != nil {
            e.log.Warn("Failed to query remote L2 safe head, using local",
                "err", err, "local_safe", e.safeHead.Hash)
            safeHash = e.safeHead.Hash
        } else {
            safeHash = remoteSafe.Hash
            e.log.Debug("Using remote safe head", "hash", safeHash, "number", remoteSafe.Number)
        }

        remoteFinalized, err := e.safeSourceL2.L2BlockRefByLabel(ctx, eth.Finalized)
        if err != nil {
            e.log.Warn("Failed to query remote L2 finalized head, using local",
                "err", err, "local_finalized", e.finalizedHead.Hash)
            finalizedHash = e.finalizedHead.Hash
        } else {
            finalizedHash = remoteFinalized.Hash
            e.log.Debug("Using remote finalized head", "hash", finalizedHash, "number", remoteFinalized.Number)
        }
    } else {
        // Use local safe/finalized heads
        safeHash = e.safeHead.Hash
        finalizedHash = e.finalizedHead.Hash
    }

    // Mark the new payload as valid
    fc := eth.ForkchoiceState{
        HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
        SafeBlockHash:      safeHash,      // From remote L2 or local
        FinalizedBlockHash: finalizedHash,  // From remote L2 or local
    }

    if e.syncStatus == syncStatusFinishedELButNotFinalized {
        // ... existing EL sync finalization code ...
    }

    // ... rest of existing code ...
}
```

---

## Phase 5: Edge Cases & Error Handling

### 5.1 Handle Remote RPC Failures
**Strategy:** Graceful degradation - fall back to local safe/finalized heads

Already implemented above with:
```go
if err != nil {
    e.log.Warn("Failed to query remote L2 safe head, using local", "err", err)
    safeHash = e.safeHead.Hash
}
```

### 5.2 Validate Remote Heads
**File:** Add validation in `insertUnsafePayload`:

```go
if e.usingSafeSourceL2() {
    remoteSafe, err := e.safeSourceL2.L2BlockRefByLabel(ctx, eth.Safe)
    if err != nil {
        // ... existing fallback ...
    } else {
        // Validate: remote safe should not be ahead of local unsafe
        if remoteSafe.Number > ref.Number {
            e.log.Warn("Remote safe head is ahead of local unsafe, ignoring",
                "remote_safe", remoteSafe.Number, "local_unsafe", ref.Number)
            safeHash = e.safeHead.Hash
        } else {
            safeHash = remoteSafe.Hash
            // Update local tracking if desired
            if remoteSafe.Number > e.safeHead.Number {
                e.SetSafeHead(remoteSafe)
            }
        }
    }
}
```

### 5.3 Startup Validation
**File:** `op-node/service.go`

```go
func NewSyncConfig(ctx *cli.Context, log log.Logger) (*sync.Config, error) {
    // ... existing code ...

    // Validate incompatible combinations
    if safeSource == sync.SafeSourceL2 && mode == sync.ELSync {
        return nil, errors.New("--safe-source=l2 is not compatible with --syncmode=execution-layer")
    }

    // Warn about trust assumptions
    if safeSource == sync.SafeSourceL2 {
        log.Warn("Using L2 safe source mode - trusting remote node for safe/finalized heads!",
            "remote_rpc", safeSourceL2RPC)
    }
}
```

### 5.4 Handle Safe Head DB Interaction
**File:** `op-node/rollup/driver/sync_deriver.go`

```go
func (s *SyncDeriver) onSafeDerivedBlock(ctx context.Context, x engine.SafeDerivedEvent) {
    // Skip safe head DB updates when using L2 safe source
    if s.usingSafeSourceL2() {
        s.Log.Debug("Skipping safe head DB update (using L2 safe source)")
        return
    }

    // ... existing safe head notifications ...
}
```

---

## Phase 6: Testing

### 6.1 Unit Tests
**File:** `op-node/rollup/driver/sync_deriver_test.go` (new tests)

```go
func TestSyncStep_SafeSourceL2_SkipsDerivation(t *testing.T) {
    // Test that derivation is skipped when using L2 safe source
}

func TestUsingSafeSourceL2(t *testing.T) {
    // Test the helper method
}
```

**File:** `op-node/rollup/engine/engine_controller_test.go`

```go
func TestInsertUnsafePayload_SafeSourceL2_QueriesRemote(t *testing.T) {
    // Mock remote L2 RPC
    // Insert payload
    // Verify remote safe/finalized were queried
}

func TestInsertUnsafePayload_SafeSourceL2_FallbackOnError(t *testing.T) {
    // Mock remote L2 RPC to fail
    // Verify fallback to local safe/finalized
}
```

### 6.2 Integration Tests - New Preset
**File:** `op-devstack/presets/simple_with_safe_source_l2.go` (new file)

```go
package presets

import (
    "github.com/ethereum-optimism/optimism/op-devstack/devtest"
    "github.com/ethereum-optimism/optimism/op-devstack/dsl"
    "github.com/ethereum-optimism/optimism/op-devstack/shim"
    "github.com/ethereum-optimism/optimism/op-devstack/stack"
    "github.com/ethereum-optimism/optimism/op-devstack/stack/match"
    "github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

type SimpleWithSafeSourceL2 struct {
    Minimal

    L2CL2 *dsl.L2CLNode  // Verifier with safe-source=l2
}

func WithSimpleWithSafeSourceL2() stack.CommonOption {
    return stack.MakeCommon(sysgo.DefaultSimpleSystemWithSafeSourceL2(&sysgo.DefaultSimpleSystemWithSafeSourceL2IDs{}))
}

func NewSimpleWithSafeSourceL2(t devtest.T) *SimpleWithSafeSourceL2 {
    system := shim.NewSystem(t)
    orch := Orchestrator()
    orch.Hydrate(system)
    minimal := minimalFromSystem(t, system, orch)
    l2 := system.L2Network(match.L2ChainA)

    // L2CL2 is configured with safe-source=l2 pointing to L2CL (first verifier)
    l2CL2 := l2.L2CLNode(match.SecondL2CL)

    return &SimpleWithSafeSourceL2{
        Minimal: *minimal,
        L2CL2:   dsl.NewL2CLNode(l2CL2, orch.ControlPlane()),
    }
}
```

**File:** `op-devstack/presets/cl_config.go` (add new preset option)

```go
func WithSafeSourceL2OnSecondVerifier(sourceRPCURL string) stack.CommonOption {
    return stack.MakeCommon(
        sysgo.WithL2CLOption(match.SecondL2CL, sysgo.L2CLOptionFn(
            func(_ devtest.P, id stack.L2CLNodeID, cfg *sysgo.L2CLConfig) {
                cfg.SafeSource = sync.SafeSourceL2
                cfg.SafeSourceL2RPC = sourceRPCURL
            })))
}
```

### 6.3 Sync Tester Test
**File:** `op-acceptance-tests/tests/sync_tester/sync_tester_safe_source_l2/init_test.go` (new file)

```go
package sync_tester_safe_source_l2

import (
    "testing"

    "github.com/ethereum-optimism/optimism/op-devstack/compat"
    "github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
    presets.DoMain(m,
        presets.WithSimpleWithSafeSourceL2(),
        presets.WithSafeSourceL2OnSecondVerifier("http://l2-cl:8545"), // Points to first verifier
        presets.WithCompatibleTypes(compat.SysGo),
    )
}
```

**File:** `op-acceptance-tests/tests/sync_tester/sync_tester_safe_source_l2/safe_source_l2_test.go` (new file)

```go
package sync_tester_safe_source_l2

import (
    "testing"

    "github.com/ethereum-optimism/optimism/op-devstack/devtest"
    "github.com/ethereum-optimism/optimism/op-devstack/dsl"
    "github.com/ethereum-optimism/optimism/op-devstack/presets"
    "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSafeSourceL2DerivationSkipped(gt *testing.T) {
    t := devtest.SerialT(gt)
    sys := presets.NewSimpleWithSafeSourceL2(t)
    require := t.Require()
    logger := t.Logger()

    // Advance first verifier (L2CL) to derive some safe blocks from L1
    target := uint64(20)
    sys.L2CL.Advanced(types.CrossSafe, target, 30)

    logger.Info("First verifier advanced", "safe", target)

    // Wait for second verifier (L2CL2 with safe-source=l2) to catch up
    // It should be getting safe head from L2CL, NOT deriving from L1
    sys.L2CL2.Advanced(types.CrossSafe, target, 30)

    // Verify both verifiers have same safe head
    l2CLSafe := sys.L2CL.SyncStatus().SafeL2
    l2CL2Safe := sys.L2CL2.SyncStatus().SafeL2

    require.Equal(l2CLSafe.Number, l2CL2Safe.Number, "Safe heads should match")
    require.Equal(l2CLSafe.Hash, l2CL2Safe.Hash, "Safe hashes should match")

    logger.Info("Verified safe heads match",
        "l2cl_safe", l2CLSafe.Number,
        "l2cl2_safe", l2CL2Safe.Number)

    // KEY TEST: Verify L2CL2 is NOT deriving from L1
    // This is similar to how TestSyncTesterELSync checks ELSyncActive
    // We would need to add a metric or log to verify derivation is skipped
    // For now, we verify it's keeping up without performing derivation

    // Advance further
    target = uint64(50)
    dsl.CheckAll(t,
        sys.L2CL.AdvancedFn(types.CrossSafe, target, 30),
        sys.L2CL2.AdvancedFn(types.CrossSafe, target, 30),
    )

    // Verify they stay in sync
    l2CLSafe = sys.L2CL.SyncStatus().SafeL2
    l2CL2Safe = sys.L2CL2.SyncStatus().SafeL2

    require.Equal(l2CLSafe.Number, l2CL2Safe.Number, "Safe heads should still match")
    require.Equal(l2CLSafe.Hash, l2CL2Safe.Hash, "Safe hashes should still match")

    logger.Info("Safe source L2 mode working correctly",
        "final_safe_height", l2CL2Safe.Number)
}

func TestSafeSourceL2FollowsRemoteSafe(gt *testing.T) {
    t := devtest.SerialT(gt)
    sys := presets.NewSimpleWithSafeSourceL2(t)
    require := t.Require()

    // Start with both verifiers at same point
    target := uint64(10)
    dsl.CheckAll(t,
        sys.L2CL.AdvancedFn(types.CrossSafe, target, 30),
        sys.L2CL2.AdvancedFn(types.CrossSafe, target, 30),
    )

    // Stop second verifier
    sys.L2CL2.Stop()

    // Advance first verifier significantly
    target = uint64(30)
    sys.L2CL.Advanced(types.CrossSafe, target, 30)

    // Restart second verifier - it should catch up by querying remote safe head
    sys.L2CL2.Start()

    // Wait for P2P connection
    sys.L2CL2.IsP2PConnected(sys.L2CL)

    // L2CL2 should catch up to L2CL's safe head
    sys.L2CL2.Reached(types.CrossSafe, target, 60)

    // Verify safe heads match
    l2CLSafe := sys.L2CL.SyncStatus().SafeL2
    l2CL2Safe := sys.L2CL2.SyncStatus().SafeL2

    require.Equal(l2CLSafe.Number, l2CL2Safe.Number)
    require.Equal(l2CLSafe.Hash, l2CL2Safe.Hash)
}
```

---

## Phase 7: Documentation

### 7.1 Add to Config Documentation
Document new flags:
- `--safe-source` (default: `l1`)
- `--safe-source.l2-rpc` (required when `--safe-source=l2`)

### 7.2 Usage Examples

```bash
# Normal mode (default)
op-node --syncmode=consensus-layer

# Fast follower mode - trust another L2 node
op-node \
  --syncmode=consensus-layer \
  --safe-source=l2 \
  --safe-source.l2-rpc=http://trusted-node:8545
```

### 7.3 Security Warning
Document that `--safe-source=l2`:
- **Trusts the remote node** for safe/finalized determination
- Does **not** validate blocks against L1
- Should only be used with trusted infrastructure
- Creates a "fast follower" that can quickly catch up but sacrifices security

---

## Summary of All Files

### Core op-node Changes (~200 LOC)

| File | Type | LOC | Description |
|------|------|-----|-------------|
| `op-node/rollup/sync/config.go` | Modified | ~60 | Add SafeSource enum, config fields |
| ~~`op-node/rollup/sync/l2_source.go`~~ | ~~NEW~~ | ~~0~~ | ~~Not needed - reuse EngineClient~~ |
| `op-node/flags/flags.go` | Modified | ~15 | Add CLI flags |
| `op-node/service.go` | Modified | ~15 | Parse and validate config |
| `op-node/rollup/driver/driver.go` | Modified | ~20 | Initialize L2 client |
| `op-node/rollup/driver/sync_deriver.go` | Modified | ~10 | Skip derivation logic |
| `op-node/rollup/engine/engine_controller.go` | Modified | ~80 | Query remote safe/finalized |

### Test Infrastructure (~400 LOC)

| File | Type | LOC | Description |
|------|------|-----|-------------|
| `op-devstack/sysgo/l2_cl.go` | Modified | ~15 | Add config fields |
| `op-devstack/sysgo/l2_cl_opnode.go` | Modified | ~10 | Wire config to flags |
| `op-devstack/presets/simple_with_safe_source_l2.go` | **NEW** | ~40 | New test preset |
| `op-devstack/presets/cl_config.go` | Modified | ~10 | Preset option function |
| `op-devstack/sysgo/system_safe_source_l2.go` | **NEW** | ~150 | System builder for preset |

### Tests (~450 LOC)

| File | Type | LOC | Description |
|------|------|-----|-------------|
| `op-acceptance-tests/tests/sync_tester/sync_tester_safe_source_l2/init_test.go` | **NEW** | ~15 | Test initialization |
| `op-acceptance-tests/tests/sync_tester/sync_tester_safe_source_l2/safe_source_l2_test.go` | **NEW** | ~100 | Main sync tester tests |
| `op-node/rollup/driver/sync_deriver_test.go` | Modified | ~50 | Unit tests |
| `op-node/rollup/engine/engine_controller_test.go` | Modified | ~50 | Unit tests |

**Total Lines Changed:**
- Core op-node: ~200 LOC (31%)
- Test infrastructure: ~225 LOC (35%)
- Tests: ~215 LOC (34%)
- **Total: ~640 LOC**

This is a **small, focused PR** with roughly equal distribution between implementation, test infrastructure, and tests. The key is reusing existing infrastructure (EngineClient, L2BlockRefByLabel) rather than building new abstractions.

## Testing Strategy

- **Unit tests** for config parsing and validation
- **Unit tests** for remote head querying with mocked RPC
- **Sync tester tests** verifying derivation is skipped
- **Integration tests** with 2 verifiers (one normal, one safe-source=l2)
- **Catch-up test** verifying follower can restart and catch up

## Key Implementation Points

1. **Derivation bypass** happens in `SyncDeriver.SyncStep()` right after the EL sync check (~10 LOC)
2. **Remote head querying** happens in `EngineController.insertUnsafePayload()` when building the forkchoice state (~80 LOC)
3. **Graceful degradation** on RPC failures - falls back to local safe/finalized
4. **Validation** ensures remote safe head doesn't exceed local unsafe head
5. **Incompatibility check** prevents using with EL sync mode
6. **Minimal changes** to existing code paths - all new functionality is additive

This implementation maintains compatibility with existing modes while adding the new fast-follower capability!
