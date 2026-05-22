# op-deployer v2 Implementation Plan

This plan captures the current design direction for decoupling `op-deployer`
from contracts releases while preserving the useful parts of the existing
tooling.

The core objective is:

```text
Build one fixed op-deployer binary that can be pointed at a contracts repo/ref
and can deploy, upgrade, inspect, and export using the interfaces that exist at
that ref, without rebuilding op-deployer for each contracts release.
```

The important constraint is that a built Go binary cannot load new compile-time
Go structs from an arbitrary future commit. Any design that requires the fixed
binary to import commit-specific generated Go packages recreates the coupling.

The implementation should therefore converge on one execution path:

```text
fixed shell binary -> stable helper protocol -> per-commit helper
```

Generated Go types can exist for authoring and programmatic consumers, but they
must serialize into the same protocol. They are not a second deployment engine.

## Target Architecture

There are three layers.

### 1. Fixed shell binary

The fixed `op-deployer` binary is contracts-version agnostic. It owns stable
operator and infrastructure concerns:

- Parse global flags and stable execution flags.
- Resolve `--contracts-source <repo-or-path>@<ref>`.
- Fetch/cache the selected contracts source by commit/content hash.
- Build or locate per-ref artifacts.
- Build and spawn the per-commit helper.
- Manage subprocess lifecycle and helper protocol negotiation.
- Own signer integration, RPC plumbing, transaction broadcast framing, logs,
  and local output paths.
- Persist generic state blobs as opaque JSON.
- Present stable CLI help, errors, and output framing.

The shell must not import packages that contain contracts-shaped structs such
as `DeployOPChainInput`, `DeploySuperchainOutput`, versioned OPCM upgrade
inputs, or fixed `intent.toml` / `state.json` schemas.

In shell source, contracts-shaped data should be one of:

- `json.RawMessage`
- protocol-level `map[string]any`
- schema values from the stable schema package

### 2. Per-commit helper

The selected contracts ref owns the deployer code that is a function of that
ref. It lives next to the contracts and artifacts.

It owns:

- Solidity-shaped generated Go structs.
- Generated schema manifest.
- Generated JSON schema for user inputs and state blobs.
- Pipeline stage order and output-to-input wiring.
- Intent/state adapters for compatibility with legacy formats.
- Script/OPCM binding metadata for that ref.
- Stable exports such as addresses, genesis, rollup config, and deployment
  state, translated from the per-ref state model.

The helper is built at runtime from the selected ref, then driven by the fixed
shell over a small versioned protocol. The boring default should be a Go
subprocess over stdin/stdout JSON-RPC. WASM or stronger sandboxing can be
revisited after the protocol is proven.

### 3. Stable protocol and SDK

The shell and helper share only a small protocol. This is the actual stability
boundary.

The protocol should be versioned explicitly and kept narrow. The helper should
import a stable helper SDK package that defines protocol types, helpers for
structured errors, and host operation clients. The shell implements the host.

Initial protocol methods:

| Method | Purpose |
| --- | --- |
| `describe` | Return protocol version, helper capabilities, manifest, schemas, scripts, adapters, and stable export surfaces. |
| `validate-intent` | Validate a user-provided input/intent blob against the helper's schema and adapters. |
| `template` | Produce an input scaffold from the selected ref's schema. |
| `plan` | Convert intent/input into a deterministic deployment or upgrade plan. |
| `apply` | Execute a plan or run the deployment workflow through host operations. |
| `script-run` | Run a selected script surface from the ref. |
| `upgrade` | Execute the native OPCM upgrade workflow for the ref. |
| `export` | Emit stable downstream outputs such as addresses, genesis, rollup config, and deployment state. |

Every request and response should include:

- protocol version
- helper identity
- selected contracts ref
- run ID
- payload as JSON
- structured errors

Contracts-shaped payloads remain JSON at the shell boundary.

## One Route, With Optional Typed Authoring

There should not be separate "typed compile-time deploy" and "untyped runtime
deploy" paths.

The single execution route is:

