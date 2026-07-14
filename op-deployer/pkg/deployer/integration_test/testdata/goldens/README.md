# op-deployer integration goldens

These fixtures pin the output of the **in-process Go script host** and are compared against the Rust
engine (the default) by the `*Golden` tests in this package. They were recorded from the Go host at
monorepo commit `c3edeeb2d3` (the `seb/script-engine-forkmode-default` tip) by running each pipeline
with `env.ScriptEngineGo` and the CREATE2 salt pinned.

Once the Go script host is deleted there is nothing left to regenerate these from, so **do not
regenerate them** — a mismatch means the Rust engine diverged from the Go host at the recording
commit. While the Go host still exists, the paired `*Parity` tests re-derive the same outputs live and
assert Rust == Go, proving these fixtures faithful.

Recording (transient; the recorder `zz_record_test.go` is not committed):

    RECORD_GOLDENS=1 go test ./op-deployer/pkg/deployer/integration_test/ -run TestZZRecordGoldens -count=1

| fixture | pins | format | backs |
|---|---|---|---|
| `l2genesis.<mode>.sha256` (5) | L2 genesis dump from a direct engine run | hash | `TestRustEngineL2GenesisParity` |
| `pipeline.l1.<mode>.sha256` (5) | sealed L1 dev-genesis dump from ApplyPipeline | hash | `TestApplyPipelineRustEngineGenesis` |
| `pipeline.l2.<mode>.sha256` (5) | L2 genesis dump from ApplyPipeline | hash | `TestApplyPipelineRustEngineGenesis` |
| `pipeline.fork-<fork>.sha256` (9) | L2 genesis dump per op-e2e genesis fork | hash | `TestApplyPipelineRustEngineGenesisForks` |
| `semvers.json` | `op-deployer inspect l2-semvers` output | raw JSON | `TestRustEngineL2SemversParity` |

Modes are `default`, `l2cm`, `cgt`, `interop`, `cgt+interop` (the `+` is written as `-` in filenames).

The genesis dumps are multi-MB, so they are pinned by the SHA-256 of their canonical `json.Marshal`
bytes rather than committed raw. Determinism: the CREATE2 salt is pinned (`…deadbeef`) so the L1
deploy addresses the L2 genesis embeds are stable across runs; `encoding/json` sorts map keys, so the
serialization is deterministic. The golden tests run structural non-vacuity guards (>2000 L2 accounts,
>50 L1 accounts, non-empty semvers) before each hash compare, and on mismatch write the actual
canonical JSON to a temp file so a divergence is debuggable without the Go host.
