# Create L2 Rollup - Code Example

This directory contains the complete working implementation that accompanies the [Create L2 Rollup tutorial](/chain-operators/tutorials/create-l2-rollup/create-l2-rollup). It provides automated deployment of an OP Stack L2 rollup testnet using official published Docker images.

## Overview

This implementation deploys a fully functional OP Stack L2 rollup testnet, including:

- **L1 Smart Contracts** deployed on Sepolia testnet (via op-deployer)
- **Execution Client** (op-geth) processing transactions
- **Consensus Client** (op-node) managing rollup consensus
- **Batcher** (op-batcher) publishing transaction data to L1
- **Proposer** (op-proposer) submitting state root proposals
- **Challenger** (op-challenger) monitoring for disputes

## Prerequisites

### Software Dependencies

| Dependency | Version | Install Command | Purpose |
|------------|---------|-----------------|---------|
| [Docker](https://docs.docker.com/get-docker/) | ^24 | Follow Docker installation guide | Container runtime for OP Stack services |
| [Docker Compose](https://docs.docker.com/compose/install/) | latest | Usually included with Docker Desktop | Multi-container orchestration |
| [jq](https://github.com/jqlang/jq) | latest | `brew install jq` (macOS) / `apt install jq` (Ubuntu) | JSON processing for deployment data |
| [git](https://git-scm.com/) | latest | Usually pre-installed | Cloning repositories for prestate generation |

### Recommended: Tool Management

For the best experience with correct tool versions, we recommend installing [mise](https://mise.jdx.dev/):

```bash
# Install mise (manages all tool versions automatically)
curl https://mise.jdx.dev/install.sh | bash

# Install all required tools with correct versions
cd docs/create-l2-rollup-example
mise install
```

**Why mise?** It automatically handles tool installation and version management, preventing compatibility issues.

### Network Access

- **Sepolia RPC URL**: Get from [Infura](https://infura.io), [Alchemy](https://alchemy.com), or another provider
- **Sepolia ETH**: At least 2-3 ETH for contract deployment and operations
- **Public IP**: For P2P networking (use `curl ifconfig.me` to find your public IP)

## Quick Start

1. **Navigate to this code directory**:
   ```bash
   cd docs/create-l2-rollup-example
   ```

2. **Configure environment variables**:
   ```bash
   cp .example.env .env
   # Edit .env with your actual values (L1_RPC_URL, PRIVATE_KEY, L2_CHAIN_ID)
   ```

3. **Download op-deployer**:
   ```bash
   make init    # Download op-deployer
   ```

4. **Deploy and start everything**:
   ```bash
   make setup   # Deploy contracts and configure all services
   make up      # Start all services
   ```

5. **Verify deployment**:
   ```bash
   make status  # Check service health
   make test-l1 # Verify L1 connectivity
   make test-l2 # Verify L2 functionality

   # Or manually check:
   docker-compose ps
   docker-compose logs -f op-node
   ```

## Environment Configuration

Copy `.example.env` to `.env` and configure the following variables:

```bash
# L1 Configuration (Sepolia testnet)
# Option 1: Public endpoint (no API key required)
L1_RPC_URL="https://ethereum-sepolia-rpc.publicnode.com"
L1_BEACON_URL="https://ethereum-sepolia-beacon-api.publicnode.com"

# Option 2: Private endpoint (requires API key)
# L1_RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"
# L1_BEACON_URL="https://ethereum-sepolia-beacon-api.publicnode.com"

# Private key for deployment and operations
# IMPORTANT: Never commit this to version control
PRIVATE_KEY="YOUR_PRIVATE_KEY_WITHOUT_0x_PREFIX"

# Optional: Public IP for P2P networking (defaults to 127.0.0.1 for local testing)
P2P_ADVERTISE_IP="127.0.0.1"

# Optional: Custom L2 Chain ID (default: 16584)
L2_CHAIN_ID="16584"
```

The `.env` file will be automatically loaded by Docker Compose.

## Manual Setup (Alternative)

For detailed manual setup instructions, see the [Create L2 Rollup tutorial](/chain-operators/tutorials/create-l2-rollup/create-l2-rollup). The tutorial provides step-by-step guidance for setting up each component individually if you prefer not to use the automated approach.

## Directory Structure

```
create-l2-rollup-example/
├── .example.env           # Environment variables template
├── docker-compose.yml     # Service orchestration
├── Makefile               # Automation commands
├── scripts/
│   ├── setup-rollup.sh   # Automated deployment script
│   └── download-op-deployer.sh # op-deployer downloader
└── README.md             # This file
```

**Generated directories** (created during deployment):

```
deployer/                # op-deployer configuration and outputs
├── .deployer/           # Deployment artifacts (genesis.json, rollup.json, etc.)
├── addresses/           # Generated wallet addresses
└── .env                 # Environment variables
batcher/                 # op-batcher configuration
└── .env                 # Environment variables
proposer/                # op-proposer configuration
└── .env                 # Environment variables
challenger/              # op-challenger configuration
├── .env                 # Challenger-specific environment variables
└── data/                # Challenger data directory
sequencer/               # op-sequencer configuration
├── .env                 # op-sequencer environment variables
├── genesis.json         # op-geth genesis file
├── jwt.txt              # JWT secret for auth RPC
├── rollup.json          # op-node rollup configuration
└── op-geth-data/        # op-geth data directory
```

## Service Ports

| Service | Port | Description |
|---------|------|-------------|
| op-geth | 8545 | HTTP RPC endpoint |
| op-geth | 8546 | WebSocket RPC endpoint |
| op-geth | 8551 | Auth RPC for op-node |
| op-node | 8547 | op-node RPC endpoint |
| op-node | 9222 | P2P networking |

## Monitoring and Logs

```bash
# View all service logs
make logs

# View specific service logs
docker-compose logs -f op-node
docker-compose logs -f op-geth

# Check service health
make status

# Restart all services
make restart

# Restart a specific service
docker-compose restart op-batcher
```

## P2P Networking Configuration

By default, this devnet disables P2P networking entirely to avoid validation warnings when running locally. The `--p2p.disable` flag is set in `docker-compose.yml` (line 26).

<Warning>
**For production deployments**, you must **remove** the `--p2p.disable` flag and configure P2P networking properly. P2P is essential for:
- Distributing newly sequenced blocks to other nodes in the network
- Enabling peer nodes to sync and validate your chain
- Supporting a decentralized network of nodes
- Network resilience and redundancy
</Warning>

### When to Enable P2P

| Environment | P2P Networking | Reason |
|-------------|---------------|---------|
| **Local devnet** | Disabled (default) | Prevents P2P warnings when testing solo without peers |
| **Private testnet** | Disabled | No other nodes to connect with |
| **Public testnet** | Enabled | Other nodes need to receive blocks and sync |
| **Production mainnet** | Enabled | Required for network operation |

### Enabling P2P for Production

1. Open `docker-compose.yml`
2. Remove `--p2p.disable  # For local devnet only...`
3. Add back the P2P configuration flags:
   ```yaml
   --p2p.listen.ip=0.0.0.0
   --p2p.listen.tcp=9222
   --p2p.listen.udp=9222
   --p2p.advertise.ip=${P2P_ADVERTISE_IP}
   --p2p.advertise.tcp=9222
   --p2p.advertise.udp=9222
   --p2p.sequencer.key=${PRIVATE_KEY}
   ```
4. Ensure your P2P networking environment is properly configured:
   - Set `P2P_ADVERTISE_IP` in `.env` to your public IP address (not 127.0.0.1)
   - Ensure port 9222 (both TCP and UDP) is accessible from the internet
   - Configure proper firewall rules to allow P2P traffic
   - Consider setting up bootnodes for better peer discovery

```bash
# Example: Quick enable P2P for testing
sed -i '' '/--p2p.disable/d' docker-compose.yml
# Then add back P2P flags manually in docker-compose.yml
docker-compose restart op-node
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 8545-8551 and 9222 are available
2. **Insufficient ETH**: Make sure your deployment wallet has enough Sepolia ETH
3. **Network timeouts**: Check your L1 RPC URL and network connectivity
4. **Docker issues**: Ensure Docker daemon is running and you have sufficient resources

### Reset Deployment

To reset and redeploy:

```bash
# Stop all services and clean up
make clean

# Re-run setup
make setup
make up
```

## Security Notes

- **Never commit private keys** to version control
- **Use hardware security modules (HSMs)** for production deployments
- **Monitor gas costs** on Sepolia testnet
- **Backup wallet addresses** and deployment artifacts

## About This Code

This code example accompanies the [Create L2 Rollup tutorial](/chain-operators/tutorials/create-l2-rollup/create-l2-rollup) in the Optimism documentation. It provides a complete, working implementation that demonstrates the concepts covered in the tutorial.

## Contributing

This code is part of the Optimism documentation. For issues or contributions:

- **Documentation issues**: Report on the main [Optimism repository](https://github.com/ethereum-optimism/optimism)
- **Code improvements**: Submit pull requests to the Optimism monorepo
- **Tutorial feedback**: Use the documentation feedback mechanisms

## License

This code is provided as-is for educational and testing purposes. See the Optimism monorepo for licensing information.