```text
payload -> shell protocol -> per-commit helper -> host operations -> outputs
```

Programmatic Go consumers can still get compile-time types by importing a
per-ref generated module, but those types only help them author payloads. They
must submit the payload through the same shell/helper protocol.

Example programmatic flow:

```go
import deployertypes "github.com/ethereum-optimism/optimism/op-deployer-types"

input := deployertypes.Intent{
    // typed fields from the pinned contracts ref
}

runner := deployer.NewRunner(deployer.RunnerConfig{
    ContractsRef: "github.com/ethereum-optimism/optimism@abc123",
})

result, err := runner.Apply(ctx, deployertypes.Manifest, input)
```

The runner serializes `input` to JSON and sends it to the helper. It also sends
the compiled-in manifest so the shell can compare the caller's type pin against
the runtime target ref before any deployment work starts.

## Generated Types and Schemas

The source of truth should be the contracts artifacts and per-ref binding
manifest.

Do not hand-write Solidity-shaped Go structs and then reflect over them as the
primary source of truth. That only moves the coupling from the fixed shell into
another manually-maintained package.

Generation should flow this way:

```text
Foundry artifacts + bindings.yaml
  -> generated Go input/output structs
  -> generated schema manifest
  -> generated JSON schema
  -> helper stubs / binding registry
```

The generated Go structs are useful for:

- per-commit helper implementation
- Go consumers that pin the matching types module
- compile-time breakage when a contracts ABI changes

The generated manifest is useful for:

- runtime compatibility checks
- CLI schema-aware validation
- editor and non-Go tooling
- schema diffs in PRs

The generated JSON schema is useful for:

- `intent.toml` / YAML validation with file and line errors
- template generation
- non-Go consumers
- operator help that is aware of the selected contracts ref

## Contracts Source Layout

The exact directory can change, but the contracts ref should contain a clear
deployer-owned area. A target shape:

```text
packages/contracts-bedrock/
  forge-artifacts/
  op-deployer/
    bindings.yaml
    helper/
      main.go
      pipeline/
      adapters/
    types/
      go.mod
      generated/
      manifest.go
      jsonschema/
```

`bindings.yaml` declares the deployer-facing surfaces for that ref. It should
name artifacts, functions, script IDs, generated package names, and optional
adapter metadata.

The current pilot manifest is:

```yaml
artifact_base: packages/contracts-bedrock/forge-artifacts
output_base: op-deployer/pkg/deployer/opcm/generated

script_bindings:
  - name: deploysuperchain
    artifact: DeploySuperchain.s.sol/DeploySuperchain.json
    package: deploysuperchain
    out: deploysuperchain
    filename: deploy_superchain.gen.go
```

That pilot lives under `op-deployer` for the v1 burn-down. In the final shape,
the manifest and generated helper/types should move to the contracts source.

## Runtime Flow

For a CLI command:

```bash
op-deployer --contracts-source github.com/ethereum-optimism/optimism@abc123 apply \
  --l1-rpc-url <url> \
  --input intent.yaml \
  --state-out state.json
```

The shell should:

1. Resolve `abc123` to an immutable commit.
2. Fetch or reuse a cached checkout.
3. Build or locate artifacts for that checkout.
4. Build the helper from that checkout.
5. Start the helper.
6. Call `describe`.
7. Validate protocol compatibility.
8. Validate the user's input using the helper-provided schema.
9. Call `plan` or `apply`.
10. Execute host operations requested by the helper.
11. Persist the helper-produced state blob.
12. Call `export` for stable downstream outputs when requested.

The shell must treat the contracts-shaped section of the intent and state as
opaque. It can validate and present schema errors, but it should not hardcode
field names like `systemConfig`, `disputeGameConfigs`, or
`superchainProxyAdminOwner`.

## Stable Outputs

The shell should not implement a contracts-shaped state normalizer. That would
become the next hardcoded coupling point.

Instead, the per-commit helper should expose stable exports:

```json
{
  "addresses": {},
  "genesis": {},
  "rollupConfig": {},
  "deploymentState": {}
}
```

