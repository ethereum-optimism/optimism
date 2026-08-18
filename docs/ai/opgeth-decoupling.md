# op-geth Decoupling Analysis

This document analyses the dependencies of the optimism monorepo Go services on op-geth–specific
APIs, and proposes decoupling strategies for each. The goal is to depend on upstream go-ethereum
instead of op-geth without opening upstream PRs. Tracking issue: #20257. Last full revisit:
2026-07 (verified by an upstream-build spike; added §§15–19).

Sections are numbered stably (§0–§19) because issues reference them. Done sections keep their
context and decisions but not their implementation detail — the landed code in `op-core/*` is its
own reference. Work-list detail (exact files, counts) is deliberately absent: it goes stale, and
an agent re-derives it in minutes from the grep recipes given per section.

**Scope: the whole monorepo.** The monorepo is a **single Go module** with one go-ethereum
`replace` directive, so at the flip *everything* in the tree must build — and behave — against
upstream go-ethereum; components can only leave scope by deletion. Each component falls into one
of three fates:

1. **Swap to `op-core/*` (+ upstream go-ethereum)** — code that stays in Go; where it used
   op-geth-specific types/helpers they swap to `op-core/*`, and some members need no changes at
   all (§12). The bulk: op-node, op-service, op-batcher, op-proposer,
   op-challenger, op-faucet, op-supernode, cannon; the test suites (op-e2e, op-acceptance-tests,
   op-devstack) and the Go proof tests under `rust/kona/tests/proofs/`; `op-chain-ops/genesis`
   (#21281) and the surviving `op-chain-ops/cmd/check-*` checkers (§18); **op-sync-tester** — a
   CL-sync EL *mock* that does **no execution** (it proxies a real op-reth EL and gates
   visibility), whose one non-swap bit, the OP-aware `PayloadID` hash, is tracked in #21525; and
   the **fork's infrastructure extensions** to geth's `log`/`rpc` packages, re-homed into owned
   op-service layers (§15).
2. **Migrate execution to op-reth / Rust** — anything that builds or executes blocks, or needs
   fork-only EVM hooks. op-acceptance-tests sequences op-reth-only for Karst+ (#21182);
   op-e2e/actions moves onto an op-reth-test-engine subprocess EL (#20415, #21196); op-e2e system
   tests and op-devstack/sysgo retire their in-process op-geth L2 EL (§17); and
   `op-chain-ops/script` + op-deployer move to a Rust script engine (§16).
3. **Delete** — geth-as-library tools with no remaining need, to be reimplemented in Rust against
   op-reth if ever needed again: op-simulate and op-run-block (#21282), `op-wheel/cheat` and the
   stale pre-Holocene `check-*` tools (§18).

Once every component is in fate 1, 2, or 3, the go.mod `replace` flips to upstream go-ethereum
(#20266). A scheduled **CI ratchet** job rehearses the flip continuously (see §19).

**Compile-compat is necessary, not sufficient.** Upstream go-ethereum does not implement the OP
state transition, so code can compile and silently behave wrongly. The classifying question for
every geth-as-library consumer: *does it execute OP state transitions?* Yes → op-reth or delete
(fate 2/3). No — plain-EVM script execution, storage-level surgery, header/RLP work → upstream is
behaviorally fine (fate 1).

The op-geth diff vs. upstream go-ethereum (currently based on v1.17.2) can be summarised in four
kinds of change:

1. **New standalone types/files** – `DepositTx`, `PostExecTx`, `RollupCostData`, the
   `superchain/` package, eip1559 Holocene/Jovian helpers.
2. **Fields/methods added to existing upstream types** – `Transaction` methods (`IsDepositTx`,
   `SourceHash`, `Mint`, `IsSystemTx`, `RollupCostData`), `Receipt` L1-cost fields, `ChainConfig`
   OP hardfork fields and methods, `PayloadAttributes` extensions.
3. **Infrastructure hooks in general-purpose packages** – `log` context methods, `rpc` recording
   hooks, EVM override hooks in `vm.Config` (§15, §16). Having a fork made "just add a hook to
   geth" cheap; the decoupling converts each such shortcut into a layer we own or a feature
   decision.
4. **Config/CLI wiring** – how op-geth starts: not relevant to the monorepo.

---

## Target package layout in the monorepo

All op-geth-specific code lives in `op-core/` packages (alongside the pre-existing
`op-core/forks/` and `op-core/predeploys/`). **All of these have landed:**

| op-geth source | Monorepo package | Content |
|---|---|---|
| `core/types/deposit_tx.go`, `post_exec_tx.go`, `receipt_opstack.go` | `op-core/types/` | `DepositTx`, `PostExecTx`, `Receipt` + helpers |
| `core/types/rollup_cost.go` | `op-core/fees/` | L1 cost + operator-fee math |
| `params/config_op.go`, `params/superchain.go` | `op-core/params/` | `OptimismConfig`, standalone `ChainConfig`, loaders |
| `superchain/` + `sync-superchain.sh` | `op-core/superchain/` | embedded registry (absorbed `op-service/superutil`) |
| `consensus/misc/eip1559/eip1559_optimism.go` | `op-core/eip1559/` | Holocene/Jovian extraData helpers |

### Design principles (apply to all future op-core additions)

- **Monorepo-best, not op-geth-faithful.** Names and shapes diverge freely; only *computed
  values* and *wire formats* must match. Equivalence is pinned by **live differential tests**
  that compare against op-geth while the build still resolves go-ethereum to the fork; all
  differential tests are deleted at the cutover (§ #20266).
- **Naming**: inside `op-core/` the `Optimism`/`OPStack` prefix is dropped (redundant), with two
  deliberate exceptions: `OptimismConfig` (pairs with the JSON-wire-load-bearing
  `ChainConfig.Optimism` field name) and the `{Validate,Decode,Encode}OptimismExtraData` helpers
  (there "Optimism" names the extraData *format* and avoids collision with per-fork helpers).
- Files needing both upstream and monorepo variants use import aliases `optypes`, `opparams`,
  `opfees`.
- **Wire compatibility** where types serialise: JSON tags identical to op-geth
  (`OptimismConfig`), byte-identical encodings (`DepositTx`: `0x7E || RLP(struct)`), verified by
  the differential tests.

### Cross-implementation parity

Every OP wire format and hash rule has a second implementation in this repo: `rust/op-alloy`
defines the OP transaction and receipt types that both op-reth and kona consume, opposite op-geth's
Go definitions that `op-core/*` mirrors. Both sides serve one chain, so a value one produces and
the other rejects is a consensus split — and agreeing with op-geth establishes only the Go half.

- **op-geth parity is necessary, never sufficient.** The differential tests pin the Go side to
  the fork; by construction they cannot see a rule where the fork itself is wrong. Any new or
  changed encoding, hash or accept-set is checked against `rust/op-alloy` too, and a disagreement
  is a question of which one is correct — decided from the [specs](https://specs.optimism.io), not
  from the incumbent. Fixing op-geth (and bumping the pin) is a normal outcome.
- **Pin the agreed value in both languages.** A shared golden vector — the same literal asserted
  in the Go and the Rust suite — is what keeps the pair locked together;
  `TestPostExecTxHashGoldenVector` (`op-core/types/post_exec_tx_test.go`) and
  `rust/op-alloy/crates/consensus/src/post_exec/tests.rs` pin the post-exec (`0x7D`) tx hash this
  way. Reference-implementation fixtures are not a substitute: they only cover values that
  implementation can produce.
- **The vectors outlive the differential tests.** op-geth differential tests are deleted at the
  cutover (§ #20266). The cross-implementation vectors are not — after the flip they are the only
  thing tying the Go types to op-reth and kona.

For batcher-controlled bytes (batches, frames, channels) the second implementation is kona's own
decoder rather than op-alloy's types, and the rule takes a different shape — accept-sets compared
over a generated corpus rather than single values; see [derivation.md](derivation.md).

---

## 0. Remove ProtocolVersions watching from op-node — **DONE**

Rather than port the deprecated protocol-versions signalling mechanism (op-node watching the L1
`ProtocolVersions` contract, `engine_signalSuperchainV1`, halt-when-outdated) into `op-core`, the
feature was **removed entirely** — op-node, kona/op-reth, contracts, and all Go wiring (#20258,
#20311, #20317, #20441, #20527). This shrank the surface to decouple and avoided carrying dead
code.

Careful when grepping: `op-service/apis/p2p.go` / `op-node/p2p/rpc_server.go` contain an
unrelated libp2p `ProtocolVersion` string — same name, different concept, left alone.

---

## 1. `core/types` – OP Stack transaction types (`DepositTx`, `PostExecTx`) — types **DONE**, swaps open

`op-core/types` ships `DepositTx` (`0x7E || RLP(struct)`) and `PostExecTx` with
`MarshalBinary`/`Unmarshal*` and differential tests (#20262). Read the package for the API.

Two wire facts worth keeping in mind (both encoded in the landed implementation):

- **`PostExecTx` (`0x7D`) has no RLP envelope** — `0x7D || data` verbatim; op-geth treats the
  payload as opaque bytes (op-alloy/kona parse it as a versioned `PostExecPayload`). The op-core
  type mirrors the opaque handling.
- **`PostExecTx` is a canonical block transaction**: on Lagoon+ it can be the *last* tx of a
  block (sequencer rebates). So an L2 block's tx list is `[L1-info deposit (0x7E), …user txs…,
  optional post-exec (0x7D)]` — under upstream go-ethereum a full-block decode fails on **both**
  synthetic types, which is why the OP-aware client (§11) must route both out.

**Open work (#20269, #20263):** swap the construction/decoding call sites. The op-geth pattern
`types.NewTx(&types.DepositTx{...}).MarshalBinary()` becomes
`optypes.DepositTx{...}.MarshalBinary()` — the `*types.Transaction` wrapper is eliminated, struct
straight to bytes. `UnmarshalDepositLogEvent` returns `*optypes.DepositTx`. Sites cluster in
op-node derivation (deposits, one `*_upgrade_transactions.go` per fork — the set grows every
fork) and span-batch code (`PostExecTxType`); derive the current list with
`grep -rn 'types\.\(DepositTx\|PostExecTx\)' --include='*.go'` over the go-ethereum import.

---

## 2. `core/types` – Transaction methods on RPC-received transactions

op-geth adds methods to `*types.Transaction` (`IsDepositTx`, `IsSystemTx`, `SourceHash`, `Mint`,
`RollupCostData`) that the monorepo calls on txs received via Engine API / RPC. `tx.Type()` and
`tx.Data()` are standard upstream — no change. The replacements exist in `op-core/types` (free
functions and `UnmarshalDepositTx`-then-read-fields) and `op-core/fees.TxRollupCostData`.

**These `*types.Transaction` helpers are transition scaffolding, not the destination.** They only
work while the build resolves go-ethereum to op-geth (where a `*types.Transaction` can carry a
`0x7E`/`0x7D` tx); they let call sites drop op-geth's methods *before* the underlying
representation changes. After the cutover an upstream `*types.Transaction` can never hold a
deposit or post-exec tx, so **the helpers and their differential tests are deleted at the cutover
(§ #20266)**. The durable shape never wraps an OP tx in `*types.Transaction`: raw bytes decode
into `optypes.DepositTx`/`optypes.PostExecTx`, and code never asks "is this a deposit?" of a
generic tx because the sources accessors already partition by class (§11).

---

## 3. `core/types` – Receipt L1-cost fields — type **DONE**, wiring open

`op-core/types.Receipt` (landed, #20262) embeds upstream `types.Receipt` and adds op-geth's
**complete** OP receipt field set (L1 fee fields, operator fee, deposit nonce, DA footprint) with
a custom JSON unmarshaler and a differential test. Complete deliberately: the e2e/acceptance
suites read more fields than the one production consumer, and a partial set would silently drop
fields at cutover — invisible while the replace still points at op-geth.

Production consumer: `op-service/txinclude` reads the fee fields off receipts fetched via RPC.
**Open work:** its `EL.TransactionReceipt` (and `apis.EthClient` / `FetchReceipts` generally)
return `*optypes.Receipt` instead of `*types.Receipt` — part of §11 (#20264). Under upstream,
the standard unmarshaler drops the OP fields silently; only the raw-RPC path through
`op-service/sources` + `optypes.Receipt` preserves them.

Context: op-node itself never reads receipt fields — it reads `L1BlockInfo` from deposit-tx
calldata; `txinclude`'s cost oracle reads the L1Block predeploy via `eth_call`. Receipts matter
mainly for txinclude's post-inclusion accounting and the test suites (§13).

---

## 4. `op-core/fees` – L1 cost and operator-fee math — **DONE** (one site rides §11)

Landed (#20261) and wired into `op-service/txinclude`. API is monorepo-best (see design
principles): plain per-era free functions (`L1CostBedrock/Ecotone/Fjord` + `L1FeeParams`) instead
of op-geth's closure factories, `OperatorCostIsthmus/Jovian` replacing inline math,
`TxRollupCostData(tx)` replacing the fork's Transaction method. Values pinned by differential
tests. Notable choice: both txinclude sites use the **Jovian** operator-fee formula
unconditionally — exact on all production chains, and on pre-Jovian chains it only
*over*-estimates, which over-reserves budget rather than under-budgeting.

Remaining: the `op-batcher` `tx.RollupCostData()` site swaps to `opfees.TxRollupCostData` as part
of the op-batcher → sources migration (§11), not here.

---

## 5. `params.ChainConfig` – OP hardfork methods — package **DONE** (#20260), swaps open (#20270)

`op-core/params` ships `OptimismConfig`, a **standalone** `ChainConfig`, and `GethChainConfig()`.

**Why standalone, not embedding upstream's `params.ChainConfig`** (decision worth remembering):
embedding would promote go-ethereum's OP-unaware EIP-1559 methods (`ElasticityMultiplier`,
`BaseFeeChangeDenominator`, `LatestFork`), which would read the embedded (nil) fields and
silently return L1 defaults — a footgun on consensus-relevant fee params, present both during the
transition and after the cutover. A standalone type cannot promote them. Code that needs a
concrete go-ethereum config converts explicitly via `GethChainConfig()` (Ethereum schedule
derived from the OP schedule: Shanghai=Canyon, Cancun=Ecotone, Prague=Isthmus, Osaka=Karst —
op-geth's own loader is missing Osaka=Karst; the monorepo follows op-reth, which is correct).

While the build still resolves go-ethereum to op-geth, `GethChainConfig()` also carries the OP
fork fields and `Optimism` struct on the produced config (in-process test ELs need them). Those
fields don't exist upstream — **shedding them is a cutover edit** (#20266).

**Open work (#20270):** `rollup.Config.ChainOpConfig` changes type to `*opparams.OptimismConfig`
(JSON tags identical → wire-compatible); `op-service/eth.BlockAsPayload`'s
`*params.ChainConfig` parameter becomes a small local `HardforkConfig` interface
(`IsCanyon`/`IsIsthmus`) that `rollup.Config` already satisfies; remaining direct OP-hardfork
reads on go-ethereum's `*params.ChainConfig` go through `rollup.Config` or
`*opparams.ChainConfig`. Grep recipe: files importing `go-ethereum/params` that call
`Is<OpFork>(` or reference `OptimismConfig`.

---

## 6. `params/superchain.go` – `LoadOPStackChainConfig` — **DONE**

Moved to `op-core/superchain`: `(*superchain.ChainConfig).OpChainConfig()` plus the by-chain-ID
loader `superchain.LoadOpChainConfig` (ex-superutil). §7 has the layering rationale.

---

## 7. `superchain/` package — **DONE** (#20267, #21487)

The registry package moved verbatim to `op-core/superchain` (embedded zip regenerated from the
repo-root `superchain-registry` submodule by `sync-superchain.sh`; the zip is gitignored, pinned
by a committed `.sha256`). `op-service/superutil` was deleted; all consumers import
`op-core/superchain`.

Layering decision worth remembering: `op-core/params` holds pure config types and must stay
free of `op-core/superchain`, which embeds the bundle — anything in its build closure needs the
bundle generated before it compiles, and external module consumers (the superchain-registry ops
tooling) cannot generate it at all. The registry→`ChainConfig` conversion (`OpChainConfig`,
`LoadOpChainConfig`) therefore lives in **`op-core/superchain`**, which imports `op-core/params`
one-way. Transitive guard tests in `op-node/rollup`, `op-chain-ops/script`, and
`op-fetcher/pkg/fetcher/fetch/script` enforce the boundary (`op-service/testutils/depguard`).

`rollup.Config` carries the full OP hardfork schedule itself (loaded from the registry,
bypassing any geth `ChainConfig`); future forks extend `rollup.Config` directly. Any remaining
gaps ride #20270.

---

## 8. `consensus/misc/eip1559` – Holocene/Jovian helpers — op-node **DONE** (#20268), helpers missing for §13/§14/§18

`op-core/eip1559` landed and op-node is fully switched (no op-node production code imports geth's
eip1559; one test file still does — a fork-comparison import that goes away at cutover).
Signatures operate on `[]byte`/`uint64` only.

Still absent from `op-core/eip1559`: only `EncodeOptimismExtraData` and `DecodeHoloceneExtraData`
(the Holocene/Jovian encode helpers are already present) — add them when §13 migrates the op-e2e
helpers. The geth-eip1559 imports remaining in genesis (§14) and the check-* tools (§18) are pure
import swaps onto the existing op-core helpers.

---

## 9. `beacon/engine` – `PayloadID` type alias — no change needed

`engine.PayloadID` is standard upstream. (op-sync-tester's OP-aware PayloadID *hash* is different
— see #21525.)

---

## 10. `beacon/engine` – `PayloadAttributes` / `ExecutableData` extensions — no change needed

op-service defines its own Engine-API types with the OP extensions. `BlockAsPayload` touches only
upstream-stable block/header fields; it just needs the `HardforkConfig` change from §5.

---

## 11. `ethclient` — JSON decoding of L2 blocks and receipts — open (#20264)

Most `ethclient` usage in the monorepo is safe (L1-only, or standard scalar/receipt-log reads).
Exactly two failure modes appear when go-ethereum resolves upstream, and both only bite against
**L2** endpoints:

1. **L2 block decoding fails**: upstream `Transaction.UnmarshalJSON` rejects the deposit
   (`0x7E`) and post-exec (`0x7D`) txs present in every L2 block.
2. **OP receipt fields silently drop**: upstream `types.Receipt` doesn't know the L1-cost /
   operator-fee fields (§3).

The audited unsafe production paths are the **op-batcher L2 block fetch** (`driver.go`'s
`L2Client`) and **txinclude receipt fetch**; op-faucet, op-proposer, op-challenger (headers
only), op-supernode (receipts via sources; reads only `Logs`) and cannon (no blockchain-layer
coupling at all — see §12) need nothing. Re-derive with
`grep -rl 'go-ethereum/ethclient'` and classify each client by endpoint.

### Strategy: make `op-service/sources` the canonical OP-aware client — **no wrapper type**

The sources clients already speak raw JSON-RPC (not `ethclient`). Extend them; do **not**
introduce an "OP ethclient" or a go-ethereum-style `Transaction`+`TxData` wrapper. The codebase's
dominant L2-tx currency is already opaque `[]hexutil.Bytes`; where typed access is needed it is
almost always class-specific, and a wrapper would re-import the signer/hashing machinery we are
shedding. Instead, partition the `apis.EthClient` accessors by tx class:

- `InfoAndUserTxs(...) (eth.BlockInfo, types.Transactions, error)` — **excludes** the synthetic
  OP types (`0x7E`/`0x7D`); the remainder are standard Ethereum txs, so the list is plain
  upstream `types.Transactions` and decodes fine post-cutover. Serves the dominant "skip
  deposits, operate on user txs" pattern (op-batcher DA estimation, most test iterators).
- `InfoAndDeposits(...) (eth.BlockInfo, []*optypes.DepositTx, error)` — deposits as op-core
  structs.
- `InfoAndFirstDeposit(...)` — header + the L1-info deposit only, for hot paths (pattern
  prototyped in the closed PR #20532; revive).
- No `InfoAndPostExecTx` **yet, deliberately**: every known consumer of post-exec data reads it
  from batch/flashblock bytes (span-batch decode, `op-chain-ops/pkg/sdm`, the flashblock
  client), not from L2 block bodies — block-fetching code only needs the *exclusion* that
  `InfoAndUserTxs` provides. Add the accessor when a consumer (e.g. a rebate checker) first
  needs to read the trailing `0x7D` tx off a block.

`InfoAndTxsBy*` stays as-is for **L1**. Internally `RPCBlock` keeps txs as raw per-tx bytes: the
transactions-trie root is recomputed directly from those bytes (a typed tx's trie leaf *is* its
opaque encoding), synthetic-tx detection is a type-byte check, and each class decodes on demand.

**Index-correlation caveat** (for §13): callers that iterate the full tx list and correlate by
position with a receipts list misalign once `InfoAndUserTxs` drops the synthetic txs — such
tests must filter receipts in lockstep or use a paired accessor.

Concrete migrations in this issue: op-batcher's `L2Client` moves to `InfoAndUserTxs` (it only
filters deposits for DA estimation; `RollupCostData` → `opfees.TxRollupCostData`);
`op-service/dial`'s `L2EndpointProvider.EthClient` returns a sources-shaped client (or the
interface is phased out); txinclude's `EL.TransactionReceipt` → `*optypes.Receipt` via raw RPC.

---

## 12. `op-proposer`, `op-challenger`, `op-supernode`, `cannon` — audited: no migration needed

- **op-proposer** — pure L1; contract calls via `sources/batching.MultiCaller`; no deposit txs,
  OP receipt fields, or OP params anywhere.
- **op-challenger** — ~90% L1; its single L2 access fetches *headers* only (no txs → no decode
  issue). If it ever needs L2 block bodies, migrate per §11 then.
- **op-supernode** — no direct `ethclient`; receipts via sources interfaces, reads only
  `receipt.Logs`. Picks up §11 improvements transparently.
- **cannon** — imports only `common`/`hexutil`/`log`; the preimage oracle treats bytes opaquely.
  Zero blockchain-layer coupling.

---

## 13. Tests: `op-e2e`, `op-acceptance-tests`, `op-devstack` — open (#20265)

Dependency profiles: **op-acceptance-tests** has zero direct `ethclient` imports — everything
goes through the op-devstack DSL to `apis.EthClient`, so it picks up §11's implementation swap
automatically (this is why §11 rejects a parallel wrapper type). **op-e2e** uses `ethclient` and
OP receipt fields directly in dozens of files. **op-devstack**'s DSL layer is clean;
`sysgo/` calls `ethclient.Dial` directly in a handful of files (mostly L1 contract/dispute-game
setup).

### Strategy

1. **`apis.EthClient.TransactionReceipt` / `FetchReceipts`** return `*optypes.Receipt` (§3);
   the many test sites reading OP receipt fields keep working unchanged.
2. **L2 tx iteration** migrates to the class-partitioned accessors from §11 (`InfoAndUserTxs`
   removes the per-tx `IsDepositTx()` checks entirely). Mind the index-correlation caveat (§11).
3. **`op-e2e/e2eutils/geth/wait.go`** (the central wait helpers, ~26 importers): none of them
   semantically need OP specifics — the failure is transport-level (full-block decode on L2).
   Split into header-only variants (`HeaderByNumber`-based, L2-safe) for the majority of callers
   that discard the block, keep a full-block variant backed by `apis.EthClient` for the few that
   read the body; `WaitForTransaction` returns `*optypes.Receipt`.
4. **`e2esys.SystemConfig.L2Client`** becomes `apis.EthClient`; L1 stays `*ethclient.Client`.
5. **`op-devstack/sysgo`**: per-call audit; L1 callers stay on `ethclient`, L2 callers migrate
   to sources.
6. **Hardfork checks** on geth `*params.ChainConfig` in tests migrate to `*opparams.ChainConfig`
   (mostly automatic once the loaders return the op-core type).
7. **`op-e2e/opgeth/` is deleted** (#20275), not migrated — it intentionally exercises op-geth
   engine internals; op-reth equivalents are introduced separately.

### Audit notes

L2-endpoint full-block/tx fetches were ~20 sites at the 2025 audit — re-derive before working
(`grep -rn 'BlockByNumber\|BlockByHash\|TransactionByHash' op-e2e/`, classify by target
endpoint); L1 fetches are safe; the generic utilities (`geth/find.go`, `wait.go`) are fixed once
via item 3 and cover both call patterns. The simulated backend (`ethclient/simulated`) uses only
upstream-stable APIs.

---

## 14. Genesis tooling — op-geth as a *library*, not an engine — open (#21281, #21282)

Removing op-geth has two parts: op-geth as the execution *engine* (in-process ELs — deleted or
replaced by op-reth; §17, #21196) and op-geth as a *library* in offline tooling. The genuine
in-scope library consumer is **`op-chain-ops/genesis`** (`BuildL2Genesis`: genesis state-root +
`genesis.json`). `op-simulate`/`op-run-block` are deleted instead (#21282, no importers).

The only op-geth *diff* the genesis path relies on: for Isthmus+ chains the genesis block's
`WithdrawalsHash` is the storage root of the `L2ToL1MessagePasser` predeploy (via the upstream
`GetStorageRoot` primitive) instead of the empty hash. Migration (#21281): genesis builds an
`opparams.ChainConfig` and serialises *that* into `genesis.json` (it carries the OP fields;
upstream's config cannot); the block/state-root is computed against upstream via
`GethChainConfig()`; the ~15-line Isthmus withdrawals-root rule is ported on top of upstream
`state`/`triedb` + `op-core/predeploys`. This is why `GethChainConfig()` is a real bridge rather
than throwaway.

---

## 15. Infrastructure fork extensions — `log` and `rpc`

op-geth doesn't only add OP *protocol* code; having a fork made it cheap to add general-purpose
hooks to geth's infrastructure packages, and op-service foundations grew to depend on them. They
sit under every service, so they break the whole tree at cutover, and two of them are *features*
rather than symbols. (Found by the 2026-07 upstream-build spike; §19 keeps finding such uses.)

**Log context extensions** — fork adds `Logger.SetContext`, `WriteCtx`, `LogAttrs`, and the
`Trace/…/ErrorContext` methods; `op-service/log`'s logfilter feature and `op-service/testlog`
build on them. Strategy: **own the log layer** — depending on an internal log API is better
layering than importing `geth/log` everywhere anyway.

- *Phase 1 (pre-cutover, mechanical, per-component):* `oplog.Logger` becomes a **type alias** of
  geth `log.Logger`, plus re-exports of the package-level API in use (`Root`, `SetDefault`,
  `NewLogger`, level parsing). Sweep every `go-ethereum/log` import to `op-service/log` — a
  no-op rename while the alias holds.
- *Phase 2 (at cutover):* flip the alias to an owned interface (upstream's method set + the
  context methods) with an slog-backed implementation. Implement over `slog` — don't copy
  upstream's LGPL log package; the fork's context-extension logic is OP-authored and ports. The
  owned interface is a superset of upstream's, so our loggers still satisfy `log.Logger` where we
  hand one into geth code (e.g. the in-process L1 geth in op-e2e).

**RPC recording hooks** — fork adds `rpc.Recorder`/`RecordedMsg`/`RecordDone`/`WithRecorder`
inside the geth RPC client *and server*; `op-service/metrics` (RPC metrics), `op-service/rpc`,
and `op-service/client` build on them. Client-side recording moves into our own client wrappers
(a seam we own); server-side needs a new interception point (HTTP middleware or handler
wrapping) — small design task, metric names/labels must be preserved. `rpc.JsonError`
(op-test-sequencer) is the same family, trivially replaced by a local error type.

**One-off fork symbols** in the same spirit — `Transaction.SetBlobTxSidecar` (op-service/signer),
`types.NewIsthmusSigner`, the fork-changed `types.NewBlock(…, *BlockConfig)` signature
(op-service/testutils), `types.LogForStorage` (op-chain-ops/crossdomain; deleted upstream),
`params.InteropCrossL2InboxAddress` (used by `op-core/interop/messages` — switch to
`op-core/predeploys`) — ride the §2-style call-site swaps (#20263 family). Derive current sites
by grepping the symbol; listed file snapshots go stale.

---

## 16. `op-chain-ops/script` + op-deployer — Rust script engine

`op-chain-ops/script` is the Foundry-style in-process forge-script executor; op-deployer runs all
deployment/genesis scripts through it. Semantically it is **plain L1 EVM** — its chain config
activates only Ethereum forks (the OP fork fields are explicitly nil) — so by §"scope" rules it
*could* be fate 1. But its cheatcode mechanism runs on fork-only EVM hooks that upstream has no
equivalent for: `vm.Config.PrecompileOverrides` (cheatcode precompiles), `vm.Config.CallerOverride`
(pranks), `vm.Config.NoMaxCodeSize`, and the exported `state.StateUpdate` without which
`script/forking.ForkDB` cannot implement upstream's `state.Database` (its `Commit` takes an
*unexported* type). "Swap to op-core" does not exist here.

**Decision: rewrite script execution as a Rust engine reusing foundry crates** (forge is the
reference executor for these scripts; revm underneath), consumed by op-deployer — subprocess/
sidecar per the op-reth-test-engine precedent (#20415), or equivalent embedding. Constraints:
op-deployer keeps working without a system-installed foundry (engine version-pinned and shipped
with our tooling); cheatcode surface limited to what our scripts use (derive from the cheatcode
dispatch in `op-chain-ops/script`); parity-gate against the Go engine on reference deployments
before switching.

**No interim module split.** Splitting op-chain-ops+op-deployer into their own Go module that
keeps the op-geth replace was considered and rejected: the epic's value only materialises when we
stop maintaining the op-geth fork entirely — any in-repo module still depending on it keeps the
fork alive. (The same lens applies to the superchain-registry repo's `ops` module, which pins its
own op-geth — outside this repo, flagged to that team.)

This is a hard blocker of #20266 and the longest pole alongside #20415/#21196.

---

## 17. In-process op-geth L2 EL in op-e2e system tests and op-devstack/sysgo

Beyond op-e2e/actions (§ scope fate 2, #21196), op-e2e *system* tests and op-devstack/sysgo can
run op-geth **in-process as the L2 EL**: `op-e2e/e2eutils/geth.InitL2` behind the
`e2eutils/el.InitL2` factory's `ELKindOpGeth`, and `op-devstack/sysgo/l2_el_opgeth.go`. After the
go.mod flip these would silently run *upstream* geth as the L2 EL — wrong OP state transition,
with no compile error to warn us. The in-process **L1** geth miner is unaffected (plain Ethereum
semantics are exactly right for it under upstream) and stays.

Plan: make op-reth the **only** L2 EL kind — delete `ELKindOpGeth`, `geth.InitL2`,
`l2_el_opgeth.go`, and the geth-only knob plumbing (`GethOptions`); resolve each geth-pinned test
case by case (migrate or delete with rationale; derive the pin list by grepping `ELKindOpGeth` /
`GethOption`). The pre-Regolith tests (#21451: op-reth returns `IsSystemTx` as a string on
pre-Regolith blocks) fold into this — likely resolved by deleting them and requiring
bedrock/regolith co-activation. Precedent: #21182 (acceptance tests op-reth-only for Karst+).

---

## 18. Remaining op-chain-ops tools — check-* and op-wheel

**`cmd/check-*` per-fork checkers** verify fork activation on live chains using fork symbols
(`types.DepositTx`, receipt `DepositReceiptVersion`, L1-cost funcs, `types.CalcDAFootprint`, geth
`eip1559` Jovian helpers). Plan: **delete the checkers for long-activated forks** (proposal:
pre-Holocene — team confirms the list on the issue) and **swap the survivors to op-core**, adding
the one missing symbol family with differential tests — DA-footprint calculation
(`types.CalcDAFootprint`) to `op-core/fees`; the eip1559 uses are import swaps onto the existing
`op-core/eip1559` helpers (§8).

**`op-wheel`** splits cleanly: the `engine` commands drive a live EL over Engine-API RPC —
upstream-safe, stay. The `cheat` commands do offline surgery on **op-geth datadirs** (raw
`rawdb`/`core/state`, fork-only `StateDB.OpenStorageTrie`, in-process block processing) — not
upstream-portable, and pointless in an op-reth fleet whose datadirs aren't geth-format. Plan:
**delete `op-wheel/cheat`** (op-simulate precedent, #21282).

---

## 19. CI ratchet — continuously rehearsing the cutover

A scheduled CI job makes decoupling progress measurable and regression-proof: in a throwaway
checkout it drops the go-ethereum `replace`, pins upstream at the current op-geth base version,
runs `go mod tidy` + `go build`/`go vet`, and compares the **set of failing packages** against a
committed baseline. A failure outside the baseline is a regression (a fork-API use crept into a
clean package) and fails the job; a baselined package turning green shrinks the baseline — it
only ever tightens. #20266 becomes executable exactly when the baseline is empty, and the flip PR
deletes the job. The op-core differential tests are expected baseline members until then (they
import op-geth-only symbols by design).

---

## Summary table

| Area | Target | Status |
|------|--------|--------|
| ProtocolVersions mechanism (§0) | deleted end-to-end | **done** |
| `DepositTx` / `PostExecTx` types + helpers (§1, §2) | `op-core/types` | **done** (#20262); call-site swaps open (#20269, #20263) |
| OP `Receipt` (§3) | `op-core/types.Receipt` | **done**; wiring rides §11 (#20264) |
| L1-cost / operator-fee math (§4) | `op-core/fees` | **done** (#20261); op-batcher site rides §11 |
| `OptimismConfig` / standalone `ChainConfig` / `GethChainConfig` (§5) | `op-core/params` | **done** (#20260); swaps + `HardforkConfig` open (#20270) |
| Superchain registry + loaders (§6, §7) | `op-core/superchain` + `op-core/params` | **done** (#20267, #21487) |
| eip1559 Holocene/Jovian helpers (§8) | `op-core/eip1559` | **done** for op-node; `EncodeOptimismExtraData`/`DecodeHoloceneExtraData` pending (§13) |
| `PayloadID`, Engine-API types (§9, §10) | upstream / op-service | no change needed |
| OP-aware eth client: class-partitioned accessors, `optypes.Receipt` returns, op-batcher + txinclude migration (§11) | `op-service/sources` / `apis.EthClient` | open (#20264) |
| op-proposer / op-challenger / op-supernode / cannon (§12) | — | audited, no work |
| Test migration: wait.go split, L2 call sites, `L2Client` type, sysgo audit, delete op-e2e/opgeth (§13) | `apis.EthClient` + header-only variants | open (#20265 + subs) |
| op-e2e/actions in-process EL | `op-reth-test-engine` subprocess | open (#20415 Rust, #21196 Go) |
| Genesis tooling (§14) | upstream geth as library + `opparams` | open (#21281) |
| op-simulate / op-run-block (§14) | delete | **done** (#21282) |
| op-sync-tester PayloadID hash | OP-aware `Id()` reimplementation | open (#21525) |
| Log context extensions (§15) | owned `op-service/log` layer (alias sweep → owned interface) | open |
| RPC recorder hooks + `JsonError` (§15) | client wrappers + server-side interception | open |
| One-off fork symbols (§15) | per-symbol swaps, ride #20263 family | open |
| `op-chain-ops/script` + op-deployer (§16) | **Rust script engine** (foundry crates) | open |
| In-process op-geth L2 EL in system tests + sysgo (§17) | op-reth-only; folds #21451 | open |
| `cmd/check-*` (§18) | delete pre-Holocene; swap survivors to op-core | open |
| `op-wheel/cheat` (§18) | delete (`engine` stays) | **done** (#21747) |
| CI ratchet (§19) | scheduled upstream-build job + tightening baseline | open |
| Final cutover: flip replace, shed `GethChainConfig` OP fields, delete differential tests + §2 scaffolding | go.mod | open (#20266) |
