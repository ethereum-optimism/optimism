# `fpvm-prestates`

Images for creating reproducible `kona-client` prestate builds for supported fault proof virtual machines.

Cannon is built from the local monorepo source.

## Usage

### `kona-client` + `cannon` prestate artifacts

```sh
# Produce the prestate artifacts for `kona-client` running on `cannon` (built from local monorepo source)
just cannon <kona|kona-int>
```

### `kona-client` + `cannon` prestate artifacts with custom output directory

```sh
just cannon <kona|kona-int> <artifacts_output_dir>
```

### `kona-client` + `cannon` prestate artifacts for custom chains

To create a reproducible kona-client prestate build that supports custom or devnet chain configurations that are not in the superchain-registry:

```sh
just cannon <kona|kona-int> <artifacts_output_dir> <custom_config_dir>
```
