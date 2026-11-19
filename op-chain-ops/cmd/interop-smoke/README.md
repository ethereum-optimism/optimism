# interop-smoke

Interop smoke tests that run against the RPCs of two live interoperable L2 chains. Each
subcommand takes `--l2a-rpc`, `--l2b-rpc` and an optional `--private-key` for a funded
account on both chains. Run `interop-smoke --help` for the list.

## invalid-spam

`invalid-spam` is a load generator for interop reorg stress tests. For `--duration`, it
sends invalid executing messages back to back on both chains (or one direction with
`--direction`). Each direction reuses one initiating message and points every executing
message at a log index that does not exist, so every block that includes one has to be
invalidated by the supernodes. It does not wait for reorgs. Use `invalid-message` or
`chained-invalid-message` to check that individual invalid transactions are reorged out,
and `interop-safety-check` to check that none reach cross-safe.

It only lands transactions when the sequencers' interop filter accepts invalid messages,
for example on a devnet that runs the permissive filter mock.

```sh
go run ./op-chain-ops/cmd/interop-smoke invalid-spam \
  --l2a-rpc https://chain-a.example --l2b-rpc https://chain-b.example \
  --private-key 0x... --duration 30m
```
