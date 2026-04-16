# Delete Supervisor Feature

**Author:** AI Tool (op-feature skill)  
**Developer:** Axel Kingsley  
**Date Started:** 2026-04-16  

## Description

Remove the `op-supervisor` standalone binary and service from the Optimism monorepo. The Supervisor has been deprecated (noted in AGENTS.md) and its functionality has been subsumed by `op-supernode` and direct interop integration. This feature involves deleting the `op-supervisor/` directory and all references to it across the codebase, while ensuring existing tests continue to pass.

## Scope Analysis

### What is `op-supervisor`?
A standalone Go service for cross-L2 interop message safety. Provides JSON-RPC APIs (`supervisor` and `admin` namespaces) for checking cross-chain message safety, syncing chain data, and managing interop state. It has its own binary entry point, Docker image, CI configuration, and is consumed by many other packages.

### Impact Surface
- **106 Go files** in `op-supervisor/` itself
- **294 Go files** across the repo reference "supervisor" (case-insensitive)
- **62 non-Go files** reference "supervisor" (Dockerfiles, CI, docs, Rust code, configs)
- **Key consuming packages:** op-node, op-devstack, op-supernode, op-e2e, op-acceptance-tests, op-service, op-challenger, op-proposer, op-conductor, op-interop-mon, op-deployer, op-program, op-chain-ops, rust/kona

### What is NOT being deleted
- `op-supervisor/supervisor/types/` — widely imported; types like `Identifier`, `SafetyLevel`, `ExecutingMessage` are used across the stack
- `op-supervisor/supervisor/backend/depset/` — dependency set logic imported by op-deployer, op-program, op-service, op-e2e
- `op-supervisor/supervisor/backend/db/logs/` — log storage imported by op-supernode, op-interop-filter
- `op-supervisor/supervisor/backend/processors/` — imported by op-supernode, op-interop-filter
- `op-supervisor/supervisor/backend/reads/` — imported by op-supernode
- Other backend sub-packages that have downstream consumers outside `op-supervisor` itself
- Rust `txpool/supervisor` — separate concern (talks to the supervisor RPC, not the Go implementation)

### What IS being deleted
- `op-supervisor/cmd/` — binary entry point
- `op-supervisor/supervisor/service.go` — service lifecycle/wiring
- `op-supervisor/supervisor/entrypoint.go` — Main/MainFn
- `op-supervisor/supervisor/frontend/` — RPC frontend
- `op-supervisor/config/` — standalone config
- `op-supervisor/flags/` — CLI flags
- `op-supervisor/metrics/` — standalone metrics
- Docker image target for op-supervisor
- CI configuration for op-supervisor
- References in root justfile, Makefile
- `op-service/sources/supervisor_client.go` — RPC client
- `op-service/apis/supervisor.go` — API interfaces (if not reused)
- Devstack wiring: `op-devstack/dsl/supervisor.go`, `op-devstack/stack/supervisor.go`
- Conductor supervisor health checks
- Proposer supervisor source
- Interop-mon supervisor endpoints

## Dependency Graph (Deletion Safety)

### Internal-only packages (0 external importers — SAFE TO DELETE):
| Package | Description |
|---------|-------------|
| `op-supervisor/cmd` | Binary entry point |
| `op-supervisor/flags` | CLI flags |
| `op-supervisor/supervisor/frontend` | RPC frontend (QueryFrontend, AdminFrontend) |
| `op-supervisor/supervisor/backend/db` (root) | DB root package |
| `op-supervisor/supervisor/backend/db/fromda` | DA-derived storage |
| `op-supervisor/supervisor/backend/db/entrydb` | Entry DB |
| `op-supervisor/supervisor/backend/db/sync` | DB sync |
| `op-supervisor/supervisor/backend/l1access` | L1 access |
| `op-supervisor/supervisor/backend/rewinder` | Rewinder |
| `op-supervisor/supervisor/backend/superevents` | Internal events |

### Externally-imported packages (MUST KEEP or RELOCATE):
| Package | External importers |
|---------|-------------------|
| `supervisor/types` | 99 files |
| `supervisor/backend/depset` | 42 files |
| `supervisor/backend/processors` | 5 files |
| `supervisor/backend/reads` | 4 files (op-supernode) |
| `supervisor/backend/syncnode` | 4 files (devstack, e2e) |
| `supervisor/backend/db/logs` | 2 files (op-supernode, op-interop-filter) |
| `supervisor/backend/cross` | 2 files (op-program) |
| `supervisor/backend/status` | 1 file (devstack) |
| `supervisor/backend` (root) | 1 file (op-e2e) |
| `op-supervisor/supervisor` (root) | 3 files (devstack, e2e) |
| `op-supervisor/config` | 4 files (devstack, e2e) |
| `op-supervisor/metrics` | 1 file (op-e2e) |

### Downstream supervisor-specific code (candidates for removal):
| Area | Files | What it does |
|------|-------|--------------|
| op-devstack/sysgo/supervisor_op.go | 1 | Wraps SupervisorFromConfig in-process |
| op-devstack/sysgo/multichain_supervisor_runtime.go | 1 | Starts standalone supervisor in test runtime |
| op-devstack/dsl/supervisor.go | 1 | DSL wrapper for supervisor sync status |
| op-devstack/stack/supervisor.go | 1 | Supervisor interface |
| op-e2e/interop/supersystem.go | 1 | E2E supervisor launch |
| op-e2e/actions/interop/dsl/interop.go | 1 | Actions DSL supervisor wiring |
| op-service/apis/supervisor.go | 1 | SupervisorAPI interfaces |
| op-service/sources/supervisor_client.go | 1 | RPC client for supervisor |
| op-service/eth/supervisor_status.go | 1 | Sync status types |
| op-service/dial/dial.go | 1 | DialSupervisorClientWithTimeout |
| op-conductor/health/supervisor.go | 1+ | Supervisor health checks |
| op-proposer source_supervisor.go | 1 | Supervisor as proposer source |
| op-challenger supervisor sync/provider | 2+ | Fault proof supervisor integration |
| op-interop-mon | 5+ | Monitoring via supervisor endpoints |

