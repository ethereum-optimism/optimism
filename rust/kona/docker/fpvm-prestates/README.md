# `fpvm-prestates`

Composed melange + apko configs for building reproducible `kona-client`
prestates for the cannon fault proof VM.

The build is split into four melange packages so each component's recipe
uses stock `uses:` pipelines and any one component can be rebuilt without
recompiling the others:

| Package | Pipeline | Purpose |
| --- | --- | --- |
| `rust-nightly-2026-02-20` | `uses: fetch` | Pinned Rust nightly toolchain + `rust-src` (hash-pinned tarballs from `static.rust-lang.org`) |
| `cannon` | `uses: go/build` (x2) | MIPS64 FPVM emulator. `cannon64-impl` is built first and embedded into `multicannon`. |
| `kona-client-elf` | `uses: cargo/build-no-auditable` (x2) | `kona-client` and `kona-client-int` MIPS64 ELFs |
| `kona-prestates` | composed; depends on `cannon` + `kona-client-elf` | Runs cannon over each ELF to generate `prestate.bin.gz` + `prestate-proof.json` |

apko then composes the final image from `kona-prestates` alone.

`cargo/build-no-auditable` is a local pipeline definition under
`melange-pipelines/cargo/`. It vendors stock `cargo/build` and swaps
`cargo auditable build` for plain `cargo build`. cargo-auditable panics
on `-Zjson-target-spec` builds, and the resulting MIPS64 no_std binary
could not carry the auditable section anyway. See the file for details.

## Prerequisites

- `melange` and `apko`
- `bubblewrap` (Linux) or `docker` (macOS) for the melange runner
- `jq` and `rsync`

## Usage

Generate both `kona-client` and `kona-client-int` prestates into the
existing checked-in output directories:

```sh
cd ../../..
just build-kona-reproducible-prestate
```

Build a single variant into a custom output dir:

```sh
cd ../../..
just build-kona-reproducible-prestate-variant <kona-client|kona-client-int> <output_dir>
```

Build a prestate that bakes in custom chain configs:

```sh
cd ../../..
KONA_CUSTOM_CONFIGS_DIR=<custom_config_dir> just build-kona-reproducible-prestate
```
