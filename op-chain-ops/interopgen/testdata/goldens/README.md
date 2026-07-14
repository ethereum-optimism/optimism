# Interopgen goldens

These fixtures pin the output of the **in-process Go script host** and are compared against the Rust
engine by `TestRustEngineInteropgenGolden`. They were recorded from the Go host at monorepo commit
`c3edeeb2d3` (the `seb/script-engine-forkmode-default` tip) by running the interopgen world deployment
with `env.ScriptEngineGo`.

Once the Go script host is deleted there is nothing left to regenerate these from, so **do not
regenerate them** — a mismatch means the Rust engine diverged from the Go host at the recording
commit. While the Go host still exists, `TestRustEngineInteropgenParity` re-derives the same outputs
live and asserts Rust == Go, proving these fixtures faithful.

Recording (transient; the recorder `zz_record_test.go` is not committed):

    RECORD_GOLDENS=1 go test ./op-chain-ops/interopgen/ -run TestZZRecordGoldens -count=1

| fixture | pins | format |
|---|---|---|
| `world_deployment.json` | every deployed contract address (L1 + both L2s) | raw JSON |
| `rollup-l2-900200.json`, `rollup-l2-900201.json` | each L2's rollup config | raw JSON |
| `genesis.l1.sha256` | SHA-256 of `json.Marshal(L1 genesis)` | hash |
| `genesis.l2-900200.sha256`, `genesis.l2-900201.sha256` | SHA-256 of `json.Marshal(L2 genesis)` | hash |

The genesis dumps are multi-MB, so they are pinned by the SHA-256 of their canonical `json.Marshal`
bytes (encoding/json sorts map keys, so the serialization is deterministic — verified by recording
twice and diffing) rather than committed raw. The golden test runs structural non-vacuity guards
(non-empty alloc, expected L2 count) before each hash compare, and on mismatch writes the actual
canonical JSON to a temp file so a divergence is debuggable without the Go host.
