# op-geth Decoupling Analysis

This document analyses the dependencies of the optimism monorepo Go services on op-geth–specific
APIs, and proposes decoupling strategies for each. The goal is to depend on upstream go-ethereum
instead of op-geth without opening upstream PRs.

**In scope:** op-node, op-service, op-batcher, op-proposer, op-challenger, op-faucet,
op-supernode, cannon, and the integration / acceptance test suites (op-e2e,
op-acceptance-tests, op-devstack).

**Out of scope:**
- **op-program** (client and host) — depends on op-geth state execution; kona supersedes it
  (in the monorepo at `rust/kona`).
- **op-supervisor** — deprecated, being replaced by op-supernode.

The op-geth diff vs. upstream go-ethereum (currently based on v1.17.2) can be summarised in three
kinds of change:

1. **New standalone types/files** – `DepositTx`, `RollupCostData`, the `superchain/` package,
   eip1559 Holocene/Jovian helpers.
2. **Fields/methods added to existing upstream types** – `Transaction` methods (`IsDepositTx`,
   `SourceHash`, `Mint`, `IsSystemTx`, `RollupCostData`), `Receipt` L1-cost fields, `ChainConfig`
   OP hardfork fields and methods, `PayloadAttributes` extensions.
3. **Config/CLI wiring** – how op-geth starts: not relevant to the monorepo.

---

## Target package layout in the monorepo

All op-geth-specific code will be extracted into `op-core/`, with new packages living directly
under that directory (alongside the existing `op-core/forks/` and `op-core/predeploys/`):

| Source (op-geth) | Destination (monorepo) |
|---|---|
| `core/types/deposit_tx.go`, `receipt_opstack.go` | `op-core/types/` |
| `core/types/rollup_cost.go` (L1 cost + operator-fee math) | `op-core/fees/` |
| `params/config_op.go`, `params/superchain.go` (`OptimismConfig` + `LoadChainConfig` only) | `op-core/params/` |
| `superchain/` package + `sync-superchain.sh` | `op-core/superchain/` |
| `op-service/superutil/` | merged into `op-core/superchain/` |
| `consensus/misc/eip1559/eip1559_optimism.go` | `op-core/eip1559/` |

**Not carried over** — the protocol-versions signalling mechanism is deprecated and will be
removed as step 0 (see §0): `eth/catalyst/superchain.go` (`SuperchainSignal`,
`LogProtocolVersionSupport`), `params.ProtocolVersion*` types and constants,
`params.OPStackSupport`.

(The `NetworkNames` map from `params/superchain.go` is used for L1-chain-name lookups in
`op-node/rollup/types.go` log/error messages — unrelated to protocol versions — and will move to
`op-core/superchain/` alongside the registry data in §7.)

### Naming conventions

Because everything in `op-core/` is implicitly OP-Stack-specific, new types drop the
`Optimism` / `OPStack` prefix:

| Op-geth name | New name in monorepo |
|---|---|
| `types.Receipt` with OP extensions | `op-core/types.Receipt` (imported as `optypes.Receipt`) |
| `types.RollupCostData`, `NewL1CostFuncFjord`, etc. | `op-core/fees.*` (imported as `opfees`) |
| `params.ChainConfig` with OP extensions | `op-core/params.ChainConfig` (imported as `opparams.ChainConfig`) |
| `params.LoadOPStackChainConfig` | `op-core/params.LoadChainConfig` |
| `superutil.LoadOPStackChainConfigFromChainID` | `op-core/superchain.LoadChainConfigFromChainID` |
| `eip1559.ValidateOptimismExtraData` | `op-core/eip1559.ValidateExtraData` |

**Exception:** `params.OptimismConfig` keeps its name. The field is `ChainConfig.Optimism
*OptimismConfig` — the type pairs with the field name, which is load-bearing for JSON wire
format.

Files that need both upstream and monorepo variants use import aliases like `optypes`,
`opparams`:

```go
import (
    "github.com/ethereum/go-ethereum/params"
    opparams "github.com/ethereum-optimism/optimism/op-core/params"
)
```

---

## 0. Remove ProtocolVersions watching from op-node

### Why

The protocol-versions signalling mechanism — op-node watching the L1 `ProtocolVersions` contract,
reporting deltas via metrics, signalling the engine over `engine_signalSuperchainV1`, and halting
the node when outdated — is a **deprecated feature**. Rather than port its op-geth types
(`params.ProtocolVersion*`, `catalyst.SuperchainSignal`, `catalyst.LogProtocolVersionSupport`,
`params.OPStackSupport`) into `op-core`, we remove the feature from op-node entirely. This is the
first step: it shrinks the surface we need to decouple and avoids carrying dead code into
`op-core`.

### Scope

**op-node (delete):**
- `op-node/node/superchain.go` — the entire file (`handleProtocolVersionsUpdate`, `haltMaybe`).
- In `op-node/node/runcfg/runtime_config.go`: remove the `ProtocolVersion` fields on
  `runtimeConfigData`, the `RequiredProtocolVersion()` / `RecommendedProtocolVersion()` methods,
  `RequiredProtocolVersionStorageSlot` / `RecommendedProtocolVersionStorageSlot` constants, and the
  corresponding `ReadStorageAt` calls in `Load`. Keep everything related to the unsafe block
  signer (`P2PSequencerAddress`, `UnsafeBlockSignerAddressSystemConfigStorageSlot`) — P2P gossip
  needs that.
- In `op-node/node/node.go`: remove the `handleProtocolVersionsUpdate(ctx)` call in
  `initRuntimeConfig`; remove the `rollupHalt` field wiring and `halted.Store(true)` flow if it is
  only driven by protocol-versions halting (needs verification).
- In `op-node/service.go`: remove the block (~lines 59-61) that clears
  `ProtocolVersionsAddress` when the load flag is unset; remove the `RollupHalt` option plumbing.
- In `op-node/flags/flags.go`: remove `RollupLoadProtocolVersions` and `RollupHalt` flags and
  their registration.
- In `op-node/rollup/types.go`: remove `Config.ProtocolVersionsAddress` field.
- In `op-node/rollup/superchain.go`: remove `OPStackSupport` var; remove the line that copies
  `ProtocolVersionsAddr` from superchain config into `Config`.
- In `op-node/metrics/metrics.go` + `noop.go`: remove `ReportProtocolVersions` method from the
  `Metricer` interface and implementations; remove `ProtocolVersionDelta` and `ProtocolVersions`
  gauge vectors.

**op-service (delete):**
- `op-service/sources/engine_client.go`: remove `SignalSuperchainV1` method and its
  `catalyst.SuperchainSignal` / `params.ProtocolVersion` usage.

