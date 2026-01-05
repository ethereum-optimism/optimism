# Boba Network

Boba Network is an Ethereum L2 Optimistic Rollup built on the [OP Stack](https://github.com/ethereum-optimism/optimism).

## Repository Structure

This repository contains Boba-specific extensions and configurations on top of the upstream Optimism codebase, which is included as a git submodule.

```
boba/
├── optimism/              # Git submodule → upstream ethereum-optimism/optimism
├── overlays/              # Boba-specific patches and extensions
│   ├── go.mod.patch       # Patch to use Boba's op-geth fork
│   └── contracts-bedrock/ # Boba contract extensions (BOBA token, etc.)
├── boba-docs/             # Boba Network documentation
├── boba-community/        # Node operator resources (Docker Compose configs)
├── boba_audits/           # Security audit reports
├── packages/boba/         # Boba-specific packages (subgraphs, APIs)
├── ops/                   # Boba operational tooling
├── Makefile               # Build system that applies patches and builds
└── scripts/               # Helper scripts
```

## Getting Started

### Prerequisites

Clone with submodules:

```bash
git clone --recurse-submodules https://github.com/bobanetwork/boba.git
cd boba
```

Or if you've already cloned:

```bash
git submodule update --init --recursive
```

### Building

We recommend building inside the provided container to ensure correct tool versions:

```bash
# Build the builder image (one-time)
podman build -t boba-builder -f ops/docker/mise-builder/Containerfile .

# Build all components
podman run --rm -v $(pwd):/workspace:Z boba-builder bash -c \
    "git submodule update --init --recursive && make build"

# Or for interactive development
podman run -it --rm -v $(pwd):/workspace:Z boba-builder
# Then inside container:
make build
```

### Build Targets

```bash
make build          # Build all Go binaries
make op-node        # Build specific target
make test           # Run tests
make clean          # Clean build artifacts
make submodule-update  # Update upstream to latest develop
make submodule-pin REF=op-node/v1.16.0  # Pin to specific tag
```

## How It Works

The build system:

1. **Applies patches** - Modifies `go.mod` in the submodule to use Boba's `op-geth` fork
2. **Overlays Boba code** - Copies Boba-specific contracts and configs into the submodule
3. **Builds** - Runs the upstream build system
4. **Restores** - Reverts patches so the submodule stays clean

This approach allows us to:
- Track upstream releases precisely with git submodule commits
- Apply minimal, auditable patches
- Avoid merge conflicts from divergent codebases
- Build releases from specific upstream tags

## Key Boba Customizations

### op-geth Fork

The primary customization is using Boba's fork of op-geth instead of the upstream version:

```
github.com/ethereum-optimism/op-geth  →  github.com/bobanetwork/op-geth
```

This is applied via `overlays/go.mod.patch`.

### Boba Contracts

Additional smart contracts in `overlays/contracts-bedrock/src/boba/`:
- `BOBA.sol` - BOBA governance token
- `L2GovernanceERC20.sol` - L2 governance token implementation
- `WETH9.sol` - Wrapped ETH implementation

### Network Configs

Boba network deployment configurations in `overlays/contracts-bedrock/`:
- `deploy-config/boba-mainnet.json`
- `deploy-config/boba-sepolia.json`
- `deployments/boba-mainnet/`
- `deployments/boba-sepolia/`

## Updating Upstream

To update to the latest upstream:

```bash
make submodule-update
# Review changes
git add optimism
git commit -m "Update upstream optimism submodule"
```

To pin to a specific release:

```bash
make submodule-pin REF=op-node/v1.16.0
git add optimism
git commit -m "Pin upstream to op-node/v1.16.0"
```

## Documentation

- [Boba Network Docs](./boba-docs/) - User and developer documentation
- [Node Operator Guide](./boba-community/) - Running Boba nodes
- [Upstream OP Docs](https://docs.optimism.io/) - OP Stack documentation

## Security

- [Security Policy](./SECURITY.md)
- [Boba Audits](./boba_audits/)

## License

[MIT License](./LICENSE)
