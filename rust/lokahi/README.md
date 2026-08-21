# `lokahi`

Rust implementation of the OP Stack supernode: a multi-chain consensus-layer host that runs several
OP Stack chains in one process and verifies cross-chain safety in process.

Each chain runs the same actor group a single-chain
[`kona-node`](../kona/bin/node) runs, composed through the shared composition entry point in
`kona-node-service`, so the two hosts cannot drift. One L1 watcher serves every chain. Cross-safe
promotion is still the trivial one — cross-safe follows local-safe — so today this is N independent
kona-nodes in one process; interop consolidation replaces that promotion in a later phase. The Go
implementation in [`op-supernode/`](../../op-supernode) remains the production component until then.

## Usage

```bash
cd rust
just build-lokahi-debug
target/debug/lokahi node --config /etc/lokahi/config.toml
```

`-V` prints the short version, `--version` prints the full build metadata block. Both come from
[`op-version`](../op-version), so a release build reports the injected `GIT_VERSION`, `GIT_COMMIT`,
`GIT_DATE`, and `BUILD_PROFILE`, and a local build reports `0.0.0-dev`.

## Configuration

Everything about the chains comes from one file, not from flags: a flag per chain per setting is not
a usable interface at N chains. The file is layered — `[l1]` and `[defaults]` state what the chains
share, and each `[[chains]]` entry states only what is specific to that chain. See
[`config.example.toml`](config.example.toml), which the test suite parses so it cannot drift from
what the code accepts.

A chain's own value always wins over `[defaults]`. What a chain must end up with, from either layer:
`l2-chain-id`, `engine-rpc`, `jwt-secret`, `p2p-tcp-port` and `p2p-udp-port`. The rollup config, the
L1 chain config and the unsafe block signer come from the superchain registry unless the file names
them — a devnet chain is not in the registry, so it names all three.

Each chain gets its own state: `<datadir>/<l2-chain-id>` holds its P2P identity, its bootstore and
its persisted admin-API state. A shared `datadir` in `[defaults]` is split per chain automatically; a
chain that names its own `datadir` gets exactly that directory.

The chain set is fixed at startup — there is no path that adds a chain to a running supernode — and
the configuration is checked before any actor exists. The same chain listed twice, two chains
sharing a P2P port, two chains sharing a data directory, or chains on different L1s are startup
errors naming the chains involved.

## Addressing

One process, one socket. `[admin] rpc-port` is the whole address of the supernode: the process-wide
namespaces answer at `/`, and each hosted chain's node RPC answers at `/<l2-chain-id>` with the
method names a single-chain node has. So a caller that knows where the supernode is can reach any
chain it hosts, and `lokahi_chains` lists the routes.

```
http://host:9500/          supernode_syncStatus, superroot_atTimestamp, lokahi_chains
http://host:9500/901       optimism_syncStatus, optimism_rollupConfig, opp2p_*, admin_* …
http://host:9500/902       the same, for chain 902
http://host:9500/901/healthz
```

This is `op-supernode`'s addressing, segment for segment
([`resources/rpc_router.go`](../../op-supernode/supernode/resources/rpc_router.go)), so a client is
pointed at either implementation with the same URL and no branch — including the refusals: a chain
id this process does not host is a `404`, and a chain it hosts but has not composed yet is waited
for and then a `503`.

## Failure behaviour

A fatal actor error anywhere stops the process, rather than leaving a supernode that serves N-1
chains and still answers for the dead one. Transient conditions — an execution layer that is down,
an L1 RPC that times out — are retried inside the actors and do not stop the node.

## Container image

```bash
docker pull us-docker.pkg.dev/oplabs-tools-artifacts/images/lokahi:develop
```

`build-images.apko` publishes the image for amd64 and arm64 on every `develop` push, and under the
release version for a `lokahi/vX.Y.Z` tag. The build has two stages, the same as every other
Rust image:

- [`melange/op-stack-rust.yaml`](../../melange/op-stack-rust.yaml) compiles the workspace once and
  packages this binary as the `lokahi` APK.
- [`apko/lokahi.yaml`](../../apko/lokahi.yaml) assembles that APK on a Wolfi base
  into the final image.

The image runs as the unprivileged `nonroot` user and its entrypoint is
`/usr/local/bin/lokahi`.

## Development

The crate is a member of the unified `rust/` Cargo workspace, so the workspace-wide targets cover
it:

```bash
cd rust
just lint       # rustfmt, clippy, rustdoc
just test-unit  # unit and integration tests
```
