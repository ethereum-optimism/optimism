# `docker`

This directory contains all of the repositories' dockerfiles as well as the [bake file](https://docs.docker.com/build/bake/)
used to define this repository's docker build configuration. In addition, the [recipes](./recipes) directory contains
example deployment strategies + grafana dashboards for applications such as [`kona-node`](../bin/node).

## Install Dependencies

* `docker`: https://www.docker.com/get-started/
* `docker-buildx`: https://github.com/docker/buildx?tab=readme-ov-file#installing

## Building Locally

To build any image in the bake file locally, use `docker buildx bake`:

```sh
# The target is one of the available bake targets within the `docker-bake.hcl`.
# A list can be viewed by running `docker buildx bake --list-targets`
export TARGET="<target_name>"

(cd "$(git rev-parse --show-toplevel)" && docker buildx bake \
  --progress plain \
  -f docker/docker-bake.hcl \
  $TARGET)
```

### Build Options

Relevant build options (variables) for each target can be viewed by running `docker buildx bake --list-variables` or
manually inspecting the targets in the `docker-bake.hcl`.

#### Troubleshooting

If you receive an error like the following:

```
ERROR: Multi-platform build is not supported for the docker driver.
Switch to a different driver, or turn on the containerd image store, and try again.
Learn more at https://docs.docker.com/go/build-multi-platform/
```

Create and activate a new builder and retry the bake command.

```sh
docker buildx create --name kona-builder --use
```

## Nightly Builds

Nightly Docker images are automatically built and published every day at 2 AM UTC for:
- `kona-node`
- `kona-host`

### Using Nightly Images

```sh
# Pull the latest nightly build (multi-platform: linux/amd64, linux/arm64)
docker pull us-docker.pkg.dev/oplabs-tools-artifacts/images/kona-node:nightly
docker pull us-docker.pkg.dev/oplabs-tools-artifacts/images/kona-host:nightly

# Pull a specific date's nightly build
docker pull us-docker.pkg.dev/oplabs-tools-artifacts/images/kona-node:nightly-2024-12-10
```

### Manual Trigger

To manually trigger a nightly build:
```sh
gh workflow run "Build and Publish Nightly Docker Images"
```

## Building Kona Prestates

### Reproducible Build (Docker — recommended for releases)

```bash
# From repo root
just reproducible-prestate-kona
```

### Native Build (Linux — for development)

#### Prerequisites

Managed by mise (`mise install` from repo root): rustup, stable Rust, the pinned
dated nightly, Go, just, jq. Both toolchains come from `mise.toml`.

`just install-nightly` then adds the `rust-src` component to the nightly (needed
for `-Zbuild-std`); `mise.toml` only pulls `rustfmt`.

**MIPS64 cross-compilation toolchain (manual, apt only):**

```bash
sudo apt install g++-mips64-linux-gnuabi64 libc6-dev-mips64-cross binutils-mips64-linux-gnuabi64
```

macOS: use the Docker path (`just reproducible-prestate-kona`).

#### Build

```bash
cd rust
just build-kona-prestates
```

#### Custom Configs

```bash
export KONA_CUSTOM_CONFIGS_DIR=/path/to/custom/configs
cd rust
just build-kona-prestates
```

### cannon-builder Image

The `cannon-builder` Docker image contains only apt-level MIPS64 cross-compilation
packages. The prestate Dockerfile installs mise on top, and mise pulls Rust (stable
+ nightly), Go, just, and jq from `mise.toml`. The image only needs to be rebuilt
when the cross-compilation toolchain packages change (rare).

## Cutting a Release (for maintainers / forks)

To cut a release of the docker image for any of the targets, cut a new annotated tag for the target like so:

```sh
# Example formats:
# - `kona-host/v0.1.0-beta.8`
# - `cannon-builder/v1.2.0`
TAG="<target_name>/<version>"
git tag -a $TAG -m "<tag description>" && git push origin tag $TAG
```

To run the workflow manually, navigate over to the ["Build and Publish Docker Image"](https://github.com/ethereum-optimism/optimism/actions/workflows/docker.yaml)
action. From there, run a `workflow_dispatch` trigger, select the tag you just pushed, and then finally select the image to release.

Or, if you prefer to use the `gh` CLI, you can run:
```sh
gh workflow run "Build and Publish Docker Image" --ref <tag> -f image_to_release=<target>
```
