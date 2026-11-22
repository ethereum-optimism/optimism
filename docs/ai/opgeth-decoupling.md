# OP Stack code on upstream go-ethereum

The monorepo is a single Go module that depends on upstream go-ethereum, not on the op-geth fork.
go-ethereum knows nothing of the OP Stack: it has no deposit (`0x7E`) or post-exec (`0x7D`)
transaction, no OP receipt fields, no OP hardforks in its `params.ChainConfig` and no OP state
transition. This doc says where the OP Stack code lives instead, and which go-ethereum uses are
safe. Tracking issue of the migration: #20257.

**Compiling is not enough.** go-ethereum code compiles against OP data and then behaves wrongly
or fails at runtime. Ask of every go-ethereum use: *does it execute an OP state transition, or
decode OP transactions?* If it does, it belongs in op-reth (Rust) or `op-core/*`. If it does
not — plain-EVM execution on L1, storage or trie work, header and RLP encoding — go-ethereum is
right.

---

## Package layout

| Package | Content |
|---|---|
| `op-core/types` | `DepositTx`, `PostExecTx` (binary and RPC JSON codecs), `Receipt` (go-ethereum's receipt plus the OP fields), `Receipts` (receipts-root encoding) |
| `op-core/fees` | L1 cost (Bedrock through Fjord), operator fee (Isthmus, Jovian), DA footprint |
| `op-core/params` | `OptimismConfig`, the standalone `ChainConfig` with the OP fork schedule, `GethChainConfig()` |
| `op-core/genesis` | the L2 `Genesis` and its JSON encoding, `ToBlock()` |
| `op-core/eip1559` | Holocene/Jovian extraData helpers |
| `op-core/superchain`, `op-core/superchain/bundle` | registry types and loader; the embedded registry |
| `op-core/forks`, `op-core/predeploys` | fork names, predeploy addresses |

Conventions:

- **Monorepo-best names and shapes.** Inside `op-core/` the `Optimism`/`OPStack` prefix is
  dropped, except for `OptimismConfig` (it pairs with the JSON key `optimism`) and the
  `{Validate,Decode,Encode}OptimismExtraData` helpers (there "Optimism" names the extraData
  format). Only *computed values* and *wire formats* must match other implementations.
- Files that need both the go-ethereum and the op-core package import the latter as
  `optypes`, `opparams`, `opfees`.
- **Wire compatibility.** JSON tags and binary encodings are those of the OP Stack RPC and
  consensus formats (`DepositTx`: `0x7E || RLP(fields)`; `PostExecTx`: `0x7D || data`, no RLP
  envelope).

### Cross-implementation parity

Every OP wire format and hash rule has a second implementation in this repo: `rust/op-alloy`
defines the OP transaction and receipt types that op-reth and kona consume. Both sides serve one
chain, so a value one produces and the other rejects is a consensus split.

- **The specs decide.** Any new or changed encoding, hash or accept-set is checked against
  `rust/op-alloy`, and a disagreement is decided from the [specs](https://specs.optimism.io), not
  from the incumbent.
- **Pin the agreed value in both languages.** A shared golden vector — the same literal asserted
  in the Go and the Rust suite — keeps the pair locked together; `TestPostExecTxHashGoldenVector`
  (`op-core/types/post_exec_tx_test.go`) and `rust/op-alloy/crates/consensus/src/post_exec/tests.rs`
  pin the post-exec tx hash this way. The op-reth L2 block and receipt fixtures in
  `op-service/sources/testdata/l2` pin the RPC wire format and the transactions and receipts
  roots, but only for values op-reth produces.
- **Golden vectors pin the Go values.** The `op-core` tests assert literal encodings, hashes,
  fees and genesis block hashes (the `GoldenVector` tests) that the Rust stack computes too. The
  values for a new fork come from op-reth, not from this code. The only test that compares the Go
  types with another implementation at runtime is the Go/Rust interop-deposits diff
  (`ops/scripts/test-interop-deposits-diff.sh`).

For batcher-controlled bytes (batches, frames, channels) the second implementation is kona's own
decoder, and the rule compares accept-sets over a generated corpus; see
[derivation.md](derivation.md).

---

## OP transactions never live in a go-ethereum transaction

A go-ethereum `*types.Transaction` cannot hold a deposit or post-exec tx: its decoders reject
both types. An L2 block's tx list is `[L1-info deposit (0x7E), …user txs…, optional post-exec
(0x7D)]` (a post-exec tx can end a Lagoon block), so **decoding any L2 block body into
go-ethereum types fails**, and so does `ethclient`'s `BlockByNumber`, `BlockByHash`,
`BlockReceipts`, `TransactionInBlock` and `TransactionByHash` against an L2 endpoint whenever the
result holds an OP tx. The code compiles; only the call fails.

A forbidigo rule in `.golangci.yaml` rejects those five `ethclient.Client` methods in the
whole tree. An L1 read carries `//nolint:forbidigo // L1: <reason>`, and an L2 read that can only
return a user tx carries `//nolint:forbidigo // L2 user tx: <reason>`. The rule matches the
receiver's static type, so a call through an interface or an embedding type is not seen: don't
wrap an L2 read in one.

- Raw bytes decode into `optypes.DepositTx` / `optypes.PostExecTx`. Code never asks "is this a
  deposit?" of a go-ethereum transaction; it checks the type byte of the encoding.
- Code that needs a whole L2 block's transactions works on their binary encodings, as an
  execution payload carries them (`opfees.CalcDAFootprint`, op-sync-tester, op-wheel).
- Never put a deposit into a `*types.Transaction` or `*types.Block`, including in test fixtures:
  L2 block fixtures are execution payloads with raw tx bytes.

### Reading L2 data: `op-service/sources`

The `op-service/sources` clients speak raw JSON-RPC and are the OP-aware client; there is no
"OP ethclient" wrapper. Their `apis.EthClient` accessors partition a block's txs by class:

- `InfoAndUserTxsBy*` — header and the standard Ethereum txs (OP txs excluded), as
  `types.Transactions`.
- `InfoAndDepositsBy*` — deposits as `optypes.DepositTx`.
- `InfoAndFirstDepositBy*` — header and the L1-info deposit only.
- `PayloadBy*` — the block as an execution payload, every tx as raw bytes.
- `HeaderBy*` — headers decode fine from any chain.

There is no accessor for a decoded `0x7D` tx (`PayloadBy*` keeps its raw bytes): a test that
counts the txs of a Lagoon block must allow for one. `InfoAndTxsBy*` and `HeaderAndTxsByHash`
are for L1: `sources.L2Client` rejects an L2 block holding an OP tx there with `ErrOpTxInBody`.

Callers that correlate txs and receipts by index must account for the excluded OP txs:
`optypes.Receipts.UserReceipts()` pairs index-for-index with `InfoAndUserTxs`.

### Receipts

`op-core/types.Receipt` embeds go-ethereum's receipt and adds the complete OP receipt field set
(deposit nonce and receipt version, L1 fee fields, operator fee, DA footprint scalar), with JSON
and consensus codecs. go-ethereum's receipt JSON decoding drops those fields silently, so
anything that reads them — the e2e and acceptance suites — fetches receipts through
`op-service/sources` as `*optypes.Receipt`. A relay of L2 receipts either uses `optypes.Receipt`
or passes the raw JSON through (op-sync-tester).

`optypes.FromGethReceipt` wraps a go-ethereum receipt (L1, or L2 code that needs only standard
fields) with the OP fields unset. It rejects a deposit receipt, whose consensus encoding needs
the deposit nonce and receipt version. `optypes.Receipts.Geth()` is a view for code that reads
only standard fields (logs).

---

## Chain config and genesis

`op-core/params.ChainConfig` is **standalone**: it does not embed go-ethereum's
`params.ChainConfig`. Embedding would promote go-ethereum's OP-unaware EIP-1559 methods
(`ElasticityMultiplier`, `BaseFeeChangeDenominator`, `LatestFork`), which would silently return
L1 values for consensus-relevant fee parameters.

`GethChainConfig()` returns the go-ethereum config with the Ethereum fork schedule derived from
the OP schedule: Shanghai = Canyon, Cancun = Ecotone, Prague = Isthmus, Osaka = Karst (op-geth's
loader lacked Osaka = Karst; the monorepo follows op-reth). It carries nothing else, so
go-ethereum code reading it applies Ethereum rules: it is fit for the genesis state root and
header encoding, not for executing OP blocks or computing OP fees. Chain ID 10 gets OP Mainnet's
legacy Berlin and London blocks.

The JSON encoding of `ChainConfig` is a go-ethereum chain config with the OP keys added: the
go-ethereum config `GethChainConfig()` derives and the OP fields, both embedded in one struct.
The two key sets must stay disjoint (`TestChainConfigJSONKeysDisjoint`): `encoding/json` drops a
key both embedded structs declare. Decoding keeps only the OP fields; an Ethereum fork key that
contradicts the derived schedule is rejected, absent and unknown keys are accepted.

`rollup.Config` carries the full OP hardfork schedule itself (`ChainOpConfig` is an
`*opparams.OptimismConfig`). The `*params.ChainConfig` values in op-node are **L1** chain configs
and are legitimately go-ethereum's. An OP Stack chain is not supported as L1:
`NewL1ChainConfigFromCLI` rejects its chain config.

op-devstack splits the same way: `stack.L1Network.ChainConfig()` returns the go-ethereum config,
`stack.L2Network.ChainConfig()` an `*opparams.ChainConfig`. Its `Optimism` EIP-1559 values are the
genesis values; from Holocene the SystemConfig sets them, so a test that needs the parameters in
effect decodes them from the block's extraData (`eip1559.DecodeOptimismExtraData`).

**Genesis.** `op-core/genesis.Genesis` is the one L2 genesis JSON encoding: the `core.Genesis`
envelope (header fields, alloc) with a `config` in `opparams.ChainConfig`'s encoding; the
`stateHash` key is rejected. `Genesis.ToBlock()` is go-ethereum's genesis block for the same
header fields, alloc and Ethereum fork schedule plus the one OP rule: from Isthmus the header's `WithdrawalsHash` is the
storage root of the `L2ToL1MessagePasser` predeploy instead of the empty withdrawals root.
`op-chain-ops/genesis.NewL2Genesis` builds it; op-deployer computes the L2 genesis state with
forge (`L2Genesis.s.sol`).

---

## Who executes blocks

- **L2: op-reth only.** op-e2e system tests and op-devstack/sysgo run op-reth as an external
  process (`op-e2e/e2eutils/reth.InitL2`, `op-devstack/sysgo/l2_el_opreth.go`), and neither has
  an EL-kind selector. op-e2e action tests and the kona proof tests run an
  `op-reth-test-engine` subprocess. A suite outside this repo that tests its own EL build points
  sysgo at a CLI-superset binary of op-reth with `DEVSTACK_L2EL_OVERRIDE_BINARY`. Never start a
  go-ethereum node as an L2 EL: it runs the Ethereum state transition and nothing fails.
- **L1: go-ethereum in-process is right.** `op-e2e/e2eutils/geth.InitL1` (with its fake-PoS
  driver) and the op-test-sequencer fake-PoS builder use go-ethereum's own block building. The
  action tests' `L1Miner` (`op-e2e/actions/helpers/l1_miner.go`) assembles blocks by hand
  through go-ethereum's `core` functions; it builds no block access lists, so it refuses
  Amsterdam blocks.
- **L1 blobs need Osaka.** go-ethereum's transaction pool accepts only blob sidecars with cell
  proofs, and its block builder includes them only from Osaka on, so a test L1 that carries blob
  transactions runs Osaka. The `L1Miner` includes cell-proof blob transactions on any fork.
- **Scripts: upstream forge.** op-deployer runs all deployment and genesis scripts with forge
  (the Go script host is gone): keyless `forge script` dry runs whose transactions op-deployer
  sends through txmgr, offline forge for genesis-target L1 and L2 genesis, and forge on an
  ephemeral anvil fork for `prepare`/`continue`.
- **No in-repo Go code executes an OP state transition.** `op-sync-tester` proxies a real op-reth
  EL; op-wheel's `engine` commands drive a live EL over the Engine API.

Gaps: no test drives an L2 EL through the Engine API in-process; on op-reth,
`eth_sendRawTransactionConditional` (#24087) and the historical fork-boundary execution rules
(#24088) have no end-to-end test. System tests don't cover pre-Regolith L2 blocks.

---

## Infrastructure seams: `log` and `rpc`

- **`op-service/log`** is the monorepo's logging API. It owns the `Logger` interface (go-ethereum's
  method set without `Write`, with `With`/`New` returning `Logger`, plus `WithContext`,
  `LogAttrs` and the `…Context` methods), its slog-backed implementation, the global logger and
  the package-level `Trace`…`Crit`. A `Logger` is immutable: `WithContext` derives a logger whose
  records default to that context. Handlers and level helpers are re-exported from go-ethereum;
  `op-service/log/geth.go` is the only file importing `go-ethereum/log`, which the `geth-log`
  depguard rule enforces. `ToGeth` converts a `Logger` where a go-ethereum API takes one;
  `SetDefault` also installs it as go-ethereum's global logger.
- **`op-service/jsonrpc`** owns RPC call recording (`Recorder`, `Message`, `Response`,
  `RecordDone`) and the JSON-RPC `Error`. `op-service/client` records calls in its own wrapper
  around the go-ethereum client; subscriptions are not recorded. Code embedding `op-service/rpc`
  counts and times HTTP requests with `WithHTTPRecorder`. The `<ns>_rpc_client_*` metric names
  and labels are pinned by `TestRPCMetricsDescriptors`.

---

## go-ethereum version bumps

A bump can change exported signatures and, more dangerously, semantics. When bumping:

- **Gas.** EVM calls take a `vm.GasBudget` (execution and state gas, EIP-8037) and return the
  leftover budget; `core.GasPool` has two dimensions from Amsterdam. Code that runs plain L1
  calls passes `vm.NewGasBudget(gas, 0)` before Amsterdam.
- **Intrinsic and floor gas** (`core.IntrinsicGas`, `core.FloorDataGas`) take the fork rules.
  `op-batcher/batcher.maxFloorDataGas` sets a batch tx's gas limit to the larger floor under the
  pre-Amsterdam and the Amsterdam rules, so a batch stays valid across L1's Amsterdam activation.
- **Blob schedule.** Upstream has no Osaka or Amsterdam blob-schedule entries: Osaka keeps
  Prague's parameters and Amsterdam the latest BPO's.
- **L1 forks.** A new L1 fork can add header fields, system calls or Engine API versions that the
  hand-built L1 blocks of `L1Miner`, the fake-PoS drivers and op-node's L1 handling must follow.

---

## Downstream repos

Other OP Labs Go repos that imported the monorepo pinned op-geth through their own `replace`.
Decoupling one means **dropping the replace and bumping to the latest go-ethereum and the latest
monorepo commit in one change**: a repo that drops the replace on an old pin stays on a stale
go-ethereum, and an old monorepo pin does not build against a current go-ethereum. Downstream repos
pin monorepo commits (pseudo-versions), not releases, so a fix is available as soon as it is on
`develop`.

- A pseudo-versioned fork pin opts a module out of advisory matching: dropping it can surface
  high-severity advisories against the go-ethereum version the module actually ran. Treat that
  as a reason to bump, not to keep the pin. Some transitive dependencies arrive through libp2p,
  independent of go-ethereum, and need their own bumps.
- An edge into `op-core/superchain/bundle` caps a consumer's monorepo version: a package reaching
  the bundle cannot be built by a downstream module at all (#22678). The packages allowed to reach
  it are the `bundleAllowed` list in `op-core/superchain/bundle/deps_test.go`; downstream modules
  import `op-core/superchain` for the registry types and loader.
- The rules above apply downstream too. Known cases: `infra/op-txproxy` used op-geth's
  `rpc.JsonError` (the monorepo's counterpart is `op-service/jsonrpc.Error`) and `params.TransactionConditional*ErrCode`,
  which need a home; `infra/proxyd` imported `core/types/interoptypes`, whose access-list encoding
  `op-core/interop/messages` carries; `monitorism/op-monitorism` reads L2 blocks with
  `ethclient.BlockByNumber`, which fails on every L2 block and must move to `op-service/sources`.
