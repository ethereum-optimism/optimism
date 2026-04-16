# ci-base-clang

This directory contains a small helper image for CircleCI Docker jobs that need `clang`, `llvm-dev`, and `libclang-dev` available up front.

The main use case is Rust jobs that pull in `bindgen` via crates such as `reth-mdbx-sys`. Instead of running `apt-get install clang` on every single CI job, we can bake those packages into the image once and refresh it only when needed.

## Base image

The image is based on:

- `cimg/base:2026.03`

And adds:

- `clang`
- `llvm-dev`
- `libclang-dev`

## Files

- `Dockerfile` — image definition
- `build.sh` — local build helper
- `publish.sh` — GHCR publish helper

## Build locally

Build and load a local amd64 image, then verify the installed toolchain:

```bash
./ops/docker/ci-base-clang/build.sh
```

Build a different tag:

```bash
./ops/docker/ci-base-clang/build.sh --tag 2026.03-clang-dev
```

Build without refreshing the upstream base image:

```bash
./ops/docker/ci-base-clang/build.sh --no-pull
```

## Publish to GHCR

Authenticate with a token that can publish GitHub Container Registry packages:

```bash
export GHCR_USERNAME="<github-username>"
export GHCR_TOKEN="<token-with-write:packages>"
```

Then publish:

```bash
./ops/docker/ci-base-clang/publish.sh --tag 2026.03-clang
```

Publish both a stable tag and a dated refresh tag:

```bash
./ops/docker/ci-base-clang/publish.sh \
  --tag 2026.03-clang \
  --tag 2026.03-clang-$(date -u +%Y%m%d)
```

By default, `publish.sh` builds a multi-platform image for:

- `linux/amd64`
- `linux/arm64`

Override that if you only want one platform:

```bash
./ops/docker/ci-base-clang/publish.sh --platforms linux/amd64 --tag 2026.03-clang
```

## Verification

Both helper scripts verify that the image contains the expected packages by checking:

- `clang --version`
- `llvm-config --version`
- `dpkg-query -W clang llvm-dev libclang-dev`

## Recommended refresh cadence

Rebuild and republish this image when one of these changes:

- the upstream CircleCI base image tag (`cimg/base:2026.03`)
- the desired LLVM/clang packages
- a security or dependency refresh is needed

## Wiring it into CircleCI

Once the image is published, you can test it in CircleCI by overriding the Docker image parameter, for example:

- `default_docker_image = ghcr.io/ethereum-optimism/ci-base:2026.03-clang`

If you want to scope it only to Rust jobs, a follow-up change can add a dedicated CircleCI parameter such as `rust_docker_image` and point only the `needs_clang: true` jobs at this image.

## Notes

- If the GHCR package is private, make sure CircleCI has permission to pull it.
- The helper scripts use `docker buildx`.
- The Dockerfile intentionally stays minimal so refreshes are cheap and easy to reason about.
