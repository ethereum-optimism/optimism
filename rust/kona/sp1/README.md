# `sp1`

This directory contains an integration of [SP1](https://github.com/succinctlabs/sp1) zero-knowledge proof capabilities into Kona, enabling validity proofs for OP Stack state transitions. This integration is derived from the [OP-Succinct](https://github.com/succinctlabs/op-succinct) project.

> **⚠️ Experimental**: The SP1 fault proof integration is currently experimental and under active development. It is not yet recommended for production use.

## Overview

The SP1 integration provides zkVM-based fault proofs for the OP Stack, allowing verifiable state transitions to be proven on-chain. This enables trustless bridging and enhanced security for rollup chains.

## Structure

### Programs (`programs/`)

zkVM programs that execute inside the SP1 prover:

- **`super-range`**: Unified super-root program for one or more chains, with modes for
  proving ranges and span-shaped consolidation.
- **`super-aggregation`**: Recursively verifies unified super-range proofs and
  commits the public values consumed by `ZKDisputeGame`.

### Crates (`crates/`)

Supporting libraries for the SP1 fault proof system:

- **`client`**: Client-side utilities and types for witness execution in the zkVM
- **`elfs`**: Runtime loading of compiled ELF binaries
- **`ethereum/client`**: Ethereum-specific client-side data availability utilities
- **`host`**: Host-side logging, metrics, prover-network configuration, and witness-generation utilities
- **`range-vkeys`**: Compile-time `super-range` guest verification key, embedded from generated
  `elf/vkeys.toml` and used by `super-aggregation`. The crate retains its historical name because
  it authenticates the shipping super-range child program.
- **`proposer`**: The `kona-sp1-proposer` service: creates super-root ZK dispute games,
  defends challenged ones with SP1 super-aggregation proofs, resolves finished games, and
  claims bonds (see the Proposer section below)
- **`super-range-executor`**: Witness synthesis and execution engine for the super-root
  programs; used as a library by the proposer and as the `kona-sp1-super-range-executor`
  validity-checker binary in acceptance tests
- **`zkvm-canary`**: The `kona-zkvm-canary` service, which continuously executes both
  `super-range` modes against finalized live-network snapshots without producing proofs

### ELF Binaries (`elf/`)

Compiled ELF binaries for the zkVM programs, used by the prover:

- **`super-aggregation-elf`**: Compiled super-root aggregation program
- **`super-range-elf`**: Compiled unified super-root range/consolidation program

In the optimism monorepo port, these files and `elf/vkeys.toml` are generated on demand and
ignored by git, matching the Cannon prestate artifact workflow. Generate reproducible ELFs
on linux/amd64 with `just build-elfs`; it builds the `super-range` leaf first, generates its vkey,
and then builds `super-aggregation` with that vkey embedded through `kona-sp1-range-vkeys`. Use
`just build-elfs-native` for local iteration and the fast per-PR compile check; CI persists the
native manifest with the generated ELFs. Native ELF hashes may differ across build environments
because paths and other environment details are embedded. A Docker-based, uncached tag/release
reproducibility CI check is intentionally left to a future follow-up.

Host-toolchain workspace builds need neither ELFs nor `vkeys.toml`. Host binaries load guest
artifacts at runtime from `KONA_SP1_ELF_DIR`; a missing or empty artifact fails as an
infrastructure error. Release automation will eventually pin per-version vkeys from the generated
manifest into `superchain-registry/validation/standard/standard-prestates.toml` and verify
reproducible builds.

Custom chains and devnets can compile separate SP1 artifacts with custom kona
registry inputs:

```bash
KONA_CUSTOM_CONFIGS_DIR=/path/to/custom/configs just build-elfs
```

The directory must contain `chainList.json`, `configs.json`, and `depsets.json`.
Those files are compiled into the kona crates used by the guest programs, so
custom configs produce different ELFs and verification-key hashes.

## Acceptance coverage

Kona-SP1 acceptance coverage targets the shipping super-root path: `super-range` and
`super-aggregation`, for both multi-chain interop and a single-chain dependency set of size one.

| Tier | Coverage | Boundary |
|---|---|---|
| Required native core | Existing two-chain scenarios and one canonical single-chain scenario collect real witnesses and replay the shared range/consolidation cores. | Does not load the zkVM ELF. |
| Scheduled full ELF | One valid cross-chain transition and one canonical single-chain transition execute `super-range-elf`. | Exercises ELF input/output and zkVM patches, but does not generate a proof. |
| Required lifecycle | The proposer and challenger assemble real child outputs and aggregation inputs, then submit mock proof bytes through `SP1PlonkAdapter` backed by `MockSP1Verifier`. | Exercises the onchain lifecycle without production proof verification. |
| Guest and artifact | Guest tests validate single-chain aggregation, child ordering, and `SUPER_RANGE_VKEY`; scheduled checks bind the built aggregation vkey to the deployed prestate. | Does not generate a recursive aggregation proof. |

Run the native and full-ELF acceptance packages from the repository root after building the
standard acceptance dependencies:

```bash
RUST_JIT_BUILD=1 go test -count=1 -timeout=60m \
  ./op-acceptance-tests/tests/interop/proofs/serial \
  ./op-acceptance-tests/tests/interop/proofs-singlechain

cd rust/kona/sp1 && just build-elfs-native && just build-super-range-executor && cd ../../..
KONA_SP1_ELF_DIR="$PWD/rust/kona/sp1/elf" \
KONA_SP1_SUPER_RANGE_ELF_EXECUTOR_PATH="$PWD/rust/target/release/kona-sp1-super-range-executor" \
RUST_JIT_BUILD=1 go test -count=1 -parallel=1 -timeout=120m \
  ./op-acceptance-tests/tests/interop/proofs/sp1
```

Real recursive proof generation, SPN credentials, and production verifier proof-byte validation
remain network-proving concerns rather than deterministic acceptance-test coverage.

## Guest Precompile Patches

Both guest programs are isolated in `programs/Cargo.toml`, a nested Cargo workspace with its
own `Cargo.lock` and `[patch.crates-io]` table. That workspace patches `sha2`, `sha3`,
`crypto-bigint`, `k256`, `p256`, and `substrate-bn` to the SP1 forks, so the
generated ELFs get zkVM precompile-accelerated crypto without changing the host
`rust/` workspace dependency graph.

The EVM-executing super-range guest also enables `revm`'s `bn` feature
in the nested workspace. That forwards to `revm-precompile`'s `substrate-bn`
backend for EIP-196/197 bn128 precompiles. EIP-2537 BLS pairing still uses
arkworks and is not SP1 accelerated.

## Usage

The shipping SP1 integration proves super roots over a non-empty dependency set:

1. **Super-range proof generation**: `super-range` executes one or more chains' range and
   consolidation transitions in the zkVM.
2. **Super-aggregation**: `super-aggregation` recursively verifies the child proofs and commits the
   public values expected by `ZKDisputeGame`.
3. **Onchain verification**: The aggregated proof is submitted to the dispute game on L1.

## Proposer (`kona-sp1-proposer`)

The proposer service (ported from op-succinct's fault-proof proposer) plays the
super-root `ZKDisputeGame` (game type 10) end to end:

1. **Create**: proposes the supernode's canonical super root on a fixed interval
   (`proposer.rs` creation scheduling).
2. **Defend**: when a game in the owned set is challenged, fetches the span's super
   roots, collects witnesses through the kona `InteropHost`
   (`super-range-executor` library), obtains SP1 proofs (`prover.rs` providers,
   `proving.rs` pipeline), and submits `prove()`. In fast finality mode the same
   pipeline also proves owned games while they are still unchallenged.
3. **Resolve and claim**: resolves finished games and claims bonds (including the
   challenger bond earned by proving).

### Ownership (which games it defends)

Defense, resolution, and bond claims use prestate-based ownership. The proposer
handles every game whose `absolutePrestate()` artifacts it can load, regardless
of creator. Games whose prestate is unknown are skipped with the
`kona_sp1_proposer_unknown_prestate_challenged` gauge as the alarm.

Fast finality uses a narrower spend policy. It proves only unchallenged games
created by the configured proposer signer and owned by prestate. After a signer
rotation, old-signer games stay eligible for defense, resolution, and claims,
but the restart scan does not fast-finalize them.

**Operational requirement**: rotated-out prestate artifacts must remain published
under `KONA_SP1_PROPOSER_PRESTATES_URL` for as long as games created under them can be live, or the
proposer loses the ability to defend, resolve, and claim those games.

### Proof providers

- `KONA_SP1_PROPOSER_PROOF_PROVIDER=network`: real SP1 proving via the Succinct Prover Network.
  Proving keys are set up per prestate on first use, and the aggregation
  verifying key must hash to the on-chain prestate (mismatches poison the
  prestate and remove its games from the owned set). The registered prestate's
  keys are verified BEFORE any game is created on it, so the proposer never
  bonds a game it has not proven it can defend.
- `KONA_SP1_PROPOSER_PROOF_PROVIDER=mock`: dev-only. Runs the full pipeline natively (witness
  collection computes the real range/consolidation outputs and the aggregation
  inputs are validated), then submits placeholder proof bytes. Only a deployment
  with a mock game verifier (devstack) accepts them. No ELFs, no SPN credentials.

### Restart behavior

Proving progress is process-local. Retries with unchanged inputs reuse submitted
request IDs, completed chunks, and fulfilled aggregation proofs. `Cancelled`,
`Expired`, `Reverted`, and `Unfulfillable` requests are retryable and may
purchase replacements. Progress is cleared when a game becomes terminal, is
evicted, fails a definitive pre-submit check, or is proven successfully.

A restart loses all progress and re-detects games that still need proofs. A
pre-flight check prevents duplicate `prove()` submissions. Fast finality also
re-detects unproven, signer-created games after restart.

### Operator alarms

`kona_sp1_proposer_game_proving_error` counts failed proving tasks. A sustained
rate needs investigation because identity changes and retryable terminal
outcomes can purchase replacement proofs.
`kona_sp1_proposer_proving_timeout_error` means a polling attempt exceeded its
client-side wait; the submitted request ID remains available to the next retry.
`kona_sp1_proposer_game_unprovable` counts games given up as permanently
unprovable. A proving task that never completes holds its capacity slot and its
game's dedup slot, so watch `kona_sp1_proposer_proving_duration_seconds` and the
per-tick task-stats log.

### Environment

All proposer-owned variables use the `KONA_SP1_PROPOSER_` prefix.

Required core configuration:

| Variable | Purpose |
|---|---|
| `KONA_SP1_PROPOSER_L1_RPC` | L1 execution RPC |
| `KONA_SP1_PROPOSER_SUPERROOT_RPCS` | op-supernode or single-chain op-node RPCs serving `superroot_atTimestamp`. Multiple comma-separated RPCs can be provided for redundancy |
| `KONA_SP1_PROPOSER_FACTORY_ADDRESS` | `DisputeGameFactory` address |
| `KONA_SP1_PROPOSER_PRESTATES_URL` | prestate artifact directory (`<vkey>.agg.bin.gz` + `<vkey>.range.bin.gz`) |
| `KONA_SP1_PROPOSER_PROOF_PROVIDER` | `network` or `mock`; no default |
| `KONA_SP1_PROPOSER_L1_BEACON_RPC` | L1 beacon API (blob sidecars for derivation witnesses) |
| `KONA_SP1_PROPOSER_L2_RPCS` | comma-separated L2 EL RPCs, one per chain (order-irrelevant) |

Optional core and operational configuration:

| Variable | Purpose |
|---|---|
| `KONA_SP1_PROPOSER_ROLLUP_CONFIG_PATHS` | comma-separated rollup config files; absent = registry fallback |
| `KONA_SP1_PROPOSER_L1_CONFIG_PATH` | L1 chain config file; absent = registry fallback |
| `KONA_SP1_PROPOSER_DEPENDENCY_SET_PATH` | dependency-set config file; absent = registry fallback |
| `KONA_SP1_PROPOSER_PROPOSAL_INTERVAL_SECONDS` | proposal interval (default `3600`) |
| `KONA_SP1_PROPOSER_PROPOSAL_SAFETY` | `safe` or `finalized` (default `finalized`) |
| `KONA_SP1_PROPOSER_FETCH_INTERVAL` | loop interval in seconds (default `30`) |
| `KONA_SP1_PROPOSER_METRICS_PORT` | `0` disables metrics; `auto` selects a free port (default `0`) |
| `KONA_SP1_PROPOSER_SYNC_L1_CONFIRMATIONS` | L1 confirmation lag for pinned reads (default `0`) |
| `KONA_SP1_PROPOSER_TX_CONFIRMATION_TIMEOUT` | transaction confirmation timeout in seconds (default `180`) |
| `KONA_SP1_PROPOSER_MAX_FEE_PER_GAS` | L1 max-fee cap in wei (default uncapped) |
| `KONA_SP1_PROPOSER_MAX_PRIORITY_FEE_PER_GAS` | L1 priority-fee cap in wei (default uncapped) |
| `KONA_SP1_PROPOSER_RANGE_SPLIT_COUNT` | chunks per defended span (default `16`, maximum `128`) |
| `KONA_SP1_PROPOSER_MAX_CONCURRENT_RANGE_PROOFS` | child-proof concurrency per game (default `1`) |
| `KONA_SP1_PROPOSER_MAX_CONCURRENT_DEFENSE_TASKS` | concurrent defended games (default `8`, minimum `1`) |
| `KONA_SP1_PROPOSER_FAST_FINALITY_MODE` | prove signer-created owned games while unchallenged (default `false`) |
| `KONA_SP1_PROPOSER_FAST_FINALITY_PROVING_LIMIT` | total in-flight proving tasks before creation pauses (default `1`) |

SP1 network configuration applies when `KONA_SP1_PROPOSER_PROOF_PROVIDER=network`:

| Variable | Purpose |
|---|---|
| `KONA_SP1_PROPOSER_NETWORK_PRIVATE_KEY` | SPN requester private key, or AWS KMS key ARN when KMS is enabled |
| `KONA_SP1_PROPOSER_NETWORK_RPC_URL` | SPN RPC override; absent or empty uses the SP1 SDK default for the selected network mode |
| `KONA_SP1_PROPOSER_USE_KMS_REQUESTER` | use AWS KMS for request signing (default `false`) |
| `KONA_SP1_PROPOSER_RANGE_PROOF_STRATEGY` | range fulfillment strategy (default `auction`) |
| `KONA_SP1_PROPOSER_AGG_PROOF_STRATEGY` | aggregation fulfillment strategy (default `auction`) |
| `KONA_SP1_PROPOSER_SP1_TIMEOUT_SECONDS` | per-proof request deadline and client wait (default `7200`) |
| `KONA_SP1_PROPOSER_NETWORK_CALLS_TIMEOUT` | individual network-call timeout (default `15`) |
| `KONA_SP1_PROPOSER_AUCTION_TIMEOUT` | unassigned mainnet request timeout (default `300`) |
| `KONA_SP1_PROPOSER_RANGE_CYCLE_LIMIT` | range request cycle limit (default `1e12`) |
| `KONA_SP1_PROPOSER_RANGE_GAS_LIMIT` | range request gas limit (default `200000000000`) |
| `KONA_SP1_PROPOSER_AGG_CYCLE_LIMIT` | aggregation request cycle limit (default `1e12`) |
| `KONA_SP1_PROPOSER_AGG_GAS_LIMIT` | aggregation request gas limit (default `1000000000`) |
| `KONA_SP1_PROPOSER_MAX_PRICE_PER_PGU` | optional maximum price per proving gas unit; unset, empty, or `0` uses mainnet auction pricing |
| `KONA_SP1_PROPOSER_MIN_AUCTION_PERIOD` | minimum auction period in seconds (default `30`) |

For mainnet auctions, the SP1 SDK derives the request ceiling from the network's
published price with its 120% buffer and auction-tick rounding. A positive
`MAX_PRICE_PER_PGU` replaces that dynamic ceiling exactly. It remains a ceiling,
not the price paid: the auction settles at the winning bid. A ceiling below the
clearing price leaves a request unbid until its deadline. Expiry makes the
request retryable, so the next game attempt may submit a new auction.
`MIN_AUCTION_PERIOD` is a floor every request waits out, so it needs to cover bid
arrival (3-10s) and no more. It must leave assignment margin under
`AUCTION_TIMEOUT`; cancellation retries the unfinished request while completed
chunks remain cached when the proving inputs are unchanged.

A defended span is split into up to the configured number of chunks. Each sufficiently
large chunk can require a range proof and a consolidation proof before the final
PLONK aggregation proof. `SP1_TIMEOUT_SECONDS` applies independently to each
proof request, not to the complete defense.

At the default one-hour interval, an OP Mainnet span contains about 1,800
two-second blocks. The default 16 chunks average 112.5 blocks each; configuring
18 targets 100 blocks per chunk. Higher counts reduce per-request work but
increase witness collection, fixed proving overhead, SPN request count, and
aggregation input size. `RANGE_GAS_LIMIT` limits each range request, not the
total work of the defense.

Transaction signing requires one of these configurations:

| Variable | Purpose |
|---|---|
| `KONA_SP1_PROPOSER_PRIVATE_KEY` | local L1 transaction-signing key |
| `KONA_SP1_PROPOSER_SIGNER_URL` | Web3Signer URL; requires `KONA_SP1_PROPOSER_SIGNER_ADDRESS` |
| `KONA_SP1_PROPOSER_SIGNER_ADDRESS` | Web3Signer address; requires `KONA_SP1_PROPOSER_SIGNER_URL` |

Logging and telemetry:

| Variable | Purpose |
|---|---|
| `KONA_SP1_PROPOSER_LOGGER_NAME` | OpenTelemetry service name (default `kona-sp1`) |
| `KONA_SP1_PROPOSER_OTLP_ENDPOINT` | OpenTelemetry endpoint (default `http://localhost:4317`) |
| `KONA_SP1_PROPOSER_OTLP_ENABLED` | enable OpenTelemetry export (default `false`) |
| `KONA_SP1_PROPOSER_LOG_FORMAT` | `pretty` or `json` (default `pretty`) |

The proposer and its dependencies also observe the standard `RUST_LOG`, `NO_COLOR`,
`SSL_CERT_DIR`, `SSL_CERT_FILE`, `OTEL_*`, proxy, AWS credential, and SP1 worker/debug
variables. `KONA_SP1_ELF_DIR` configures shared build/test infrastructure.

### Fast finality

With `KONA_SP1_PROPOSER_FAST_FINALITY_MODE=true` the proposer proves every signer-created owned
game while it is still unchallenged, spawned by the per-tick scan one fetch
interval after creation. A proven game is over immediately, so it resolves as
soon as its parent does instead of waiting out `maxChallengeDuration`: proof
spend traded for finality latency. Blacklisted and retired games are skipped.
Off by default.

Spend framing: in network mode this proves every game created by this signer
whose prestate artifacts are available. At a one-hour proposal interval that is
a baseline of 24 aggregation proofs per day; the default
`KONA_SP1_PROPOSER_FAST_FINALITY_PROVING_LIMIT=1` serializes them. An unchallenged game created by
another proposer is not proven, even when it uses a known prestate. Enabling the
mode on a chain with an existing unchallenged backlog proves the signer-created
owned backlog.

Concurrency interaction:

| Situation | Effect |
|---|---|
| active proving tasks (defense + fast finality) >= limit | no new fast-finality proving; game creation paused this tick |
| defense tasks alone >= limit | same: defense load pauses creation (upstream parity) |
| fast-finality tasks in flight | never count against `KONA_SP1_PROPOSER_MAX_CONCURRENT_DEFENSE_TASKS` |
| game challenged while a fast-finality proof is in flight | the proof stays valid; per-game dedup prevents a second task |
| a fast-finality proof keeps failing at the limit | creation stays paused until it succeeds or is classified unprovable; watch `kona_sp1_proposer_game_proving_error` |

## Live execution canary (`kona-zkvm-canary`)

`kona-zkvm-canary` checks that the published `super-range` guest agrees with a finalized,
L1-pinned live-network view. One process owns one network and one authenticated artifact release.
It selects a consecutive finalized span, runs range mode and consolidation mode sequentially,
and classifies their results. The L1 pin is the earlier of the supernode's fully processed block
(`CurrentL1 - 1`) and L1's `finalized` block; a snapshot whose `required_l1` is above that chosen
pin is an input error. Finalized supernode responses are assumed immutable for a timestamp. An
identical successful fingerprint is not re-executed; a correctness failure receives one
confirmation attempt. The next selection is scheduled only after the prior attempt and its
cadence plus bounded jitter have completed.

The canary always uses SP1 CPU `execute` mode. This runs the real RISC-V ELF and returns an
`ExecutionReport`, but it never invokes a proving API, submits work to the Succinct Prover Network,
or creates proof bytes. This differs from the proposer's `mock` provider, which does not execute an
ELF and submits placeholder proof bytes to a development verifier. It also differs from the
one-shot executor's `--native-core` mode, which replays witnesses without the SP1 emulator. The
canary exposes neither a mock mode nor a native-core switch.

### Artifact identity and startup

`KONA_ZKVM_CANARY_PRESTATES_URL` and `KONA_ZKVM_CANARY_PRESTATE` resolve exactly one artifact:
`<PRESTATES_URL>/<PRESTATE>.range.bin.gz`. Startup downloads and decompresses it in memory under
the configured size and request limits, computes the decompressed ELF's SHA-256 digest, derives its
SP1 verification-key hash, and requires both to match `KONA_ZKVM_CANARY_ELF_SHA256` and
`KONA_ZKVM_CANARY_RANGE_VKEY`. The aggregation ELF is not downloaded. The authenticated bytes stay
in memory for the process lifetime; rotating the prestate, vkey, digest, or artifact requires a
restart. The service image contains only the host binary, not a locally generated guest ELF.

Startup is fail-fast and ordered: parse and validate all configuration, authenticate the artifact,
bind the host-utils metrics server, register canary metrics, then emit the structured
`kona-zkvm-canary started` log. Configuration, artifact digest, or vkey failures return non-zero
before a metrics listener is bound. Host-utils serves `/health` unchanged. There is no `/ready`
endpoint or ready gauge: a scrapeable process already holds a validated artifact.

### Configuration

All canary-owned variables use the `KONA_ZKVM_CANARY_` prefix. RPC URLs must use HTTP or HTTPS and
must not contain user information, query parameters, or fragments. Production artifact URLs must
use HTTPS with the same restrictions and redirects disabled. A `file://` artifact directory is
accepted only with `--once`. Numeric limits marked non-zero fail validation when set to zero.

Required configuration:

| Variable | Purpose |
|---|---|
| `KONA_ZKVM_CANARY_SUPERROOT_RPC` | op-supernode endpoint serving `superroot_atTimestamp` |
| `KONA_ZKVM_CANARY_L1_RPC` | L1 execution JSON-RPC endpoint used to pin and canonicalize block identities |
| `KONA_ZKVM_CANARY_L1_BEACON_RPC` | L1 beacon API endpoint used for blob sidecars during witness collection |
| `KONA_ZKVM_CANARY_L2_RPCS` | Comma-separated `<chain-id>=<http(s)-url>` L2 execution endpoints; decimal and `0x` chain IDs are accepted and duplicates are rejected |
| `KONA_ZKVM_CANARY_PRESTATES_URL` | Immutable published-prestates directory; `file://` is diagnostic and `--once` only |
| `KONA_ZKVM_CANARY_PRESTATE` | Aggregation-prestate key naming the paired `.range.bin.gz` artifact |
| `KONA_ZKVM_CANARY_RANGE_VKEY` | Expected SP1 verification-key hash for the decompressed range ELF |
| `KONA_ZKVM_CANARY_ELF_SHA256` | Expected SHA-256 digest of the decompressed range ELF |
| `KONA_ZKVM_CANARY_GUEST_CYCLE_LIMIT` | Non-zero maximum cycles for each range or consolidation guest execution; no default |

Optional execution, scheduling, and input limits:

| Variable | Default | Purpose |
|---|---:|---|
| `KONA_ZKVM_CANARY_ROLLUP_CONFIG_PATHS` | registry | Comma-separated rollup-config JSON files with exact L2 endpoint coverage |
| `KONA_ZKVM_CANARY_L1_CONFIG_PATH` | registry | L1 chain-config JSON override |
| `KONA_ZKVM_CANARY_DEPENDENCY_SET_PATH` | registry | Dependency-set JSON override with exact L2 endpoint coverage |
| `KONA_ZKVM_CANARY_FINALIZED_SPAN` | `1` | Consecutive finalized timestamps per attempt; range `1..=16` |
| `KONA_ZKVM_CANARY_CADENCE_SECONDS` | `300` | Non-zero wait after an attempt completes |
| `KONA_ZKVM_CANARY_JITTER_SECONDS` | `min(30, cadence)` | Maximum additional wait; zero is allowed and the value cannot exceed cadence |
| `KONA_ZKVM_CANARY_ATTEMPT_DEADLINE_SECONDS` | `10800` | Non-zero deadline for cancellable attempt stages |
| `KONA_ZKVM_CANARY_RPC_REQUEST_TIMEOUT_SECONDS` | `30` | Non-zero deadline for each parent JSON-RPC request |
| `KONA_ZKVM_CANARY_ARTIFACT_REQUEST_TIMEOUT_SECONDS` | `60` | Non-zero whole-request artifact deadline |
| `KONA_ZKVM_CANARY_MAX_PARENT_RESPONSE_BYTES` | `4194304` | Non-zero maximum parent JSON-RPC response body |
| `KONA_ZKVM_CANARY_MAX_PARENT_RESPONSE_ENTRIES` | `256` | Non-zero maximum parent response entries and configured chains; maximum `256` |
| `KONA_ZKVM_CANARY_MAX_ARTIFACT_COMPRESSED_BYTES` | `268435456` | Non-zero compressed artifact ceiling |
| `KONA_ZKVM_CANARY_MAX_ARTIFACT_DECOMPRESSED_BYTES` | `1073741824` | Non-zero decompressed ELF ceiling |
| `KONA_ZKVM_CANARY_MEMORY_LIMIT` | `25769803776` | SP1 emulator accounted-memory limit in bytes; not a process RSS ceiling |
| `KONA_ZKVM_CANARY_METRICS_PORT` | disabled | `disabled`, empty, or any numeric zero disables metrics; `auto` binds an ephemeral port; a non-zero port binds that port on all interfaces |

Optional structured logging and telemetry configuration is handled by host-utils:

| Variable | Default | Purpose |
|---|---:|---|
| `KONA_ZKVM_CANARY_LOGGER_NAME` | `kona-sp1` | OpenTelemetry service name |
| `KONA_ZKVM_CANARY_OTLP_ENDPOINT` | `http://localhost:4317` | OpenTelemetry log-export endpoint |
| `KONA_ZKVM_CANARY_OTLP_ENABLED` | `false` | Enable OpenTelemetry log export |
| `KONA_ZKVM_CANARY_LOG_FORMAT` | `pretty` | `pretty` or structured `json` logs |

The logger also observes `RUST_LOG` and `NO_COLOR`. TLS and HTTP clients may observe the standard
certificate and proxy variables. `KONA_SP1_ELF_DIR` is not a canary input: the canary uses only its
authenticated in-memory artifact. `TRACE_FILE` must be unset; startup rejects it because SP1's
cycle-tracker feature can otherwise persist a profile, while the canary has a no-file-output
contract.

### Execution bounds and termination

The bounds serve different purposes:

- `GUEST_CYCLE_LIMIT` is applied independently to range and consolidation execution. SP1 stops an
  execution that exceeds it and releases that work; the run outcome is `cycle_limit_exceeded`, not
  a guest rejection or divergence.
- `MEMORY_LIMIT` is passed to the in-process SP1 CPU emulator and bounds memory tracked by its
  internal accounting. Allocator overhead and other process memory are outside that accounting, so
  this is not a hard RSS ceiling. The process metrics below expose observed resident and virtual
  memory; the orchestrator or cgroup remains the hard process-memory backstop.
- `ATTEMPT_DEADLINE_SECONDS` covers cancellable selection and witness collection. It deliberately
  does not wrap SP1 execution in a Tokio timeout: SP1 uses blocking work that Tokio cannot cancel,
  so such a timeout would report completion while retaining the CPU and memory load.

If the emulator wedges beyond the cycle and memory bounds, or the process is killed for memory,
the orchestrator must restart it. `SIGINT` and `SIGTERM` stop all further scheduling and exit the
process. A signal received during SP1 execution abandons that execution by terminating the process;
the service does not claim to cancel the in-flight Tokio blocking task.

### Metrics

When enabled, host-utils serves Prometheus samples and updates process metrics every 750 ms. The
canary metric namespace is `kona_zkvm_canary`:

| Metric | Meaning |
|---|---|
| `kona_zkvm_canary_up` | `1` after metrics registration with an authenticated artifact; `0` when one-shot operation completes normally |
| `kona_zkvm_canary_scheduler_heartbeat_timestamp_seconds` | Unix time of the latest selection cycle |
| `kona_zkvm_canary_run_active` | Whether the one sequential attempt is active |
| `kona_zkvm_canary_last_attempt_timestamp_seconds` / `kona_zkvm_canary_last_success_timestamp_seconds` | Completion time of the latest attempt / valid attempt; zero means unknown |
| `kona_zkvm_canary_last_attempted_target_timestamp` / `kona_zkvm_canary_last_successful_target_timestamp` | Latest attempted / valid finalized target; zero means unknown |
| `kona_zkvm_canary_consecutive_failures` | Consecutive non-valid terminal outcomes |
| `kona_zkvm_canary_runs_total{outcome}` | Attempt count by `valid`, `guest_rejected`, `output_mismatch`, `input_error`, `cycle_limit_exceeded`, or `timeout` |
| `kona_zkvm_canary_run_duration_seconds` | Total attempt duration |
| `kona_zkvm_canary_input_selection_duration_seconds` | Canonical snapshot selection duration |
| `kona_zkvm_canary_stage_witness_duration_seconds{mode}` / `kona_zkvm_canary_stage_execute_duration_seconds{mode}` | Witness and SP1 execution duration for `range` or `consolidation` |
| `kona_zkvm_canary_selected_span_length` / `kona_zkvm_canary_selected_chain_count` | Timestamp and chain counts in the latest selected input |
| `kona_zkvm_canary_finalized_target_lag_seconds` | Wall-clock lag of the latest attempted finalized target |
| `kona_zkvm_canary_report_target_timestamp{mode}` | Target associated with the latest completed SP1 report for the mode |
| `kona_zkvm_canary_report_pgu{mode}` | Latest normalized SP1 proving-gas-unit estimate; an absent SP1 value leaves the prior gauge unchanged |
| `kona_zkvm_canary_report_instructions{mode}` / `kona_zkvm_canary_report_syscalls{mode}` | Latest instruction and syscall totals |
| `kona_zkvm_canary_report_record_bytes{mode}` | Latest estimated SP1 execution-record size; not process memory |
| `kona_zkvm_canary_report_touched_addresses{mode}` | Count of distinct touched guest addresses; not bytes or RSS |
| `kona_zkvm_canary_report_exit_code{mode}` | Latest SP1 guest exit code |
| `kona_zkvm_canary_artifact_info{prestate,range_vkey,elf_sha256,sp1_version}` | Constant `1` identifying the authenticated artifact and pinned SP1 release |

Host-utils additionally exports unprefixed process samples including
`process_resident_memory_bytes`, `process_virtual_memory_bytes`,
`process_virtual_memory_max_bytes`, `process_cpu_seconds_total`, file-descriptor counts, thread
count, and process start time where the platform supports them. Because both guest modes run in
the service process, resident-memory samples include SP1 execution rather than a child process.
Metric labels never contain RPC URLs, run IDs, roots, chain IDs, error strings, opcode names, or
syscall names. Per-phase cycle and invocation totals remain available in the bounded
`range_report` and `consolidation_report` fields of each structured attempt log; they are not
exported as Prometheus series.

### One-shot diagnostics

`--once` calls the same `Runner::run_one` selection and execution path as the loop. It emits the
terminal outcome and both bounded range/consolidation report summaries as structured fields, then
exits `0` for `valid`, `1` for `guest_rejected` or `output_mismatch`, and `2` for every other
outcome. It accepts no result path and writes no artifact, witness, report, or result file. For a
local authenticated artifact directory:

```bash
KONA_ZKVM_CANARY_PRESTATES_URL="file:///absolute/path/to/prestates/" \
KONA_ZKVM_CANARY_PRESTATE="0x..." \
KONA_ZKVM_CANARY_RANGE_VKEY="0x..." \
KONA_ZKVM_CANARY_ELF_SHA256="0x..." \
KONA_ZKVM_CANARY_GUEST_CYCLE_LIMIT="..." \
KONA_ZKVM_CANARY_SUPERROOT_RPC="http://127.0.0.1:9545" \
KONA_ZKVM_CANARY_L1_RPC="http://127.0.0.1:8545" \
KONA_ZKVM_CANARY_L1_BEACON_RPC="http://127.0.0.1:5052" \
KONA_ZKVM_CANARY_L2_RPCS="901=http://127.0.0.1:9546" \
cargo run --locked --package kona-zkvm-canary --bin kona-zkvm-canary -- --once
```

All snapshots, witnesses, authenticated ELF bytes, and report data remain in memory. The canary
does not generate or persist a proof in either looping or one-shot operation.

## Building

Programs are compiled for the zkVM target through the recipes in this directory's `justfile`.

The `cargo prove` subcommand is pinned by `mise.toml`. Install the native Succinct
toolchain for non-Docker local builds with:

```bash
just install-sp1-toolchain
```

## Testing (SP1 execute action tests)

The `super-range-executor` crate (`crates/super-range-executor`) builds a host binary,
`kona-sp1-super-range-executor`, that runs the `super-range` guest in SP1 **execute** mode (no
proving) against a real chain's witness. It resolves the span from `superroot_atTimestamp`,
collects range and consolidation witnesses through kona's `InteropHost`, runs the
`super-range` ELF in the SP1 emulator, and exits `0` (valid claim) / `1` (invalid claim) / `2`
(infrastructure error) — mirroring the native fault-proof program convention.

The op-e2e action test `TestSP1SuperRangeSimpleEmptyChain`
(`rust/kona/tests/proofs/sp1_simple_program_test.go`) drives this binary against an
in-process action-test chain, exercising both guest modes end-to-end on real inputs. Run it
with:

```bash
cd rust/kona/tests && just action-tests-sp1
```

That recipe builds the guest ELFs (`just build-elfs`, Dockerized SP1 toolchain), builds the
`super-range-executor` binary, and runs the test with `KONA_SP1_SUPER_RANGE_ELF_EXECUTOR_PATH`
and `KONA_SP1_ELF_DIR` set — the same two variables the acceptance full-ELF suite reads. The
executor loads the `super-range` ELF at runtime. The test skips when the executor-path variable
is unset, so the heavy SP1 toolchain is only required when explicitly running the SP1 action
tests.

Because the executor is a separate process that resolves the transition itself, the action-test
harness serves op-node's superroot API over a loopback HTTP listener
(`L2Verifier.StartSuperRootHTTPRPC`) and passes it as `--supernode-address`. op-node answers
with a one-chain response, which is what the action-test chain is.

For faster coverage of the super-range logic, the same executor also supports `--native-core`.
This mode still collects the real witnesses, but replays them through the shared native cores
instead of executing the SP1 ELF. Use the default SP1 execute path for a small smoke test of
the ELF, SP1 stdin, and public-values boundary; use `--native-core` when broad action-test
coverage would otherwise multiply SP1 emulator cost.

The test covers both an honest claim and an invalid claim. Note the invalid-claim path is
driven by **corrupting the claim the guest sees**, not by feeding the executor a wrong claim:
the executor synthesizes the agreed pre-state and the claim from the supernode and collects
witnesses against them, so there is nothing to pass a junk value to, and a witness collected
against a bad claim would fail host-side before the guest ran (a confusing infra error, exit
2). So an invalid-claim test sets `--corrupt-claimed-root` (via `WithCorruptClaim()` in the Go
harness), which flips a bit in the claimed optimistic output root *after* witness collection,
so the guest re-derives the real root, finds the mismatch, and aborts (exit 1) — a soundness
smoke test that a false transition cannot be executed (and thus could not be proven). If the
guest instead runs the tampered claim to completion and agrees with the honest outputs, the
executor exits `2` rather than reporting the claim valid. Do **not** write an SP1 negative test
by passing a junk `WithL2Claim(...)`.
## Dependencies

This integration depends on:
- SP1 SDK and zkVM runtime
- Core Kona libraries (`kona-proof`, `kona-derive`, `kona-executor`, etc.)
- Alloy and OP-Alloy for Ethereum types
- RocksDB for witness data storage

## License and Attribution

This implementation is derived from [OP-Succinct](https://github.com/succinctlabs/op-succinct)
by Succinct Labs and incorporates code licensed under the MIT License and Apache License 2.0.
Significant modifications have been made to integrate with the Kona monorepo architecture.

See [LICENSE-THIRD-PARTY](./LICENSE-THIRD-PARTY) for full license details and attribution.
