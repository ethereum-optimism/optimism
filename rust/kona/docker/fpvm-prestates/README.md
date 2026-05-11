# `fpvm-prestates`

melange and apko configuration for creating reproducible `kona-client` prestate builds for supported fault proof virtual machines.

Cannon is built from the local monorepo source inside the melange package pipeline. apko then composes a minimal image from the locally built `kona-prestates` APK, and the build script extracts the prestate artifacts back into the existing `rust/kona/prestate-artifacts-*` directories.

## Prerequisites

- `melange`
- `apko`
- `jq`
- `rsync`

## Usage

### `kona-client` + `cannon` prestate artifacts

```sh
# Produce the prestate artifacts for `kona-client` running on `cannon` (built from local monorepo source)
cd ../../..
just build-kona-reproducible-prestate
```

### `kona-client` + `cannon` prestate artifacts with custom output directory

```sh
cd ../../..
just build-kona-reproducible-prestate-variant <kona-client|kona-client-int> <artifacts_output_dir>
```

### `kona-client` + `cannon` prestate artifacts for custom chains

To create a reproducible kona-client prestate build that supports custom or devnet chain configurations that are not in the superchain-registry:

```sh
cd ../../..
KONA_CUSTOM_CONFIGS_DIR=<custom_config_dir> just build-kona-reproducible-prestate
```