The helper translates its per-ref state into these stable shapes. The shell
persists, frames, and routes them to files or stdout.

This keeps downstream consumers such as `op-node`, registry tooling, and
operators insulated from contracts release internals without moving release
knowledge into the fixed binary.

## Runtime Compatibility Check

The per-ref generated types module should ship a manifest:

```go
var Manifest = schema.Manifest{
    SchemaHash: "<hash of canonicalized field definitions>",
    Structs: map[string]schema.StructDef{
        "DeployOPChainInput": {
            Fields: []schema.Field{
                {Name: "OpChainProxyAdminOwner", Type: "address", Required: true},
                {Name: "OperatorFeeScalar", Type: "uint32", Required: true},
            },
        },
    },
}
```

At runtime, programmatic consumers can pass their compiled manifest to the
shell. The helper returns its own manifest from `describe`.

The shell classifies the comparison:

| Result | Meaning | Action |
| --- | --- | --- |
| Identical | Schema hashes match. | Run. |
| Compatible skew | Required fields and types match; only optional fields differ. | Run, optionally warn. |
| Incompatible | Required field missing, type changed, or struct removed. | Refuse before deployment. |

Example error:

```text
target ref:  abc123 (manifest hash 5a2...)
your types:  def456 (manifest hash 9bf...)
status:      INCOMPATIBLE

DeployOPChainInput:
  + EthLockboxOwner (address, required)   target requires, caller types missing
  ~ OperatorFeeConstant: uint64 -> uint128

Fix: bump op-deployer-types to a ref >= abc123, or change --contracts-source.
```

This check covers ABI shape compatibility. It cannot prove semantic
compatibility if a field keeps the same name and type but changes meaning.

## CLI Surface

Stable CLI flags are shell concerns:

- `--contracts-source`
- `--l1-rpc-url`
- signer flags
- broadcast/simulation mode
- input/output paths
- cache paths
- logging

OPCM/script fields are not stable first-class flags. They come from the
selected contracts source and are supplied through files or schema-aware
overrides.

Native commands should be source-backed:

```text
source info
source list
template
apply
upgrade
bootstrap
script run
inspect/export
verify
clean
gen
```

Legacy-compatible commands can remain, but they should be implemented as
adapters from legacy CLI/file formats into the helper protocol. The adapter
code should live with the selected contracts ref, not in the fixed shell.

## Current Pilot State

The current repo already has the first generator spike:

- `op-deployer/pkg/deployer/gen`
- `op-deployer gen bindings --config <manifest> --base-dir <repo>`
- `op-deployer gen script-binding` as a lower-level primitive
- `op-deployer/pkg/deployer/opcm/bindings.yaml`
- `op-deployer/pkg/deployer/opcm/generate.go`
- generated `DeploySuperchain` types under
  `op-deployer/pkg/deployer/opcm/generated/deploysuperchain`
- experimental `op-deployer-types` package generated from its own
  `bindings.yaml`
- generated typed `DeploySuperchain` package under
  `op-deployer-types/generated/deploysuperchain`
- generated `op-deployer-types.Manifest` schema value with a deterministic
  schema hash

The generated `DeploySuperchain` fields come from the current artifact ABI,
not the handwritten Go struct. This already surfaced a useful issue: the
artifact contains protocol-version fields that differ from the currently
inspected Solidity source. That is exactly the kind of drift a generation
freshness check should expose.

This pilot is a v1 burn-down step, not the final v2 architecture. It proves
that script types can be generated from artifacts and checked into source. The
final runtime should move generated commit-specific packages out of the fixed
binary path.

## Implementation Phases

### Phase 0: Harden the v1 generator pilot

Goal: prove generated script bindings can replace handwritten script structs in
current `op-deployer` without changing public behavior.

Tasks:

- Keep the manifest-driven generator as the normal workflow:

  ```bash
  go generate ./op-deployer/pkg/deployer/opcm
  ```

