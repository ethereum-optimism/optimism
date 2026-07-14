# Rust script-engine goldens

These fixtures pin the output of the **in-process Go script host** and are compared against the Rust
engine by the `*Golden` tests in this package. They were recorded from the Go host at monorepo commit
`c3edeeb2d3` (the `seb/script-engine-forkmode-default` tip) by running each test's Go leg and
serializing its output.

Once the Go script host is deleted there is nothing left to regenerate these from, so **do not
regenerate them** — a `*Golden` mismatch means the Rust engine diverged from the behavior the Go host
had at the recording commit, not that the fixture is stale. While the Go host still exists, the paired
`*Parity` tests re-derive the same outputs live and assert Rust == Go, which is what proves these
fixtures faithfully capture the Go host.

Recording (transient; the recorder file `zz_record_test.go` is not committed):

    RECORD_GOLDENS=1 go test ./op-chain-ops/script/rustengine/ -run TestZZRecordGoldens -count=1

| fixture | pins | source Go leg |
|---|---|---|
| `scriptexample.dump{1,A,B}.json` | ScriptExample state dumps after deploy / call A / call B | `TestRustEngineParity/stateDump` |
| `scriptexample.broadcasts.json` | the 7 recorded broadcasts of `runBroadcast()` | `TestRustEngineParity/broadcast` |
| `scriptexample.nonces.json` | final nonces of the 4 broadcast participants | `TestRustEngineParity/broadcast` |
| `setbalance.dump.json` | state dump after `SetBalance(7e18)` | `TestRustEngineSetBalance` |
| `opcm.single.{output,dump}.json` | OPCM `RunScriptSingle` output struct + state dump | `TestRustEngineOPCMParity/single` |
| `opcm.void.dump.json` | OPCM `RunScriptVoid` state dump | `TestRustEngineOPCMParity/void` |
| `fork.version.txt` | DisputeGameFactory `version()` read through the fork | `TestRustEngineForkedParity` |
| `fork.diff.json` | scaffolding-pruned fork overlay diff | `TestRustEngineForkedParity` |
| `fork.broadcasts.json` | non-isolated SetDisputeGameImpl broadcast bundle | `TestRustEngineForkedParity` |
| `fork.isolated.broadcasts.json` | isolated broadcast bundle (higher gasUsed) | `TestRustEngineForkedIsolatedParity` |

The fork goldens replay the committed RPC fixture `../fork/setdisputegameimpl-sepolia.json`, so they
are hermetic (no network/secret/anvil), same as the parity tests.
