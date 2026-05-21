# `fpvm-prestates`

Images for creating reproducible `kona-client` prestate builds for supported fault proof virtual machines.

Cannon is built from the local monorepo source.

## Usage

### Build all reproducible `kona-client` + `cannon` prestate artifacts

```sh
# From the monorepo root
just reproducible-prestate-kona
```

This builds both supported variants:

- `kona-client` into `rust/kona/prestate-artifacts-cannon`
- `kona-client-int` into `rust/kona/prestate-artifacts-cannon-interop`

### Build reproducible prestate artifacts for custom chains

To create a reproducible kona-client prestate build that supports custom or devnet chain configurations that are not in the superchain-registry:

```sh
cd rust
KONA_CUSTOM_CONFIGS_DIR=<custom_config_dir> \
  just build-kona-reproducible-prestate-variant <kona-client|kona-client-int> <artifacts_output_dir>
```
