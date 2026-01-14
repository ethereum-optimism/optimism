# `fpvm-prestates`

Images for creating reproducible `kona-client` prestate builds for supported fault proof virtual machines.

## Usage

### All prestate artifacts

```sh
# Produce the prestate artifacts for `kona-client` running on `asterisc` and `cannon`
# (FPVM versions specified by `asterisc_tag` + `cannon_tag`)
just all <kona|kona-int> <kona_tag> <asterisc_tag> <cannon_tag>
```

### `kona-client` + `asterisc` prestate artifacts

```sh
# Produce the prestate artifacts for `kona-client` running on `asterisc` (version specified by `asterisc_tag`)
just asterisc <kona|kona-int> <kona_tag> <asterisc_tag>
```

### `kona-client` + `cannon` prestate artifacts

```sh
# Produce the prestate artifacts for `kona-client` running on `cannon` (version specified by `cannon_tag`)
just cannon <kona|kona-int> <kona_tag> <cannon_tag>
```

### `kona-client` + `cannon` prestate artifacts for custom chains

To create a reproducible kona-client prestate build that supports custom or devnet chain configurations that are not in the superchain-registry:

```sh
# Produce the prestate artifacts for `kona-client` running on `cannon` (version specified by `cannon_tag`)
just cannon <kona|kona-int> <kona_tag> <cannon_tag> <artifacts_output_dir> <custom_config_dir>
```