**Unrelated — leave alone** (same "ProtocolVersion" name, different concept):
- `op-service/apis/p2p.go:46` `ProtocolVersion string` — libp2p peer protocol version string.
- `op-node/p2p/rpc_server.go:115` — reads libp2p `"ProtocolVersion"` from peerstore.

### Outcome

After this step, op-node no longer imports `params.ProtocolVersion*` or `eth/catalyst` for
protocol-version signalling. The remaining superchain registry usages (loading chain config,
`NetworkNames`-style lookups) continue to work and are addressed in §§6–7.

---

## 1. `core/types` – Deposit transaction

### Current usage

`types.DepositTx` (new file in op-geth, type `0x7E`) is used in op-node to construct deposit
transactions. The **universal pattern** is:

```go
opaqueTx, err := types.NewTx(&types.DepositTx{
    SourceHash:          source.SourceHash(),
    From:                someAddr,
    To:                  nil,
    Mint:                big.NewInt(0),
    Value:               big.NewInt(0),
    Gas:                 375_000,
    IsSystemTransaction: false,
    Data:                bytecode,
}).MarshalBinary()
```

Every call immediately calls `.MarshalBinary()` and discards the `*types.Transaction`. The result
is always `[]byte` (opaque RLP for the Engine API payload). Locations:

- `op-node/rollup/derive/deposits.go` – `DeriveDeposits` (user deposits from L1 logs)
- `op-node/rollup/derive/deposit_log.go` – `UnmarshalDepositLogEvent` (returns `*types.DepositTx`)
- `op-node/rollup/derive/*_upgrade_transactions.go` – Ecotone, Fjord, Holocene, Isthmus, Jovian,
  Interop (one `types.NewTx(...).MarshalBinary()` per upgrade deposit)
- `op-node/rollup/interop/indexing/attributes.go` – builds then immediately marshals

The only exception is `DecodeInvalidatedBlockTx` (same file), which decodes an RPC-received tx:

```go
var tx types.Transaction
_ = tx.UnmarshalBinary(raw)
tx.Type()  // checks == types.DepositTxType
tx.From()  // op-geth specific – returns DepositTx.From field
tx.Data()  // standard upstream go-ethereum
```

`types.DepositTxType` (the `0x7E` constant) is used in op-node and op-service to identify
transactions by type byte.

### Proposed decoupling

**Define `DepositTx` in `op-core/types/deposit_tx.go`**. The wire format is `0x7E || RLP(struct)`,
matching the spec and what op-geth implements. No dependency on go-ethereum's `TxData` interface.

```go
const DepositTxType = byte(0x7E)

type DepositTx struct {
    SourceHash          common.Hash
    From                common.Address
    To                  *common.Address
    Mint                *big.Int
    Value               *big.Int
    Gas                 uint64
    IsSystemTransaction bool
    Data                []byte
}

func (d *DepositTx) MarshalBinary() ([]byte, error) { /* 0x7E || RLP(d) */ }
func UnmarshalDepositTx(raw []byte) (*DepositTx, error) { /* strip 0x7E, decode RLP */ }
```

**Wire compatibility test**: add a differential test in `op-core/types/deposit_tx_test.go` that
imports op-geth's `types.DepositTx`, serialises identical structs with both implementations, and
asserts byte-for-byte equality. This test will be removed when the op-geth dependency is migrated
to upstream go-ethereum.

For type-checking without op-geth, `tx.Type()` already exists on upstream `*types.Transaction`.
So:

```go
func IsDepositTx(tx *types.Transaction) bool { return tx.Type() == DepositTxType }
```

For `tx.From()` (single call site in `DecodeInvalidatedBlockTx`): replace with
`UnmarshalDepositTx(rawBytes)` and read `.From` directly.

**All call-sites** that do `types.NewTx(&types.DepositTx{...}).MarshalBinary()` become
`op-core/types.DepositTx{...}.MarshalBinary()`. The `*types.Transaction` wrapper is eliminated;
we go straight from struct to `[]byte`.

**`UnmarshalDepositLogEvent`** returns `*types.DepositTx` today; change to return
`*optypes.DepositTx`. `DeriveDeposits` calls `types.NewTx(dep).MarshalBinary()`; replace
with `dep.MarshalBinary()`.

---

## 2. `core/types` – Transaction methods on RPC-received transactions

### Current usage

The following methods on `*types.Transaction` are op-geth additions, called on transactions that
arrive as raw bytes from the Engine API or ethclient RPC:

| Method | Locations | Purpose |
|--------|-----------|---------|
| `IsDepositTx()` | op-node `payload_util.go`, op-service `sources/types.go`, op-batcher `types.go` | Detect deposit type (type byte == 0x7E) |
| `tx.From()` | op-node `interop/indexing/attributes.go:105` | Read `From` field of a deposit tx |
| `tx.Data()` | op-node `payload_util.go` | Get tx calldata – **exists upstream** |
| `tx.Type()` | op-node, op-service | Get tx type byte – **exists upstream** |
| `IsSystemTx()` | op-node `rollup/engine/build_seal.go` | Detect system deposit flag |
| `SourceHash()` | op-node (multiple derive files) | Read deposit source hash |
| `Mint()` | op-node | Read deposit mint amount |
| `RollupCostData()` | op-service `txinclude/`, op-batcher `types.go` | L1 cost estimation |

### Proposed decoupling

- `tx.Type()` and `tx.Data()` are **standard upstream** – no change needed.
- `IsDepositTx()`: replace with the free function `IsDepositTx(tx)` from `op-core/types`.
- `tx.From()` (one call site): replace with `UnmarshalDepositTx(rawBytes).From`.
- `IsSystemTx()`, `SourceHash()`, `Mint()`: call `UnmarshalDepositTx(rawTxBytes)` and read
  fields from the monorepo struct. Wrap in short helper functions.
- `RollupCostData()`: replaced by `opfees.TxRollupCostData(tx)` free function in `op-core/fees`
  (see §4). The computation only requires `tx.Data()` and `tx.Type()`, both upstream.

---

## 3. `core/types` – Receipt L1-cost fields

### Current usage

op-geth adds OP-specific fields to `types.Receipt`. Investigation confirmed they are actively used
in exactly one non-test location: **`op-service/txinclude/txbudget.go::AfterIncluded`**:

