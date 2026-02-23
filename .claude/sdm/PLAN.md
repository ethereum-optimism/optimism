# SDM Benchmark Implementation Plan

## Context

SDM PoC1 implements a wall-clock based `opgas` model where each transaction receives an `OPGasRefund` (stored in `OPContainer` per block). The key concern from POC1_REVIEW.md: SSTORE-heavy transactions may execute fast in wall-clock and incorrectly receive large refunds, making state growth cheaper.

This benchmark validates that empirically by deploying adversarial and benign contracts, sending batches of transactions, and comparing `GasUsed` vs `OPGasRefund` across 4 categories.

---

## Files to Create

### 1. `op-acceptance-tests/tests/sdm/bench_test.go`

Single test function `TestSDMBenchmark(gt *testing.T)`.

**Structure:**

```
TestSDMBenchmark
├── Setup: devtest.SerialT, presets.NewSingleChainMultiNode (reuses init_test.go)
├── Fund alice (eth.OneEther) and bob (eth.ZeroWei)
├── Deploy contracts:
│   ├── ComputeHeavy (raw bytecode, keccak256 loop)
│   ├── StateBloat (raw bytecode, SSTORE loop)
│   └── EventLogger (alice.DeployEventLogger())
├── Open output file (env SDM_BENCH_OUTPUT or /tmp/sdm_bench.jsonl)
├── For each category × txCount:
│   ├── Send tx (Transfer / Transact with calldata)
│   ├── Get receipt: ptx.Included.Value()
│   ├── Get payload: sys.L2EL.PayloadByNumber(blockNum)
│   ├── Lookup refund: iterate OPContainer.MetadataOPGas matching entry.Index == txIndex
│   └── Record {category, gasUsed, opGasRefund, refundRatio}
├── Compute summaries: mean, p50, p95, p99 per category
└── Write JSONL records to output file
```

**Contract bytecodes** — compiled from Solidity, embedded as `const` hex strings:

- `computeHeavyBin`: `run(uint256 n)` loops keccak256 n times (pure computation)
- `stateBloatBin`: `run(uint256 n)` writes n unique SSTORE slots (state growth)

Deploy pattern (matches `DeployEventLogger` at `op-devstack/dsl/eoa.go:233`):
```go
tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(common.FromHex(bytecodeHex)))
res, err := tx.Included.Eval(t.Ctx())
addr := res.ContractAddress
```

**Calldata encoding** — use `w3.MustNewFunc` (available in go.mod, used in `op-service/txintent/interop_call.go:35`):
```go
var funcRun = w3.MustNewFunc("run(uint256)", "")
var funcEmitLog = w3.MustNewFunc("emitLog(bytes32[],bytes)", "")
```

**Transaction sending** — sequential, one per iteration:
- `eoa_transfer`: `alice.Transfer(bob.Address(), eth.OneHundredthEther)`
- `compute_heavy`: `alice.Transact(alice.Plan(), txplan.WithTo(&addr), txplan.WithData(encodeRun(500)))`
- `event_emitter`: `alice.Transact(alice.Plan(), txplan.WithTo(&addr), txplan.WithData(encodeEmitLog(4, 100)))`
- `state_bloat`: `alice.Transact(alice.Plan(), txplan.WithTo(&addr), txplan.WithData(encodeRun(50)))`

**OPContainer lookup** — manual iteration (no `GasRefundForIdx` helper in this repo):
```go
func lookupRefund(payload *eth.ExecutionPayloadEnvelope, txIndex uint) uint64 {
    container := payload.ExecutionPayload.OPContainer
    if container == nil { return 0 }
    for _, entry := range container.MetadataOPGas {
        if entry.Index == uint64(txIndex) { return entry.OPGasRefund }
    }
    return 0
}
```

**JSONL output** — write to file via `json.NewEncoder(f)`, not stdout (avoids mixing with test framework output):
```jsonl
{"type":"run_config","tx_count":10,"categories":[...],"op_container_present":true}
{"type":"tx","category":"compute_heavy","block":43,"tx_index":0,"canonical_gas":150000,"op_gas_refund":8000,"effective_op_gas":142000,"refund_ratio":0.053}
{"type":"summary","category":"state_bloat","count":10,"mean_canonical":220000,"mean_effective":42000,"mean_ratio":0.79,"p50_ratio":0.75,"p95_ratio":0.90}
```

**Percentile computation** — sort + linear interpolation:
```go
func percentile(sorted []float64, p float64) float64 {
    idx := p / 100.0 * float64(len(sorted)-1)
    lower, upper := int(math.Floor(idx)), int(math.Ceil(idx))
    if lower == upper { return sorted[lower] }
    weight := idx - float64(lower)
    return sorted[lower]*(1-weight) + sorted[upper]*weight
}
```

**Key imports** (all available in root go.mod):
- `github.com/ethereum-optimism/optimism/op-devstack/devtest`
- `github.com/ethereum-optimism/optimism/op-devstack/dsl`
- `github.com/ethereum-optimism/optimism/op-devstack/presets`
- `github.com/ethereum-optimism/optimism/op-service/eth`
- `github.com/ethereum-optimism/optimism/op-service/txplan`
- `github.com/ethereum/go-ethereum/common`
- `github.com/ethereum/go-ethereum/core/types`
- `github.com/lmittmann/w3`

### 2. `op-acceptance-tests/tests/sdm/visualize.py`

Reads JSONL from `--input FILE` (or stdin). Produces 3 subplots in a single PNG:

1. **Bar chart**: mean refund ratio by category
2. **Grouped bar chart**: mean canonical gas vs mean effective gas
3. **Box plot**: refund ratio distribution per category

Dependencies: `matplotlib` + stdlib (`json`, `argparse`, `sys`).

---

## Critical Files (read-only references)

| File | Why |
|---|---|
| `op-acceptance-tests/tests/sdm/init_test.go` | Reused preset (SingleChainMultiNode, batcher stopped) |
| `op-devstack/dsl/eoa.go:140-191` | `Plan()`, `Transact()`, `Transfer()` APIs |
| `op-devstack/dsl/eoa.go:233-239` | `DeployEventLogger()` pattern for raw bytecode deployment |
| `op-devstack/dsl/l2_el.go:254-258` | `PayloadByNumber()` for reading OPContainer |
| `op-service/eth/types.go:272` | `ExecutionPayload.OPContainer` field |
| `op-service/eth/ssz.go:64-68` | `OPGasEntry{Index, OPGasRefund}` layout |
| `op-service/txintent/interop_call.go:35` | `w3.MustNewFunc` calldata encoding pattern |

---

## Verification

```bash
cd op-acceptance-tests

# 1. Run benchmark
SDM_BENCH_OUTPUT=bench.jsonl go test ./tests/sdm/ -run TestSDMBenchmark -v -count=1 -timeout 5m

# 2. Spot-check
grep '"type":"summary"' bench.jsonl

# 3. Visualize
python3 tests/sdm/visualize.py --input bench.jsonl --output report.png
open report.png
```

Expected: `state_bloat` has higher refund ratio than `compute_heavy` (confirming the wall-clock concern). `compute_heavy` has low refund ratio (slow = no discount). Charts show clearly separated bars per category.