- Add `op-deployer gen bindings --check` so CI can validate freshness without
  relying only on `git diff` after generation.
- Complete ABI type support:
  - addresses
  - bool/string
  - bytes and bytesN
  - uint/int sizes
  - fixed and dynamic arrays
  - nested tuples
  - tuple arrays
- Replace `opcm/superchain.go` handwritten input/output structs with aliases
  to generated `deploysuperchain.Input` and `deploysuperchain.Output`.
- Update script constructors to use generated metadata constants.
- Add CI:

  ```bash
  go generate ./op-deployer/pkg/deployer/opcm
  git diff --exit-code op-deployer/pkg/deployer/opcm/generated
  ```

Acceptance criteria:

- `bootstrap superchain` behavior is unchanged.
- `apply` superchain stage behavior is unchanged.
- `opcm/superchain.go` no longer owns handwritten `DeploySuperchainInput` and
  `DeploySuperchainOutput`.
- CI fails when the artifact ABI changes and generated Go is stale.

Next v1 targets:

1. `DeployImplementations`
2. `ReadSuperchainDeployment`
3. `DeployOPChain`
4. `ReadImplementationAddresses`
5. `L2Genesis`
6. upgrade and migrate bindings

### Phase 1: Extract the stable schema and protocol packages

Goal: define the data model that lets shell and helper communicate without
shared contracts-shaped Go structs.

Tasks:

- Add a stable schema package with:
  - `Manifest`
  - `StructDef`
  - `Field`
  - canonicalization and hashing
  - compatibility diffing
  - ABI type naming
  - JSON schema emission
- Add a stable protocol package with:
  - request/response envelopes
  - protocol version negotiation
  - structured errors
  - helper capability model
  - host operation request/response types
- Add golden tests for manifest hashing and compatibility diffs.
- Add fixtures for at least two schema versions to test compatible and
  incompatible skew.

Acceptance criteria:

- Manifest hashes are deterministic.
- Compatibility errors are structured and useful.
- The shell can compare caller manifest vs helper manifest without importing
  helper code.

### Phase 2: Add contracts-source resolution and caching

Goal: let the fixed binary resolve a contracts source independently of its own
build.

Tasks:

- Define `--contracts-source` grammar:
  - local path
  - local path plus ref
  - GitHub repo plus commit/tag
  - possibly artifact bundle URI later
- Resolve refs to immutable commits.
- Cache checkouts and build products by commit/content hash.
- Build or locate Foundry artifacts for a selected source.
- Record source metadata:
  - repo
  - ref
  - resolved commit
  - artifact hash
  - helper binary hash
- Add cache cleanup and diagnostics.

Acceptance criteria:

- `op-deployer source info --contracts-source <source>` prints resolved commit
  and artifact/helper status.
- Repeated runs reuse cache.
- The binary does not need embedded artifacts for source-backed commands.

### Phase 3: Build the per-commit helper MVP

Goal: prove the shell can build a helper from the selected ref and talk to it
over the stable protocol.

Tasks:

- Add helper scaffold in the contracts source.
- Generate a helper registry from `bindings.yaml`.
- Implement `describe`.
- Implement `template` from JSON schema.
- Implement `validate-intent` for a simple script input.
- Implement `script-run` for `DeploySuperchain` or a read-only script.
- Add shell subprocess management:
  - build helper
  - start helper
  - send JSON requests
  - stream logs
  - handle structured errors
- Add protocol version negotiation.

Acceptance criteria:

- Fixed shell can run `describe` against a selected contracts ref.
- Fixed shell can validate an input using helper-provided schema.
- Fixed shell can invoke a simple helper method without importing generated
  types.

### Phase 4: Move deployment meaning into the helper

Goal: move the release-specific pipeline out of the fixed binary.

Tasks:

- Implement helper-side pipeline definition:
  - stage names
  - stage order
  - input sources
  - output-to-input wiring
  - state writes
- Add static lint for the pipeline:
  - every stage input field has a source
  - every referenced output field exists
  - required state fields are written before consumed
