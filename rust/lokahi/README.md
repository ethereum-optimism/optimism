# `lokahi`

Rust implementation of the OP Stack supernode: a multi-chain consensus-layer host that runs several
OP Stack chains in one process and verifies cross-chain safety in process.

This crate is the skeleton for that rewrite. It currently builds a CLI that prints a greeting and
exits; the operator flags, embedded kona virtual nodes, RPC server, metrics, and interop
verification arrive in later changes. The Go implementation in
[`op-supernode/`](../../op-supernode) remains the production component until then.

## Usage

```bash
cd rust
just build-lokahi-debug
target/debug/lokahi
```

```console
Hello Lokahi
```

`-V` prints the short version, `--version` prints the full build metadata block. Both come from
[`op-version`](../op-version), so a release build reports the injected `GIT_VERSION`, `GIT_COMMIT`,
`GIT_DATE`, and `BUILD_PROFILE`, and a local build reports `0.0.0-dev`.

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
