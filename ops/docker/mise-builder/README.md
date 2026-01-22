# Mise Builder Container

This directory contains a Containerfile (Dockerfile) for building a development environment that includes all the tools and dependencies needed to build and test the Boba/Optimism monorepo.

## Overview

The container image uses [mise](https://mise.jdx.dev/) to manage tool versions, ensuring you have the exact same versions of Go, Rust, Python, Foundry, and other tools as specified in the repository's `mise.toml` file.

### What's Included

Based on `mise.toml`, this image includes:

- **Languages & Runtimes:**
  - Go 1.23.8
  - Rust 1.83.0
  - Python 3.12.0

- **Build Tools:**
  - Foundry (forge, cast, anvil)
  - just (command runner)
  - jq, yq (JSON/YAML processors)

- **Go Tools:**
  - geth, abigen
  - gotestsum
  - mockery
  - golangci-lint

- **Python Tools:**
  - slither-analyzer
  - semgrep
  - md_toc

- **Other Tools:**
  - svm-rs (Solidity version manager)
  - codecov-uploader
  - goreleaser-pro
  - kurtosis

## Building the Image

### Option 1: Build from Repo Root (Recommended)

From the repository root directory:

```bash
podman build -t boba-builder -f ops/docker/mise-builder/Containerfile .
```

This uses your local `mise.toml` file.

### Option 2: Build Without Repo Context

Build standalone without needing the full repository:

```bash
podman build -t boba-builder ops/docker/mise-builder/
```

This fetches `mise.toml` from the `develop` branch on GitHub.

### Option 3: Build from a Specific Branch/Commit

```bash
podman build -t boba-builder \
  --build-arg REPO_REF=main \
  ops/docker/mise-builder/
```

Replace `main` with any branch name or commit hash.

## Using the Container

### Interactive Development

Start an interactive shell with your repository mounted:

```bash
podman run -it --rm -v $(pwd):/workspace:Z boba-builder
```

Inside the container (git safe.directory is configured automatically):

```bash
# Initialize submodules
git submodule update --init --recursive

# Build the project
make build

# Run tests
make test
```

### One-Off Commands

Run a build without entering the container:

```bash
podman run --rm -v $(pwd):/workspace:Z boba-builder bash -c \
  "git submodule update --init --recursive && make build"
```

Run tests:

```bash
podman run --rm -v $(pwd):/workspace:Z boba-builder bash -c \
  "git submodule update --init --recursive && make test"
```

## Notes

- **Volume Mounting:** The `:Z` flag is required for SELinux systems (like Fedora/RHEL). Omit it on other systems if needed.
- **Build Time:** The initial build will take a while (15-30 minutes) as mise compiles tools from source.
- **Image Size:** The resulting image is fairly large (~5-8 GB) due to all the development tools.
- **User:** The container runs as root to avoid permission issues with mounted volumes.
- **Working Directory:** The container expects your repo to be mounted at `/workspace`.

## Comparison with CI

This container provides a similar environment to the CircleCI builds, which use:
- The `utils/checkout-with-mise` orb to set up tools
- The same `mise.toml` for tool versions

However, CI uses the custom `us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go` image for production builds, while this container is designed for local development and testing.

## Known Issues

### Build Errors Not Seen in CI

The container performs **clean builds from source**, which may reveal compilation errors that are hidden in CI.

**Example:** `OPSuccinctFaultDisputeGame.sol` has a compilation error (wrong event parameter count), but CI doesn't catch it because it uses pre-built artifacts from Google Cloud Storage rather than compiling from scratch.

**Why this happens:**
- CI downloads cached build artifacts before running `forge build`
- If artifacts exist, Forge skips compilation
- Your container does honest, clean builds from source

**Workarounds:**

1. **Pull CI artifacts first** (mimics CI behavior):
   ```bash
   podman run --rm -v $(pwd):/workspace:Z boba-builder bash -c \
     "cd packages/contracts-bedrock && \
      bash scripts/ops/pull-artifacts.sh && \
      forge build"
   ```
   Note: This only works if artifacts exist for your exact commit/tree state.

2. **Build only Go components**:
   ```bash
   podman run --rm -v $(pwd):/workspace:Z boba-builder bash -c \
     "git submodule update --init --recursive && make build-go"
   ```

3. **Accept the build failure**: The container is correctly identifying real bugs that should be fixed.

## Troubleshooting

### mise not activating

If tools aren't available, manually activate mise:

```bash
eval "$(mise activate bash)"
```

### Git safe.directory errors

The container automatically configures git to trust `/workspace` on startup. If you still see errors, you can manually run:

```bash
git config --global --add safe.directory /workspace
```

### Permission issues

The container runs as root to avoid permission conflicts with mounted volumes. Files created in the container will be owned by root. If you need files to be owned by your user, you can:

1. Run `chown -R $(id -u):$(id -g) .` on your host after building
2. Use podman's `--userns=keep-id` flag (experimental)
3. Run cleanup commands in the container before exiting