- Implement legacy `intent.toml` adapter for the selected ref.
- Implement helper-owned state schema for the selected ref.
- Implement stable `export` for:
  - addresses
  - genesis
  - rollup config
  - deployment state
- Keep state internals opaque to the shell.

Acceptance criteria:

- `apply` can run through the helper for one simple deployment path.
- Shell persists opaque state and stable exports.
- No fixed-shell code maps `intent.toml` fields to script input fields.

### Phase 5: Implement native upgrade through the helper

Goal: remove hardcoded versioned upgrade packages from the fixed binary.

Tasks:

- Generate OPCM `upgrade` input schema from the selected OPCM ABI.
- Hardcode only the user workflow name `upgrade`, not the input fields.
- Accept:
  - `--input <yaml/json>`
  - schema-aware `--input.<path>=<value>` overrides later
- Encode `upgrade` call data from ABI-derived schema.
- Support delegatecall execution and simulation.
- Support nested dynamic values such as dispute game configs and
  extra instructions from schema rather than handwritten encoders.
- Produce plan JSON before broadcast.

Acceptance criteria:

- New upgrade path can run against a selected ref without adding a new
  versioned Go package.
- Adding a required OPCM input in contracts forces helper/types generation and
  schema updates in that contracts ref, not a fixed-shell release.

### Phase 6: Add generated types module for Go consumers

Goal: preserve compile-time typed authoring for Go consumers without coupling
the fixed binary.

Tasks:

- Add `op-deployer-types` module under the contracts source.
- Generate:
  - intent structs
  - OPCM input/output structs
  - script input/output structs
  - manifest
  - JSON schema
  - typed helper methods that serialize into protocol payloads
- Publish or make it importable as:

  ```go
  require github.com/ethereum-optimism/optimism/op-deployer-types v0.0.0-<commit>
  ```

- Add a consumer-side pin convention:

  ```go
  const ContractsRef = "abc123"
  ```

- Add a lint that checks the `ContractsRef` and module pseudo-version agree.
- Add runtime manifest comparison between consumer manifest and helper
  manifest.

Acceptance criteria:

- Go consumers get compile-time errors when they use fields not present in
  their pinned types.
- Runtime fails early if the caller's types pin and `--contracts-source` ref
  are incompatible.
- Programmatic deploys still use the same shell/helper protocol.

### Phase 7: Migrate in-tree consumers

Goal: remove internal dependencies on fixed `op-deployer` contracts-shaped
packages.

Consumer groups:

| Group | Current shape | Migration path |
| --- | --- | --- |
| Shared infra users such as `op-fetcher` | Reuse `broadcaster`, `env`, script host internals. | Move reusable infra to neutral packages or expose stable shell host APIs. |
| Read-only canonical lookup such as `op-validator` | Import `standard` constants. | Resolve from selected contracts source or move stable constants to neutral package. |
| Intent builders such as `op-e2e/e2eutils/intentbuilder` | Build `state.Intent` directly. | Import generated `op-deployer-types` pinned to the test ref. |
| In-process deploy such as `op-devstack/sysgo` and `interopgen` | Call `ApplyPipeline` or typed `opcm` bindings. | Use typed module for payloads and execute through shell/helper runner, or invoke shell as subprocess. |
| Superchain registry tooling | Parse current `state.json` and inspect helpers. | Consume helper `export` outputs or selected-ref state adapters. |

Acceptance criteria:

- No in-tree consumer imports fixed-shell packages that contain
  contracts-shaped generated types.
- Consumers either use the generated types module or the stable protocol/schema
  API.

### Phase 8: Retire release-coupled v1 surfaces

Goal: remove the hardcoded release catalog from the fixed binary after parity
exists.

Targets:

- versioned `upgrade/vX_Y_Z` packages
- embedded artifact defaults for source-backed commands
- `standard.CurrentTag` as an execution default
- fixed legacy `state.Intent` and `state.State` in the shell path
- fixed pipeline stage structs in the shell path
- fixed `opcm.Scripts` script catalog in the shell path