## Breakdown (Sub-features as commits)

TBD — awaiting developer guidance on approach.

---

## Diary

### Prompt 1: Initial Setup (2026-04-16)
**Prompt:** Checkout latest develop, read AI guidance, read overprompt.md. Feature is "Delete Supervisor" — analyze from perspective of existing tests passing, expect many thousands of lines removable.

**Actions taken:**
- Checked out latest develop branch (fast-forwarded 4 commits)
- Read ~/overprompt.md (op-feature skill)
- Read AGENTS.md — confirmed op-supervisor is marked DEPRECATED
- Read docs/ai/go-dev.md for build/test/lint guidance
- Launched 3 parallel exploration agents:
  1. Explored supervisor directories, types, interfaces, entry points (found 106 Go files in op-supervisor)
  2. Found all 294 Go files referencing supervisor across repo
  3. Found all 62 non-Go files referencing supervisor
- Created this feature diary file

**Key findings:**
- op-supervisor is already marked DEPRECATED in AGENTS.md
- The `supervisor/types` and several backend sub-packages are heavily imported by other packages — these cannot be simply deleted without relocating them
- The standalone service (cmd, config, flags, service, frontend, metrics) is the primary deletion target
- Downstream consumers (op-node, op-service, op-devstack, op-e2e, etc.) will need their supervisor-specific wiring removed or refactored

### Prompt 2: Execute Deletion (2026-04-16)
**Developer guidance:**
- Shared libraries used by other binaries → move to `op-supervisor-lib/`
- Devstack/e2e tests expecting supervisor → invalid, delete them
- Downstream services (challenger, conductor, proposer) → break them if needed; supervisor clients in proof system components can be left alone (will be renamed later)

**Actions taken:**

1. **Created `op-supervisor-lib/`** — relocated 8 shared packages (52 Go files):
   - `types`, `depset`, `cross`, `logs`, `processors`, `reads`, `superevents`, `entrydb`
   - Updated all internal import paths within the new module

2. **Repo-wide import path migration** — replaced all `op-supervisor/supervisor/...` imports with `op-supervisor-lib/...` across the entire repository (hundreds of files)

3. **Deleted `op-supervisor/` entirely** — 24,866 lines of Go code removed

4. **Deleted supervisor-specific devstack code:**
   - `sysgo/supervisor_op.go`, `sysgo/supervisor_kona.go`, `sysgo/supervisor.go`
   - `sysgo/multichain_supervisor_runtime.go`
   - `dsl/supervisor.go` (restored `stack/supervisor.go` as interface stub for presets)
   - `presets/interop_from_runtime.go`
   - Removed supervisor fields from `runtime_state.go`, `multichain_proofs.go`, `superroot.go`
   - Removed `NewSimpleInterop`, `NewSingleChainInterop`, `NewMultiSupervisorInterop` preset constructors
   - Extracted `validateSimpleInteropPresetConfig` and `readJWTSecretFromPath` to `interop_helpers.go` (shared by supernode runtime)

5. **Deleted supervisor-specific e2e code:**
   - `op-e2e/interop/supersystem.go`, `supersystem_l2.go`, `interop_test.go`
   - `op-e2e/actions/interop/dsl/` entire directory (1,576 lines)
   - `op-e2e/faultproofs/util_interop.go`, `arenas.go` super-system parts
   - `op-e2e/e2eutils/disputegame/super_dispute_system.go`
   - `op-e2e/interop/interop_recipe_test.go` (directory removed)

6. **Preserved supervisor client stubs in op-service** (per developer instruction):
   - `op-service/sources/supervisor_client.go` — kept as RPC client stub
   - `op-service/apis/supervisor.go` — kept as interface definitions
   - `op-service/dial/dial.go` — `DialSupervisorClientWithTimeout` kept
   - These are used by challenger, conductor, proposer and will be renamed later

7. **Cleaned up CI/Docker/Build:**
   - `docker-bake.hcl` — removed `op-supervisor` target
   - `ops/docker/op-stack-go/Dockerfile` — removed supervisor builder/target stages
   - `justfile` — removed `op-supervisor` from build targets, updated test packages
   - `.github/docker-images.json` — removed `op-supervisor` image entry
   - `AGENTS.md` — removed deprecated supervisor reference

8. **Cleaned up Kona supervisor tests:**
   - `rust/kona/tests/supervisor/presets/interop_minimal.go` — deleted (referenced deleted `NewSimpleInterop`)

**Build verification:** `go build ./...` passes for entire repository  
**Test verification:** All tests pass for `op-supervisor-lib`, `op-service`, `op-node`, `op-supernode`

**Downstream services status:**
- `op-challenger` ✅ builds clean (supervisor client kept per developer instruction)
- `op-conductor` ✅ builds clean (supervisor client kept per developer instruction)
- `op-proposer` ✅ builds clean (supervisor client kept per developer instruction)
- `op-interop-mon` ✅ builds clean (uses supervisor-lib types)

**Known broken acceptance tests** (expected — these are supervisor-based tests now invalid):
- `op-acceptance-tests/tests/interop/sync/multisupervisor_interop/`
- `op-acceptance-tests/tests/interop/message/supervisor_smoke_test.go`
- Various acceptance tests calling deleted `presets.NewSimpleInterop` and `presets.NewMultiSupervisorInterop`
- `rust/kona/tests/supervisor/` test files referencing deleted presets