```go
receipt := tx.Receipt  // *types.Receipt, fetched via ethclient.TransactionReceipt()

// l1Cost
if receipt.L1BaseFeeScalar != nil {
    l1BaseFeeScalar := new(big.Int).SetUint64(*receipt.L1BaseFeeScalar)
    l1BlobBaseFeeScalar := new(big.Int).SetUint64(*receipt.L1BlobBaseFeeScalar)
    costFunc := types.NewL1CostFuncFjord(receipt.L1GasPrice, receipt.L1BlobBaseFee, ...)
    l1Cost, _ := costFunc(tx.Transaction.RollupCostData())
    actualCost.Add(actualCost, l1Cost)
}
// operatorCost
if receipt.OperatorFeeScalar != nil {
    // uses *receipt.OperatorFeeScalar and *receipt.OperatorFeeConstant
}
```

The receipt travels: `ethclient.TransactionReceipt()` → `EL` interface → `Monitor` → `Persistent`
→ `IncludedTx.Receipt *types.Receipt`. With upstream go-ethereum, the extra fields would be nil
since the standard JSON unmarshaler does not know about them.

Fields used: `L1BaseFeeScalar *uint64`, `L1BlobBaseFeeScalar *uint64`, `L1GasPrice *big.Int`,
`L1BlobBaseFee *big.Int`, `OperatorFeeScalar *uint64`, `OperatorFeeConstant *uint64`.

Note: `op-service/txinclude/isthmus_cost_oracle.go` does **not** fetch receipts. It reads fee
parameters directly from the L1Block predeploy contract via batch `eth_call`. Receipt fields are
not involved there.

Also note: op-node does **not** read receipt fields directly. It reads `L1BlockInfo` from the
deposit transaction calldata in payloads, which encodes the same values.

### Proposed decoupling

**Define `Receipt` in `op-core/types/`**, embedding `types.Receipt` with the extra fields
and a custom JSON unmarshaler:

```go
// in op-core/types, called simply Receipt; consumers import as optypes
type Receipt struct {
    types.Receipt
    L1GasPrice          *big.Int `json:"l1GasPrice,omitempty"`
    L1BlobBaseFee       *big.Int `json:"l1BlobBaseFee,omitempty"`
    L1BaseFeeScalar     *uint64  `json:"l1BaseFeeScalar,omitempty"`
    L1BlobBaseFeeScalar *uint64  `json:"l1BlobBaseFeeScalar,omitempty"`
    OperatorFeeScalar   *uint64  `json:"operatorFeeScalar,omitempty"`
    OperatorFeeConstant *uint64  `json:"operatorFeeConstant,omitempty"`
}
```

The `EL` interface in `txinclude` returns `*optypes.Receipt` instead of `*types.Receipt`.
`IncludedTx.Receipt` becomes `*optypes.Receipt`. This contains all the changes within
`op-service/txinclude/`.

---

## 4. `op-core/fees` – L1 cost and operator-fee math

### Current usage

op-geth's `core/types/rollup_cost.go` provides the L1 cost calculation machinery. The monorepo
uses only the Fjord-era subset:

- `types.RollupCostData` — struct carrying `Zeroes`, `Ones`, `FastLzSize` byte counts
- `types.NewRollupCostData(data []byte) RollupCostData` — compute RCD from tx calldata
- `types.NewL1CostFuncFjord(l1BaseFee, l1BlobBaseFee, baseFeeScalar, blobFeeScalar *big.Int)` —
  returns `func(RollupCostData, blockTime uint64) *big.Int`
- `(RollupCostData).EstimatedDASize()` — used by op-batcher for DA size estimation

Call sites:
- `op-service/txinclude/txbudget.go` — L1 cost calc + **inline operator-fee arithmetic**
- `op-service/txinclude/isthmus_cost_oracle.go` — pre-estimate L1 cost + **same inline
  operator-fee arithmetic** (duplicated from txbudget.go)
- `op-batcher/batcher/types.go` — `tx.RollupCostData()` for DA size estimation

The two `txinclude` files share this snippet verbatim (modulo source fields) for operator fee:

```go
operatorCost := new(big.Int).SetUint64(gasUsed)
operatorCost.Mul(operatorCost, new(big.Int).SetUint64(scalar))
operatorCost = operatorCost.Div(operatorCost, oneMillion)
operatorCost = operatorCost.Add(operatorCost, new(big.Int).SetUint64(constant))
```

Note: `txbudget.go:95` has a TODO noting the Jovian formula will change this (multiplies by 100
instead of dividing by a million). A single shared helper also gives us one place to switch.

### Proposed decoupling

**Create `op-core/fees/`**, consumers import as `opfees`:

```go
package fees // in op-core/fees/

type RollupCostData struct { Zeroes, Ones, FastLzSize uint64 }

func NewRollupCostData(data []byte) RollupCostData { /* copied verbatim */ }
func (r RollupCostData) EstimatedDASize() *big.Int { /* copied verbatim */ }

type L1CostFunc func(RollupCostData, blockTime uint64) *big.Int

func NewL1CostFuncFjord(l1BaseFee, l1BlobBaseFee, baseFeeScalar, blobFeeScalar *big.Int) L1CostFunc

// Replaces inline math duplicated across txbudget.go and isthmus_cost_oracle.go.
// Isthmus formula: (gasUsed * scalar / 1_000_000) + constant.
// Jovian formula will be switched here when activated (see TODO in txbudget.go).
func OperatorCost(gasUsed, scalar, constant uint64) *big.Int

// Replaces the (tx *types.Transaction).RollupCostData() method from op-geth.
func TxRollupCostData(tx *types.Transaction) RollupCostData {
    return NewRollupCostData(tx.Data()) // + any blob/tx-type adjustments per op-geth
}
```

Computation is pure arithmetic on byte counts and `*big.Int`. No dependency on
`op-core/types`, no dependency on go-ethereum beyond `common`, `uint256` and `math/big`.
Graph: `op-core/fees` and `op-core/types` are siblings — no cycle possible.

---

## 5. `params.ChainConfig` – OP hardfork methods

### Current usage

op-geth adds OP hardfork timestamp fields to `params.ChainConfig` (`CanyonTime`, `EcotoneTime`,
`FjordTime`, `GraniteTime`, `HoloceneTime`, `IsthmusTime`, `JovianTime`, `KarstTime`,
`InteropTime`, `BedrockBlock`) and methods like `IsCanyon(t)`, `IsEcotone(t)`, etc.