Acceptance criteria:

- Adding a new contracts release does not require modifying the fixed shell
  binary.
- New release support is added by updating the contracts ref's generated
  bindings, helper, adapters, and CI fixtures.

## CI Plan

### Contracts repo CI

This is the load-bearing CI because the per-commit helper lives with the
contracts.

Checks:

1. Generate:

   ```bash
   go generate ./packages/contracts-bedrock/op-deployer/...
   git diff --exit-code
   ```

2. Build helper and generated types module.
3. Static pipeline lint:
   - all required inputs sourced
   - consumed outputs exist
   - state fields read after write
4. Surface diff guard:
   - if deployer-facing ABI changes, generated code/schema must change
   - PR output includes a human-readable schema diff
5. Smoke deploy against ephemeral L1 with a pinned fixed shell binary.
6. Protocol fixture tests against the minimum supported shell protocol version.

### Fixed shell CI

Checks:

- Unit tests for source resolver/cache.
- Protocol compatibility fixtures.
- Manifest compatibility diff tests.
- No imports of per-commit generated packages.
- End-to-end tests against checked-in helper fixtures from multiple refs.

### Consumer CI

Checks:

- Build against pinned generated types module.
- Lint that `ContractsRef` matches the types module commit.
- Optional smoke run against the fixed shell and the same ref.

## Trust and Security

Running code from an arbitrary contracts ref is powerful. The production trust
model must be explicit.

MVP constraints:

- Default to local paths and allowlisted GitHub repos.
- Resolve tags to commits and record immutable commit hashes.
- Cache by commit and helper binary hash.
- Print helper source and hash in logs.
- Require explicit opt-in for dirty local worktrees.

Future hardening:

- signed refs or signed helper manifests
- reproducible helper builds
- allowlist policy for production deploys
- sandboxed helper runtime

## Error Handling

The shell should surface immediate, structured errors for:

- source resolution failure
- artifact build failure
- helper build failure
- protocol version mismatch
- schema/manifest incompatibility
- input validation errors with file/line context
- helper static validation errors
- host operation failures such as RPC/broadcast errors

The error boundary should be clear:

| Failure mode | Caught where |
| --- | --- |
| Solidity ABI changed but generated code stale | contracts repo CI |
| pipeline references missing output | contracts repo static lint |
| Go consumer uses missing field | consumer compile time |
| consumer types pin differs from runtime ref | shell startup manifest check |
| user typo in input file | shell startup using helper schema |
| semantic field meaning changed without type/name change | smoke deploy or runtime |

## Open Decisions

1. Helper host model:
   - Should the helper request transactions from the shell as host operations,
     or receive a constrained RPC/broadcast client from the shell SDK?
   - Preferred direction: helper owns deployment meaning, shell owns signer and
     broadcast authority.

2. In-process deploy support:
   - Keep it through a Go runner that uses the same helper protocol.
   - Avoid maintaining a second deployment engine.

3. Legacy format ownership:
   - Preferred direction: legacy `intent.toml` and `state.json` adapters live
     with the selected contracts ref.

4. Stable export shape:
   - Need to define exactly which exports are stable enough for downstream
     consumers: addresses, genesis, rollup config, deployment metadata.

5. Source layout:
   - Decide the exact contracts repo directory for helper, generated types,
     schemas, and adapters.

## Near-Term Next Steps

The immediate next implementation sequence should be:

1. Add `gen bindings --check`.
2. Replace `DeploySuperchainInput` and `DeploySuperchainOutput` with generated
   aliases in v1.
3. Add the generated freshness check to CI.
4. Extract a small `schema` package and generate a manifest for
   `DeploySuperchain`.
5. Add manifest hash and compatibility tests.
6. Create a minimal helper scaffold with `describe`.
7. Teach the shell to build/spawn that helper from a local contracts checkout.
8. Run `describe` and validate a `DeploySuperchain` input through the helper
   without importing generated types into the shell.

That sequence keeps momentum on the concrete v1 burn-down while building
toward the final commit-independent architecture.
