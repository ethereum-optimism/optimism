# `rust-supernode`

Rust implementation of the OP Stack supernode: a multi-chain consensus-layer host that runs several
OP Stack chains in one process and verifies cross-chain safety in process.

This crate is the skeleton for that rewrite. It currently builds a CLI that prints a greeting and
exits; the operator flags, embedded kona virtual nodes, RPC server, metrics, and interop
verification arrive in later changes. The Go implementation in
[`op-supernode/`](../../op-supernode) remains the production component until then.

## Usage

```bash
cd rust
just build-rust-supernode-debug
target/debug/rust-supernode
```

```console
Hello Rust Supernode
```

`-V` prints the short version, `--version` prints the full build metadata block. Both come from
[`op-version`](../op-version), so a release build reports the injected `GIT_VERSION`, `GIT_COMMIT`,
`GIT_DATE`, and `BUILD_PROFILE`, and a local build reports `0.0.0-dev`.

## Development

The crate is a member of the unified `rust/` Cargo workspace, so the workspace-wide targets cover
it:

```bash
cd rust
just lint       # rustfmt, clippy, rustdoc
just test-unit  # unit and integration tests
```