**Direct calls on `*params.ChainConfig`** (not through op-node's `rollup.Config` wrapper):
- `op-service/eth/types.go:BlockAsPayload` – `config.IsCanyon(t)`, `config.IsIsthmus(t)`

All other hardfork checks in op-node derivation code go through op-node's own `rollup.Config`
type, which has its own `IsCanyon(t)`, `IsHolocene(t)` etc. Those do not touch
`params.ChainConfig`.

`rollup.Config.ChainOpConfig *params.OptimismConfig` carries the OP-specific EIP-1559 parameters.

### Proposed decoupling

**Redefine `OptimismConfig` and an augmented `ChainConfig` in `op-core/params/`**:

```go
// op-core/params/chain_config.go

// OptimismConfig holds OP Stack–specific EIP-1559 parameters.
// Mirrors params.OptimismConfig from op-geth; JSON tags are identical for wire compatibility.
type OptimismConfig struct {
    EIP1559Elasticity        uint64  `json:"eip1559Elasticity"`
    EIP1559Denominator       uint64  `json:"eip1559Denominator"`
    EIP1559DenominatorCanyon *uint64 `json:"eip1559DenominatorCanyon,omitempty"`
}

// ChainConfig wraps upstream params.ChainConfig and adds OP Stack–specific fields.
type ChainConfig struct {
    params.ChainConfig                             // embed upstream
    Optimism       *OptimismConfig `json:"optimism,omitempty"`
    BedrockBlock   *big.Int        `json:"bedrockBlock,omitempty"`
    RegolithTime   *uint64         `json:"regolithTime,omitempty"`
    CanyonTime     *uint64         `json:"canyonTime,omitempty"`
    EcotoneTime    *uint64         `json:"ecotoneTime,omitempty"`
    FjordTime      *uint64         `json:"fjordTime,omitempty"`
    GraniteTime    *uint64         `json:"graniteTime,omitempty"`
    HoloceneTime   *uint64         `json:"holoceneTime,omitempty"`
    IsthmusTime    *uint64         `json:"isthmusTime,omitempty"`
    JovianTime     *uint64         `json:"jovianTime,omitempty"`
    KarstTime      *uint64         `json:"karstTime,omitempty"`
    InteropTime    *uint64         `json:"interopTime,omitempty"`
}

func (c *ChainConfig) IsOptimism() bool  { return c.Optimism != nil }
func (c *ChainConfig) IsCanyon(t uint64) bool  { return isTimestampForked(c.CanyonTime, t) }
// ... etc.
```

**`rollup.Config.ChainOpConfig`** changes type from `*params.OptimismConfig` to
`*opparams.OptimismConfig`. JSON field names are identical so wire format is preserved.

**`BlockAsPayload`** in op-service changes its `config *params.ChainConfig` parameter to a
`HardforkConfig` interface:

```go
type HardforkConfig interface {
    IsCanyon(timestamp uint64) bool
    IsIsthmus(timestamp uint64) bool
}
```

`rollup.Config` already satisfies this interface. This removes the only direct dependency on
`*params.ChainConfig` for hardfork detection in op-service.

---

## 6. `params/superchain.go` – `LoadOPStackChainConfig`

### Current usage

op-geth's `params/superchain.go` provides several things; after §0 only one of them still has
users in the monorepo:

- `LoadOPStackChainConfig(chainCfg *superchain.ChainConfig) (*ChainConfig, error)` —
  used by `op-service/superutil/chain_config.go` and (transitively) `op-node/rollup/superchain.go`.

The `ProtocolVersion*` types, `ProtocolVersionComparison`, `AheadMajor` / `OutdatedMajor`
constants, `OPStackSupport` variable, and `NetworkNames` map in the same op-geth file are **all
removed in §0** and are not carried over.

### Proposed decoupling

**Move to `op-core/params/` as `LoadChainConfig`** (drops the `OPStack` prefix —
redundant inside an `op-core` package), alongside `OptimismConfig` (§5). Post-decoupling it
produces an `*opparams.ChainConfig` from the embedded registry data. Details in §7.

---

## 7. `superchain/` package – entirely op-geth specific

### Current usage

The op-geth `superchain/` package embeds chain configuration data (TOML configs from the
superchain registry, zipped into `superchain-configs.zip`) and provides `GetChain(chainID)`,
`GetSuperchain(network)`, and supporting types. Used in:

- `op-node/rollup/superchain.go` – `superchain.GetChain`, `superchain.GetSuperchain`
- `op-node/chaincfg/chains.go` – `superchain.GetChain`
- `op-service/superutil/chain_config.go` – `superchain.GetChain` + `params.LoadChainConfig`

The embedded data is synced from the `ethereum-optimism/superchain-registry` git repo via
`sync-superchain.sh`, which clones the registry at a pinned commit (`superchain-registry-commit.txt`)
and zips the configs. op-geth does **not** depend on `superchain-registry` as a Go module; the
data is embedded as a raw binary blob at compile time via `//go:embed`.

### Proposed decoupling

**Move the entire `superchain/` package and `sync-superchain.sh` to `op-core/superchain/`**,
verbatim from op-geth. This is a self-contained package with no dependencies on other op-geth
internals (it only imports `BurntSushi/toml`, `klauspost/compress/zstd`, and standard library).

**No new Go module dependency is required**: the data remains embedded exactly as in op-geth. The
sync script is also copied.

**`op-service/superutil/`** is merged into `op-core/superchain/`. Its single function:

```go
// current (op-service/superutil/chain_config.go, against op-geth):
func LoadOPStackChainConfigFromChainID(chainID uint64) (*params.ChainConfig, error) {
    chain, err := superchain.GetChain(chainID)
    // ...
    return params.LoadOPStackChainConfig(chainCfg)
}
```

becomes:

```go
// in op-core/superchain/, against upstream go-ethereum:
func LoadChainConfigFromChainID(chainID uint64) (*opparams.ChainConfig, error) {
    chain, err := GetChain(chainID)   // local, now in op-core/superchain
    // ...
    return opparams.LoadChainConfig(chainCfg)  // now in op-core/params
}
```

**Hardfork schedule in `rollup.Config`** (option a.): `rollup.Config` is extended to load and
carry all OP hardfork timestamps from the registry, rather than going through `*params.ChainConfig`.
`rollup.Config.ChainOpConfig *params.OptimismConfig` becomes
`rollup.Config.ChainOpConfig *opparams.OptimismConfig`. The existing hardfork timestamp fields on
`rollup.Config` already cover most forks; any gaps (KarstTime, InteropTime) are filled in.

---

## 8. `consensus/misc/eip1559` – Holocene/Jovian helpers

### Current usage

op-geth adds `eip1559_optimism.go` with self-contained functions for Holocene/Jovian parameter
encoding. Used in op-node:

- `rollup/derive/payload_util.go` – `EncodeHolocene1559Params`
- `rollup/interop/indexing/attributes.go` – `EncodeHolocene1559Params`, `DecodeJovianExtraData`
- `rollup/attributes/engine_consolidate.go` – `DecodeHolocene1559Params`

Signatures operate on `[]byte` and `uint64` scalars only. No go-ethereum type dependencies.

### Proposed decoupling

**Move to `op-core/eip1559/`**. Copy verbatim; the only import is `errors`.

---

## 9. `beacon/engine` – `PayloadID` type alias

### Current usage

`op-service/eth/types.go` defines `type PayloadID = engine.PayloadID` where `engine.PayloadID` is
`[8]byte`. This is **standard upstream** go-ethereum.

### Proposed decoupling

None needed — keep the `engine.PayloadID` import.

---

## 10. `beacon/engine` – `PayloadAttributes` / `ExecutableData` extensions

### Current status

op-service defines its **own** `PayloadAttributes`, `ExecutionPayload`, and
`ExecutionPayloadEnvelope` types in `op-service/eth/types.go`. These mirror the Engine API types
but are entirely monorepo-defined with the OP-specific extra fields.

The conversion function `BlockAsPayload` accesses only:

```go
bl.Header().WithdrawalsHash   // standard go-ethereum since EIP-4895 / Shanghai
bl.BeaconRoot()               // standard go-ethereum since EIP-4788 / Cancun
```

Both fields exist unchanged in upstream go-ethereum.

### Proposed decoupling

None needed. The `BlockAsPayload` function just needs the `HardforkConfig` interface change
from §5 and will compile against upstream go-ethereum.

---

## 11. `ethclient` — JSON decoding of L2 blocks and receipts

### Problem

`github.com/ethereum/go-ethereum/ethclient` is used in many places in the monorepo, but **most
usages are safe**: they only touch L1, standard receipt fields, or scalar values (chain ID,
balance). The concern is narrower than it looks.

Two failure modes when switching to upstream go-ethereum:

1. **L2 block decoding fails on deposit transactions.** op-geth's `core/types/transaction_marshalling.go`
   has a `case DepositTxType:` arm in `Transaction.UnmarshalJSON` that upstream does not have.
   With upstream go-ethereum, `ethclient.BlockByNumber` / `BlockByHash` / `TransactionByHash`
   against an L2 endpoint will fail to unmarshal deposit txs ("transaction type not supported").
2. **Receipt L1-cost/operator-fee fields are silently dropped.** `ethclient.TransactionReceipt`
   returns `*types.Receipt`. Upstream `types.Receipt` has no `L1GasPrice`, `L1BlobBaseFee`,
   `L1BaseFeeScalar`, `L1BlobBaseFeeScalar`, `OperatorFeeScalar`, `OperatorFeeConstant`. These
   JSON fields exist in the RPC response but are dropped by the unmarshaler. (See §3.)

### Call-site survey (in scope: op-node, op-service, op-batcher)

Files importing `ethclient`, categorised:

**Safe — L1-only or standard fields only** (no change needed):
- `op-node/cmd/genesis/cmd.go`, `op-node/cmd/batch_decoder/*` — L1 `BlockByNumber` / `ChainID`.
- `op-service/txmgr/cli.go` — `ChainID` only.
- `op-service/dial/dial.go` — just the `Dial` wrapper; returned client's use-sites are what matter.
- `op-service/metrics/balance.go`, `op-batcher/metrics/metrics.go` + `noop.go` — `BalanceAt`.
- `op-service/gnosis/client.go` + `integration_test/helpers.go` — Gnosis Safe (L1).
- `op-service/bgpo/oracle.go` — blob tip oracle, operates on L1.
- `op-node/withdrawals/utils.go` — calls `TransactionReceipt` but reads only `receipt.Logs`
  (standard field). Safe.
- `op-service/testutils/devnet/anvil.go` — anvil = L1 simulator, no deposit txs.

**Unsafe — needs migration:**
- `op-batcher/batcher/driver.go` — `l2Client.BlockByNumber(...)` against the L2 endpoint, via
  the `L2Client` interface in `driver.go:77` and `EthClientInterface` in
  `op-service/dial/ethclient_interface.go`. **Will fail on deposit tx decoding under upstream
  go-ethereum.**
- `op-service/txinclude/` — `el.TransactionReceipt` (via the `EL` interface in
  `interfaces.go:44`) returns `*types.Receipt`; fields read in `txbudget.go` require the OP
  extensions (§3).

**op-faucet — just works on upstream:**
- `op-faucet/faucet/backend/faucet.go` — only direct `ethclient` call is `BalanceAt` (L1
  balance lookup, upstream-safe). No deposit-tx decoding, no OP receipt fields, no
  `params.OptimismConfig`, no `params.ProtocolVersion`, no `superchain` imports.
- `op-service/testutils/simulated_eth_client.go` — wraps `ethclient/simulated.Backend`, used
  by `op-faucet/faucet/backend/faucet_test.go`. Only uses
  `simulated.{Backend,Client,NewBackend,WithBlockGasLimit}` — all present in upstream
  go-ethereum.

No migration work needed for op-faucet.

### Proposed decoupling

**Don't create a new "OP ethclient" wrapper package.** Instead:

1. **Reuse and extend `op-service/sources`.** The clients there (`EthClient`, `L1Client`,
   `L2Client`) already do raw JSON-RPC via `client.RPC` — not via `ethclient`. Their `RPCBlock`
   type in `op-service/sources/types.go` already deserialises blocks with `[]*types.Transaction`
   and already calls `IsDepositTx()` on L2 blocks. We add custom JSON unmarshaling there so the
   transactions list round-trips deposit txs (type 0x7E) against upstream go-ethereum: decode each
   entry by inspecting the `"type"` field, routing `0x7e` to `op-core/types.DepositTx` and all
   others to upstream `types.Transaction`.

2. **Migrate `op-batcher/batcher/driver.go` to `op-service/sources.EthClient`.** Its `L2Client`
   interface (`BlockByNumber` returning `*types.Block`) changes to a sources-based accessor that
   returns whatever shape the batcher needs (block info + transaction bytes or typed txs). The
   batcher only uses the block to iterate transactions and filter out deposits for DA
   estimation — it does not need go-ethereum's `*types.Block` specifically. Also change
   `op-service/dial/ethclient_interface.go` so `L2EndpointProvider.EthClient` returns a sources
   client instead of `*ethclient.Client` (or phase that interface out altogether).

3. **Change `op-service/txinclude/EL.TransactionReceipt` return type to `*optypes.Receipt`**
   (from §3). The underlying implementation switches from `ethclient.TransactionReceipt` to a
   raw JSON-RPC call in `op-service/sources` that unmarshals into the extended receipt. Same
   interface pattern as elsewhere in sources.

### Test implications

- **No production test-double depends on op-geth-specific simulated backend APIs.**
  `op-service/testutils/simulated_eth_client.go` (used by `op-faucet/faucet/backend/faucet_test.go`)
  only uses symbols that exist in upstream go-ethereum; it needs no changes.
- Tests that currently use `ethclient.Dial` against real L1 endpoints (anvil, etc.) are
  unaffected: anvil is L1 and has no deposit txs, and op-geth's receipt extensions on L1 don't
  apply.
- Tests that fetch L2 receipts or blocks via `ethclient` (e.g., in `op-e2e/`) are **out of the
  current scope** but will need the same treatment if we later bring them into the decoupling.
  Likely pattern: import the relevant `op-service/sources` client in the tests.
- op-challenger/op-proposer use `ethclient` for L1 balance metrics and L1/L2 header access. The
  L2 header path (`op-challenger/game/client/provider.go`) would have the same deposit-tx
  decoding issue if it fetched full blocks, but headers don't contain transactions so they're
  fine. These components are out of current scope anyway.

### Summary

Answer to "do we need an extended OP-Stack ethclient?": **No.** The deposit-tx decoding issue
only bites in two concrete places (batcher L2 block fetch, txinclude receipt fetch), both already
behind interfaces or trivially migratable to `op-service/sources`. The `op-service/sources`
clients become the canonical "OP-aware ethclient" — we extend their existing raw-RPC decoding
rather than introducing a parallel wrapper.

---

## 12. `op-proposer`, `op-challenger`, `op-supernode`, `cannon` — near-zero op-geth coupling

Audit result: all four are essentially safe to run against upstream go-ethereum with zero or
near-zero changes.

**`op-proposer`** — pure L1 operations. Imports `types.Header` and `ethereum.CallMsg` only;
no `types.Transaction` or `types.Receipt` anywhere. All contract calls go through
`op-service/sources/batching.MultiCaller`. `ethclient.Client` is instantiated but only used for
upstream-compatible L1 methods (`HeaderByNumber`, `CodeAt`, `CallContract`). Zero uses of
deposit txs, OP receipt fields, or OP params/hardfork methods. **No migration needed.**

**`op-challenger`** — ~90% L1, one L2 interaction that is safe.
- The only L2 client access is `op-challenger/game/client/provider.go:24,77` (owns an
  `ethclient.Client` for L2). The `L2HeaderSource` interface
  (`op-challenger/game/fault/trace/utils/local.go:22-24`) exposes only `HeaderByNumber`.
- The sole call site (`op-challenger/game/fault/trace/outputs/provider.go:121`) fetches a
  header for proof generation — **headers don't contain transactions**, so the deposit-tx
  decode issue from §11 doesn't apply.
- L1 work uses `op-service/sources.L1Client` and other sources-based abstractions.
- Zero uses of `BlockByNumber`/`BlockByHash`/`TransactionByHash`, extended `Receipt` fields,
  or `DepositTx`.
- **No migration needed.** If anything in this package ever starts needing L2 block bodies
  (txs), migrate at that point per §11.

**`op-supernode`** — no direct `ethclient` imports. Receipts are obtained via the monorepo's
`FetchReceipts(ctx, blockHash) → (eth.BlockInfo, types.Receipts, error)` interface
(`chain_container/engine_controller/engine_controller.go:28`), backed by `op-service/sources`.
The only fields read from receipts are `receipt.Logs` (standard upstream — see
`supernode/activity/interop/logdb.go:265`, `verification_view.go:59`). No uses of
`DepositTx`, OP receipt fields, `params.OptimismConfig`, `params.ProtocolVersion*`, or the
`superchain` package. **No migration needed** — implementation changes behind
`op-service/sources` (§11) are picked up transparently.

**`cannon`** — pure MIPS FPVM. Imports only `common`, `hexutil`, `log` from go-ethereum. No
`ethclient`, no `types.Block`/`Transaction`/`Receipt`, no `core/types`. The preimage oracle
interface accepts arbitrary bytes — it never decodes OP Stack types. **Zero blockchain-layer
coupling; zero migration needed.**

### Out of scope: `op-program/`, `op-supervisor/`

**op-program** (client and host) depends on op-geth state execution (`core/state`, `core/vm`,
etc.). The replacement lives in-tree at `rust/kona` (client + host), a Rust implementation
of the fault-proof program. **op-supervisor** is deprecated and being replaced by op-supernode
(§12).

---

## 13. Tests: `op-e2e`, `op-acceptance-tests`, `op-devstack`

### Survey

Two different dependency profiles across the test suites:

**op-e2e** — 271 Go files, 45 import `ethclient` directly. Uses `types.Receipt` fields widely.
Central utility: `op-e2e/e2eutils/geth/wait.go` (wait-for-block / wait-for-tx helpers) is
imported by ~26 test files. Test framework `e2esys.SystemConfig` exposes `L1Client` /
`L2Client` as `*ethclient.Client`.

**op-acceptance-tests** — 132 Go files, **0 direct `ethclient` imports**. Everything goes
through the `op-devstack` DSL (`dsl.L2ELNode`, `dsl.Funder`, etc.). Tests ultimately reach an
`apis.EthClient` via `sys.L2EL.Escape().EthClient()`. This suite is already half-decoupled.

**op-devstack** — 157 Go files. The DSL layer (`dsl/`, `presets/`) exposes `apis.EthClient`
(interface defined in `op-service/apis/eth.go`, already in our repo, not go-ethereum's). The
infrastructure layer (`op-devstack/sysgo/`) calls `ethclient.Dial` / `ethclient.NewClient`
directly in ~7 files — mostly for L1 contract interaction (L1 contract deployment,
`OptimismPortal2` game-type setup, faultproof dispute-game setup).

### Concrete patterns and scale

| Pattern | Files (e2e + acceptance) | Risk under upstream go-ethereum |
|---|---|---|
| `ethclient` / `apis.EthClient` against L1 only (balance, nonce, chainID, receipt logs) | ~majority | Safe |
| `ethclient.BlockByNumber` / `InfoAndTxsByNumber` against L2 (full block JSON incl. deposit txs) | unknown, needs audit per file | Deposit-tx decode fails — covered by §11 |
| Read OP-extended receipt fields (`L1Fee`, `L1GasPrice`, `L1BlobBaseFee`, `L1BaseFeeScalar`, `L1BlobBaseFeeScalar`, `OperatorFeeScalar`, `OperatorFeeConstant`, `DepositNonce`) | 21 files (17 e2e + 4 acceptance) | Fields silently drop — covered by §3 |
| Construct `types.DepositTx` or call `IsDepositTx()` / `SourceHash()` / `Mint()` | 7 files | Covered by §1 / §2 |
| `params.ChainConfig` OP hardfork checks (`IsCanyon`, `IsIsthmus`, etc.) | ~10-24 files | Covered by §5 once `HardforkConfig` interface lands, or directly on `opparams.ChainConfig` |
| EIP-1559 Holocene/Jovian extra-data helpers | 2 files (Jovian min-basefee tests) | Covered by §8 |
| `ethclient/simulated.Backend` for L1 simulation | a handful in `op-e2e/opgeth/` and `op-devstack/sysgo/` | Simulated backend APIs are upstream; no change unless tests exercise L2 deposit paths |

### Key observation: op-acceptance-tests already uses the right shape

Because the DSL routes all client access through `apis.EthClient` (monorepo interface) and
returns results via DSL result types that embed `*types.Receipt`, every improvement we make to
the implementation behind `apis.EthClient` automatically benefits all acceptance tests. In
particular:

- Swapping the implementation from `ethclient`-based to `op-service/sources`-based fixes L2
  block decoding transparently.
- Changing the `TransactionReceipt` return type from `*types.Receipt` to
  `*optypes.Receipt` (from §3) ripples through the DSL result types mechanically.

This reinforces the recommendation in §11: **don't introduce a parallel "OP ethclient" type;
make `op-service/sources` the canonical implementation of `apis.EthClient`.**

### Proposed decoupling strategy

1. **`apis.EthClient.TransactionReceipt`**: change return type from `*types.Receipt` to
   `*optypes.Receipt` (from §3). Same for `ReceiptsFetcher.FetchReceipts`.
   Implementations in `op-service/sources` unmarshal the extended fields; all 21 call sites
   that read OP receipt fields keep working.

2. **`apis.EthBlockInfo.InfoAndTxsBy*`**: the current signature returns `types.Transactions`.
   Extend the underlying JSON unmarshal (in `op-service/sources`) to route type `0x7e` to
   `op-core/types.DepositTx` (from §11). Tests that call `IsDepositTx()` on returned
   transactions migrate to `optypes.IsDepositTx(tx)` free function.

3. **`op-e2e/e2eutils/geth/wait.go`**: the central wait helpers (`WaitForBlock`,
   `WaitForBlockToBeSafe`, `WaitForBlockToBeFinalized`, `WaitForTransaction`,
   `WaitUntilTransactionNotFound`, `WaitForL1OriginOnL2`) take `*ethclient.Client`. **None of
   these functions semantically require OP-Stack specifics** — they use only standard
   go-ethereum APIs. The failure mode under upstream is purely transport-level: the
   transitive `BlockByNumber` / `TransactionByHash` calls decode block bodies, which contain
   deposit txs when pointed at an L2 endpoint.

   A majority of callers pass the L2 sequencer/verifier client but discard the returned
   `*types.Block` (`_, err := geth.WaitForBlock(...)` — they only care about the "reached
   height" side effect). So the simpler fix is to **split these helpers into
   header-only versus full-block variants**:
   - `WaitForBlockHeight(num, client)` returning `*types.Header` via `HeaderByNumber`
     (L2-safe — headers don't contain transactions).
   - Keep `WaitForBlock` returning `*types.Block` for the handful of callers that use the
     body (`eip1559params_test.go`, etc.), and migrate it to `apis.EthClient`-backed
     `sources.EthClient` so deposit txs decode.
   - `WaitForTransaction` returns `*types.Receipt` — change to `*optypes.Receipt` per §3;
     callers already reading `receipt.L1Fee` etc. keep working.
   - `WaitUntilTransactionNotFound`'s `TransactionByHash` call would fail if the target tx
     is itself a deposit tx under upstream. Audit shows callers use non-deposit user txs, so
     the simplest patch is to document this and, if needed, switch to a sources-backed path.

4. **`op-e2e/system/e2esys.SystemConfig.L2Client`**: change from `*ethclient.Client` to
   `apis.EthClient` (backed by `sources.EthClient`). L1 can remain `*ethclient.Client` since
   it's upstream-safe.

5. **`op-devstack/sysgo/` direct `ethclient.Dial` calls**: audit each file. Calls on the L1
   endpoint (contract deployment, deposit proxy setup, faultproof dispute-game setup) are
   safe to leave as-is — L1 receipts and blocks don't contain OP-specific encoding. Any call
   on an L2 endpoint needs migration to `sources.EthClient`.

6. **Hardfork checks in tests**: tests that call `chainCfg.IsCanyon(t)` on `*params.ChainConfig`
   migrate to calling the same method on `*opparams.ChainConfig` (from §5). The superchain
   registry loader (`LoadChainConfigFromChainID`) returns `*opparams.ChainConfig`
   post-§7, so most test setups get the new type automatically.

7. **`op-e2e/opgeth/` directory**: this is a tightly-coupled engine-API/op-geth test package
   (block building, fork choice, extra-data validation) — it intentionally exercises op-geth
   internals. **Plan: delete this package as part of the decoupling.** Equivalent tests against
   op-reth will be introduced separately. No migration attempt; just excise with the final
   decoupling. Documented here so future contributors know not to invest in migrating these
   tests.

### Test implications

- No new test doubles are required beyond what §3 and §11 already introduce.
- The simulated backend (`ethclient/simulated.Backend`) uses only upstream-stable APIs in
  current test usage; no migration needed unless we add L2-simulating tests.
- Tests written against the `op-devstack` DSL (all of op-acceptance-tests) pick up decoupling
  improvements for free once the `apis.EthClient` implementation is swapped.
- Tests written directly against `*ethclient.Client` in op-e2e must be migrated in order to:
  (a) decode L2 blocks containing deposit txs, (b) read OP receipt fields. The `wait.go`
  migration (item 3 above) is the biggest single unblock.

### Audit: op-e2e direct L2 block/tx fetches

Deep audit of all `BlockByNumber`, `BlockByHash`, `TransactionByHash` call sites in
`op-e2e/**` (excluding `op-e2e/opgeth/`, which is being removed):

**L2 block/tx fetches — 19 call sites** — these fail under upstream go-ethereum due to
deposit-tx decoding. Concentrated in:
- `op-e2e/actions/proofs/isthmus_fork_test.go` (3 sites)
- `op-e2e/actions/upgrades/ecotone_fork_test.go` (3 sites)
- `op-e2e/actions/helpers/user.go` (2 sites — `CrossLayerUser`)
- `op-e2e/faultproofs/cannon_benchmark_test.go` (1 site)
- `op-e2e/interop/interop_test.go` (1 site)
- `op-e2e/system/bridge/validity_test.go` (1 site)
- `op-e2e/system/conductor/system_adminrpc_test.go` (3 sites)
- `op-e2e/system/fees/` (3 sites across `fees_test.go`, `l1info_test.go`)
- `op-e2e/system/verifier/legacy_pending_test.go` (2 sites)

Migration: point these through `apis.EthClient` / `sources.EthClient`, or where only height
matters, switch to `HeaderByNumber`.

**L1 block/tx fetches — 7 call sites** — safe under upstream go-ethereum. No change needed.
Locations in `e2eutils/disputegame/`, `actions/upgrades/ecotone_fork_test.go`,
`interop/interop_test.go`, `faultproofs/cannon_benchmark_test.go`, `system/fees/l1info_test.go`.

**Ambiguous utility functions — 7 call sites across 2 files**:
- `op-e2e/e2eutils/geth/find.go:32` — `FindBlock(client *ethclient.Client, ...)`, generic.
- `op-e2e/e2eutils/geth/wait.go:44,80,164,176,219,229` — wait helpers (see item 3 above).

These are invoked with both L1 and L2 clients depending on caller. Migration of these
utilities (header-only split + `apis.EthClient` backing) automatically handles both call
patterns.

---

## Summary table

| Area | op-geth source | Target in monorepo | Effort |
|------|---------------|-------------------|--------|
| Remove protocol-versions watching (§0) | n/a | deletion in op-node/op-service | Medium |
| `DepositTx` type + `MarshalBinary` | `core/types/deposit_tx.go` | `op-core/types/` | Medium |
| `DepositTxType` constant | `core/types/deposit_tx.go` | `op-core/types/` | Trivial |
| `IsDepositTx()` free function | `core/types/transaction.go` | `op-core/types/` | Trivial |
| `IsSystemTx()`, `SourceHash()`, `Mint()` helpers | `core/types/transaction.go` | `op-core/types/` | Low |
| `RollupCostData`, `NewRollupCostData`, `EstimatedDASize`, `NewL1CostFuncFjord`, `L1CostFunc` | `core/types/rollup_cost.go` | `op-core/fees/` | Low |
| `OperatorCost(gasUsed, scalar, constant)` — new helper | n/a (deduplicates inline math) | `op-core/fees/` | Trivial |
| `TxRollupCostData(tx)` — replaces method | `core/types/transaction.go` | `op-core/fees/` | Trivial |
| `Receipt` (receipt L1-cost fields) | `core/types/receipt_opstack.go` | `op-core/types/` | Medium |
| `OptimismConfig` struct | `params/config.go` | `op-core/params/` | Trivial |
| `ChainConfig` (wraps upstream ChainConfig) | `params/config.go`, `params/config_op.go` | `op-core/params/` | Medium |
| `LoadChainConfig` | `params/superchain.go` | `op-core/params/` | Medium |
| `superchain/` package (data + loader) | `superchain/` | `op-core/superchain/` | Low (copy) |
| `sync-superchain.sh` | root | `op-core/superchain/` | Trivial |
| `op-service/superutil/` | monorepo | merged into `op-core/superchain/` | Low |
| EIP-1559 Holocene/Jovian helpers | `consensus/misc/eip1559/eip1559_optimism.go` | `op-core/eip1559/` | Trivial |
| `HardforkConfig` interface | n/a | `op-service/eth/` (for `BlockAsPayload`) | Trivial |
| op-batcher L2 block fetch (ethclient → sources) | `op-batcher/batcher/driver.go` | migrate to `op-service/sources.EthClient` | Medium |
| op-service/sources deposit-tx JSON decoding | `op-service/sources/types.go` | custom `RPCBlock.Transactions` unmarshal | Medium |
| `apis.EthClient.TransactionReceipt` return type | `op-service/apis/eth.go` | change to `*optypes.Receipt` | Low |
| `op-e2e/e2eutils/geth/wait.go` — split into header-only + full-block variants | `op-e2e/e2eutils/geth/wait.go` | `HeaderByNumber`-based helpers for height-only callers; migrate block-body callers to `apis.EthClient` | Medium |
| `op-e2e/system/e2esys.SystemConfig.L2Client` type | `op-e2e/system/e2esys/` | change to `apis.EthClient` | Medium |
| op-e2e direct L2 block/tx call sites (19 sites, 9 files) | op-e2e/actions,faultproofs,interop,system | migrate to `apis.EthClient` or header-only variant | Medium |
| `op-devstack/sysgo/` L2 ethclient uses (audit) | `op-devstack/sysgo/*.go` | migrate L2 callers to `sources.EthClient`; L1 stays on `ethclient` | Low-Medium |
| `op-e2e/opgeth/` package | `op-e2e/opgeth/` | **delete as part of decoupling**; op-reth equivalents introduced separately | Trivial |
| `beacon/engine.PayloadID` | upstream | no change needed | None |
| `Header.WithdrawalsHash`, `BeaconRoot()` | upstream since Shanghai/Cancun | no change needed | None |
| `core.FloorDataGas` | upstream (EIP-7623) | no change needed | None |
| `txpool.ErrAlreadyReserved` | upstream | no change needed | None |
| `params.TxGas`, `BlobTxBlobGasPerBlob` | upstream | no change needed | None |

## Implementation notes

### Wire compatibility for `OptimismConfig`

`rollup.Config.ChainOpConfig *params.OptimismConfig` is serialised to JSON (and potentially sent
over the wire between op-node and other services). The new `*opparams.OptimismConfig` must use
identical JSON field names. Current op-geth JSON tags:

```go
EIP1559Elasticity        uint64  `json:"eip1559Elasticity"`
EIP1559Denominator       uint64  `json:"eip1559Denominator"`
EIP1559DenominatorCanyon *uint64 `json:"eip1559DenominatorCanyon,omitempty"`
```

These must be preserved verbatim in `op-core/params.OptimismConfig`.

### DepositTx RLP wire compatibility

The `op-core/types.DepositTx.MarshalBinary` implementation must produce byte-for-byte identical
output to `types.NewTx(&types.DepositTx{...}).MarshalBinary()` in op-geth. This is verified by
the differential test mentioned in §1. The wire format is:

```
0x7E || RLP([sourceHash, from, to, mint, value, gas, isSystemTransaction, data])
```

with `mint` and `to` following the standard RLP optional-pointer encoding.

### Rollup.Config hardfork schedule

`rollup.Config` already carries timestamp fields for all hardforks up through Karst (merged
Feb 2026, PR #19250), plus `InteropTime`. The `op-core/params.ChainConfig` loader
(`LoadChainConfig`) populates `rollup.Config` directly from the superchain registry data,
bypassing `params.ChainConfig`. Any future hardforks will be added to `rollup.Config` directly
rather than via `params.ChainConfig`.
