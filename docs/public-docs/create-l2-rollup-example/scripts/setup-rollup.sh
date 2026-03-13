#!/bin/bash

# OP Stack L2 Rollup Setup Script
# This script automates the deployment of a complete L2 rollup testnet

set -e

# Check if running in Docker
if [ -f "/.dockerenv" ]; then
    log_error "This script should not be run inside Docker. Run it on the host system."
    exit 1
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
L1_CHAIN_ID=11155111  # Sepolia
L2_CHAIN_ID_DECIMAL=${L2_CHAIN_ID:-16584}  # Default test chain ID (decimal)
L2_CHAIN_ID=$(printf "0x%064x" "$L2_CHAIN_ID_DECIMAL")  # Convert to full 64-char hex format for TOML
P2P_ADVERTISE_IP=${P2P_ADVERTISE_IP:-127.0.0.1}  # Default to localhost for local testing
WORKSPACE_DIR="$(pwd)"
ROLLUP_DIR="$WORKSPACE_DIR"
DEPLOYER_DIR="$ROLLUP_DIR/deployer"
SEQUENCER_DIR="$ROLLUP_DIR/sequencer"
BATCHER_DIR="$ROLLUP_DIR/batcher"
PROPOSER_DIR="$ROLLUP_DIR/proposer"
CHALLENGER_DIR="$ROLLUP_DIR/challenger"
DISPUTE_MON_DIR="$ROLLUP_DIR/dispute-mon"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v op-deployer &> /dev/null; then
        log_error "op-deployer not found. Please ensure it's downloaded"
        log_info "Run: make download"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Generate wallet addresses
generate_addresses() {
    log_info "Generating wallet addresses..."

    mkdir -p "$DEPLOYER_DIR/addresses"
    log_info "Created addresses directory: $DEPLOYER_DIR/addresses"

    cd "$DEPLOYER_DIR/addresses"
    log_info "Changed to directory: $(pwd)"

    # Generate addresses for different roles using openssl
    for role in admin base_fee_vault_recipient l1_fee_vault_recipient sequencer_fee_vault_recipient system_config unsafe_block_signer operator_fee_vault_recipient chain_fees_fee_recipient batcher proposer challenger ; do
        # Generate a random 32-byte private key, ensuring it's not zero
        private_key=""
        while [ -z "$private_key" ] || [ "$private_key" = "0000000000000000000000000000000000000000000000000000000000000000" ]; do
            private_key=$(openssl rand -hex 32)
        done

        # For this demo, we'll create a fake but valid Ethereum address
        # In a real scenario, you'd derive the actual Ethereum address from the private key
        # Create a 40-character hex address (20 bytes)
        address="0x$(echo "$private_key" | head -c 40)"
        echo "$address" > "${role}_address.txt"
        log_info "Created wallet for $role: $address"
    done

    log_success "Wallet addresses generated"
    log_info "Addresses generated successfully, continuing to init..."
}

# Initialize op-deployer
init_deployer() {
    log_info "Initializing op-deployer..."

    cd "$DEPLOYER_DIR"

    # Create .env file
    cat > .env << EOF
L1_RPC_URL=$L1_RPC_URL
PRIVATE_KEY=$PRIVATE_KEY
EOF

    # Remove any existing .deployer directory for clean state
    rm -rf .deployer

        # Initialize intent
    op-deployer init \
        --l1-chain-id $L1_CHAIN_ID \
        --l2-chain-ids "$L2_CHAIN_ID_DECIMAL" \
        --workdir .deployer \
        --intent-type standard-overrides

    log_success "op-deployer initialized"
}

# Update intent configuration
update_intent() {
    log_info "Updating intent configuration..."

    # Read generated addresses
    BASE_FEE_VAULT_ADDR=$(cat addresses/base_fee_vault_recipient_address.txt)
    L1_FEE_VAULT_ADDR=$(cat addresses/l1_fee_vault_recipient_address.txt)
    SEQUENCER_FEE_VAULT_ADDR=$(cat addresses/sequencer_fee_vault_recipient_address.txt)
    SYSTEM_CONFIG_ADDR=$(cat addresses/system_config_address.txt)
    UNSAFE_BLOCK_SIGNER_ADDR=$(cat addresses/unsafe_block_signer_address.txt)
    BATCHER_ADDR=$(cat addresses/batcher_address.txt)
    PROPOSER_ADDR=$(cat addresses/proposer_address.txt)
    CHALLENGER_ADDR=$(cat addresses/challenger_address.txt)
    OPERATOR_FEE_VAULT_ADDR=$(cat addresses/operator_fee_vault_recipient_address.txt)
    CHAIN_FEES_RECIPIENT_ADDR=$(cat addresses/chain_fees_fee_recipient_address.txt)

    # Keep the default contract locators and opcmAddress from op-deployer init

    # Update only the chain-specific fields in the existing intent.toml
    L2_CHAIN_ID_HEX=$(printf "0x%064x" "$L2_CHAIN_ID")
    sed -i.bak "s|id = .*|id = \"$L2_CHAIN_ID_HEX\"|" .deployer/intent.toml
    sed -i.bak "s|baseFeeVaultRecipient = .*|baseFeeVaultRecipient = \"$BASE_FEE_VAULT_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|l1FeeVaultRecipient = .*|l1FeeVaultRecipient = \"$L1_FEE_VAULT_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|sequencerFeeVaultRecipient = .*|sequencerFeeVaultRecipient = \"$SEQUENCER_FEE_VAULT_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|systemConfigOwner = .*|systemConfigOwner = \"$SYSTEM_CONFIG_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|unsafeBlockSigner = .*|unsafeBlockSigner = \"$UNSAFE_BLOCK_SIGNER_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|batcher = .*|batcher = \"$BATCHER_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|proposer = .*|proposer = \"$PROPOSER_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|challenger = .*|challenger = \"$CHALLENGER_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|fundDevAccounts = .*|fundDevAccounts = true|" .deployer/intent.toml
    sed -i.bak "s|operatorFeeVaultRecipient = .*|operatorFeeVaultRecipient = \"$OPERATOR_FEE_VAULT_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|chainFeesRecipient = .*|chainFeesRecipient = \"$CHAIN_FEES_RECIPIENT_ADDR\"|" .deployer/intent.toml
    log_success "Intent configuration updated"
}

# Deploy L1 contracts
deploy_contracts() {
    log_info "Deploying L1 contracts..."

    cd "$DEPLOYER_DIR"

    op-deployer apply \
        --workdir .deployer \
        --l1-rpc-url "$L1_RPC_URL" \
        --private-key "$PRIVATE_KEY"

    log_success "L1 contracts deployed"
}

# Generate chain configuration
generate_config() {
    log_info "Generating chain configuration..."

    cd "$DEPLOYER_DIR"

    op-deployer inspect genesis --workdir .deployer "$L2_CHAIN_ID" > .deployer/genesis.json
    op-deployer inspect rollup --workdir .deployer "$L2_CHAIN_ID" > .deployer/rollup.json

    log_success "Chain configuration generated"
}

# Setup sequencer
setup_sequencer() {
    log_info "Setting up sequencer..."

    mkdir -p "$SEQUENCER_DIR"
    cd "$SEQUENCER_DIR"

    # Copy configuration files
    cp "$DEPLOYER_DIR/.deployer/genesis.json" .
    cp "$DEPLOYER_DIR/.deployer/rollup.json" .

    # Generate JWT secret
    openssl rand -hex 32 > jwt.txt
    chmod 600 jwt.txt

    # Create .env file
    cat > .env << EOF
L1_RPC_URL=$L1_RPC_URL
L1_BEACON_URL=$L1_BEACON_URL
PRIVATE_KEY=$PRIVATE_KEY
P2P_ADVERTISE_IP=$P2P_ADVERTISE_IP
L2_CHAIN_ID=$L2_CHAIN_ID
EOF

    log_success "Sequencer setup complete"
}

# Setup batcher
setup_batcher() {
    log_info "Setting up batcher..."

    mkdir -p "$BATCHER_DIR"
    cd "$BATCHER_DIR"

    # Copy state file
    cp "$DEPLOYER_DIR/.deployer/state.json" .
    INBOX_ADDRESS=$(cat state.json | jq -r '.opChainDeployments[0].SystemConfigProxy')
    
    # Create .env file with OP_BATCHER prefixed variables
    cat > .env << EOF
OP_BATCHER_L2_ETH_RPC=http://op-geth:8545
OP_BATCHER_ROLLUP_RPC=http://op-node:8547
OP_BATCHER_PRIVATE_KEY=$PRIVATE_KEY
OP_BATCHER_POLL_INTERVAL=1s
OP_BATCHER_SUB_SAFETY_MARGIN=6
OP_BATCHER_NUM_CONFIRMATIONS=1
OP_BATCHER_SAFE_ABORT_NONCE_TOO_LOW_COUNT=3
OP_BATCHER_INBOX_ADDRESS=$INBOX_ADDRESS
EOF

    log_success "Batcher setup complete"
}

# Setup proposer
setup_proposer() {
    log_info "Setting up proposer..."

    mkdir -p "$PROPOSER_DIR"
    cd "$PROPOSER_DIR"

    # Copy state file
    cp "$DEPLOYER_DIR/.deployer/state.json" .

    # Extract dispute game factory address
    GAME_FACTORY_ADDR=$(cat state.json | jq -r '.opChainDeployments[0].DisputeGameFactoryProxy')

    # Create .env file with OP_PROPOSER prefixed variables
    cat > .env << EOF
OP_PROPOSER_GAME_FACTORY_ADDRESS=$GAME_FACTORY_ADDR
OP_PROPOSER_PRIVATE_KEY=$PRIVATE_KEY
OP_PROPOSER_POLL_INTERVAL=20s
OP_PROPOSER_GAME_TYPE=0
OP_PROPOSER_PROPOSAL_INTERVAL=3600s
EOF

    log_success "Proposer setup complete"
}

# Setup challenger
setup_challenger() {
    log_info "Setting up challenger..."

    mkdir -p "$CHALLENGER_DIR"
    cd "$CHALLENGER_DIR"

    # Copy configuration files
    cp "$DEPLOYER_DIR/.deployer/genesis.json" .
    cp "$DEPLOYER_DIR/.deployer/rollup.json" .

    # Extract dispute game factory address
    GAME_FACTORY_ADDR=$(cat ../deployer/.deployer/state.json | jq -r '.opChainDeployments[0].DisputeGameFactoryProxy')

    # Check for existing prestate file
    CHALLENGER_PRESTATE_FILE=""
    if [ -d "$CHALLENGER_DIR" ]; then
        # Look for any .bin.gz file in the challenger directory
        EXISTING_PRESTATE=$(find "$CHALLENGER_DIR" -maxdepth 1 -name "*.bin.gz" -print -quit 2>/dev/null)
        if [ -n "$EXISTING_PRESTATE" ]; then
            CHALLENGER_PRESTATE_FILE=$(basename "$EXISTING_PRESTATE")
            log_info "Found existing prestate file: $CHALLENGER_PRESTATE_FILE"
        fi
    fi

    # Ensure prestate file exists
    if [ -z "$CHALLENGER_PRESTATE_FILE" ] || [ ! -f "$CHALLENGER_DIR/$CHALLENGER_PRESTATE_FILE" ]; then
        log_error "Challenger prestate file not found in $CHALLENGER_DIR/"
        log_error "Make sure generate_challenger_prestate runs before setup_challenger"
        return 1
    fi

    # Create .env file with OP_CHALLENGER prefixed variables
    cat > .env << EOF
OP_CHALLENGER_GAME_FACTORY_ADDRESS=$GAME_FACTORY_ADDR
OP_CHALLENGER_PRIVATE_KEY=$PRIVATE_KEY
OP_CHALLENGER_CANNON_PRESTATE=/workspace/$CHALLENGER_PRESTATE_FILE
EOF

    log_success "Challenger setup complete"
}

# Validate main .env file for docker-compose
validate_main_env() {
    log_info "Validating .env file..."

    # Load user-provided .env file
    USER_ENV_FILE="$(dirname "$0")/../.env"
    if [ ! -f "$USER_ENV_FILE" ]; then
        log_error ".env file not found. Please run 'make init' first."
        log_info "Run: make init"
        return 1
    fi

    log_info "Loading .env file..."
    set -a  # automatically export all variables
    source "$USER_ENV_FILE"
    set +a

    # Validate required environment variables
    if [ -z "$L1_RPC_URL" ]; then
        log_error "L1_RPC_URL is not set. Please set it in your .env file."
        log_info "Example: L1_RPC_URL=https://ethereum-sepolia-rpc.publicnode.com"
        return 1
    fi

    if [ -z "$L1_BEACON_URL" ]; then
        log_error "L1_BEACON_URL is not set. Please set it in your .env file."
        log_info "Example: L1_BEACON_URL=https://ethereum-sepolia-beacon-api.publicnode.com"
        return 1
    fi

    if [ -z "$PRIVATE_KEY" ]; then
        log_error "PRIVATE_KEY is not set. Please set it in your .env file."
        log_info "Example: PRIVATE_KEY=0x..."
        return 1
    fi

    if [ -z "$L2_CHAIN_ID" ]; then
        log_error "L2_CHAIN_ID is not set. Please set it in your .env file."
        log_info "Example: L2_CHAIN_ID=16584"
        return 1
    fi

    # Set defaults for optional values
    P2P_ADVERTISE_IP=${P2P_ADVERTISE_IP:-127.0.0.1}

    log_success ".env file validated"
}

# Generate challenger prestate
generate_challenger_prestate() {
    log_info "Generating challenger prestate..."

    # Get the chain ID from the rollup config
    if [ -f "$DEPLOYER_DIR/.deployer/rollup.json" ]; then
        CHAIN_ID=$(jq -r '.l2_chain_id' "$DEPLOYER_DIR/.deployer/rollup.json")
    else
        log_error "Could not find rollup.json to determine chain ID"
        return 1
    fi

    log_info "Chain ID for prestate generation: $CHAIN_ID"

    # Create optimism directory for prestate generation
    OPTIMISM_DIR="$ROLLUP_DIR/optimism"
    if [ ! -d "$OPTIMISM_DIR" ]; then
        log_info "Cloning Optimism repository..."
        git clone https://github.com/ethereum-optimism/optimism.git "$OPTIMISM_DIR"
        cd "$OPTIMISM_DIR"

        # Find the latest op-program tag
        log_info "Finding latest op-program version..."
        OP_PROGRAM_TAG=$(git tag --list "op-program/v*" | sort -V | tail -1)
        if [ -z "$OP_PROGRAM_TAG" ]; then
            log_error "Could not find any op-program tags"
            return 1
        fi
        log_info "Using op-program version: $OP_PROGRAM_TAG"

        git checkout "$OP_PROGRAM_TAG"
        git submodule update --init --recursive
    else
        log_info "Optimism repository already exists, checking configuration..."
        cd "$OPTIMISM_DIR"

        # Check if we're on the correct op-program tag
        CURRENT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
        LATEST_TAG=$(git tag --list "op-program/v*" | sort -V | tail -1)

        if [ "$CURRENT_TAG" != "$LATEST_TAG" ]; then
            log_info "Updating to latest op-program version: $LATEST_TAG"
            git checkout "$LATEST_TAG"
            git submodule update --init --recursive
        else
            log_info "Already on correct op-program version: $CURRENT_TAG"
        fi
    fi

    # Copy configuration files
    log_info "Copying chain configuration files..."
    mkdir -p op-program/chainconfig/configs
    cp "$DEPLOYER_DIR/.deployer/rollup.json" "op-program/chainconfig/configs/${CHAIN_ID}-rollup.json"
    cp "$DEPLOYER_DIR/.deployer/genesis.json" "op-program/chainconfig/configs/${CHAIN_ID}-genesis-l2.json"

    # Generate prestate
    log_info "Generating reproducible prestate..."
    make reproducible-prestate

    # Extract the prestate hash from the JSON file
    PRESTATE_HASH=$(jq -r '.pre' op-program/bin/prestate-proof-mt64.json)
    if [ -z "$PRESTATE_HASH" ] || [ "$PRESTATE_HASH" = "null" ]; then
        log_error "Could not extract prestate hash from prestate-proof-mt64.json"
        return 1
    fi

    log_info "Prestate hash: $PRESTATE_HASH"

    # Move the prestate file
    log_info "Moving prestate file to challenger directory..."
    mkdir -p "$ROLLUP_DIR/challenger"
    mv "op-program/bin/prestate-mt64.bin.gz" "$ROLLUP_DIR/challenger/${PRESTATE_HASH}.bin.gz"

    # Verify the prestate file was created successfully
    if [ ! -f "$ROLLUP_DIR/challenger/${PRESTATE_HASH}.bin.gz" ]; then
        log_error "Failed to create prestate file: $ROLLUP_DIR/challenger/${PRESTATE_HASH}.bin.gz"
        return 1
    fi

    # Clean up
    cd "$ROLLUP_DIR"
    # rm -rf "$OPTIMISM_DIR"  # Uncomment to clean up after successful generation

    log_success "Challenger prestate generation complete: $ROLLUP_DIR/challenger/${PRESTATE_HASH}.bin.gz"
}

# Setup dispute monitor
setup_dispute_monitor() {
    log_info "Setting up dispute monitor..."

    mkdir -p "$DISPUTE_MON_DIR"
    cd "$DISPUTE_MON_DIR"

    # Get required addresses from state.json
    GAME_FACTORY_ADDRESS=$(jq -r '.opChainDeployments[0].DisputeGameFactoryProxy' "$DEPLOYER_DIR/.deployer/state.json")
    PROPOSER_ADDRESS=$(jq -r '.appliedIntent.chains[0].roles.proposer' "$DEPLOYER_DIR/.deployer/state.json")
    CHALLENGER_ADDRESS=$(jq -r '.appliedIntent.chains[0].roles.challenger' "$DEPLOYER_DIR/.deployer/state.json")

    log_info "Game Factory: $GAME_FACTORY_ADDRESS"
    log_info "Proposer: $PROPOSER_ADDRESS"
    log_info "Challenger: $CHALLENGER_ADDRESS"

    # Create environment file for dispute monitor
    cat > .env << EOF
# Rollup RPC Configuration
ROLLUP_RPC=http://op-node:8547

# Contract Addresses
OP_DISPUTE_MON_GAME_FACTORY_ADDRESS=$GAME_FACTORY_ADDRESS

# Honest Actors
PROPOSER_ADDRESS=$PROPOSER_ADDRESS
CHALLENGER_ADDRESS=$CHALLENGER_ADDRESS

# Network Configuration
OP_DISPUTE_MON_NETWORK=op-sepolia

# Monitoring Configuration
OP_DISPUTE_MON_MONITOR_INTERVAL=10s
EOF

    # Create logs directory
    mkdir -p logs

    log_success "Dispute monitor configuration created"
    log_info "Dispute monitor will start with 'docker-compose up -d' from project root"
    log_info "To verify it's working, run: curl -s http://localhost:7300/metrics | grep -E \"op_dispute_mon_(games|ignored)\" | head -10"
}

# Add op-deployer to PATH if it exists in the workspace
if [ -f "$(dirname "$0")/../op-deployer" ]; then
    OP_DEPLOYER_PATH="$(cd "$(dirname "$0")/.." && pwd)/op-deployer"
    export PATH="$(dirname "$OP_DEPLOYER_PATH"):$PATH"
    log_info "Added op-deployer to PATH: $OP_DEPLOYER_PATH"
fi

# Main execution
main() {
    log_info "Starting OP Stack L2 Rollup deployment..."
    log_info "L2 Chain ID: $L2_CHAIN_ID"
    log_info "L1 Chain ID: $L1_CHAIN_ID"

    # Clean start - remove any existing deployer directory
    log_info "Cleaning up any existing deployment..."
    rm -rf "$DEPLOYER_DIR"
    mkdir -p "$DEPLOYER_DIR"

    validate_main_env
    check_prerequisites
    generate_addresses
    init_deployer
    update_intent
    deploy_contracts
    generate_config
    setup_sequencer
    setup_batcher
    setup_proposer
    generate_challenger_prestate
    setup_challenger
    setup_dispute_monitor

    log_success "OP Stack L2 Rollup deployment complete!"
    log_info "Run 'docker-compose up -d' to start all services (including dispute monitor)"
    log_info "Dispute monitor metrics: http://localhost:7300/metrics"
}

# Handle command line arguments for standalone function calls
if [ $# -gt 0 ]; then
    case "$1" in
        "prestate"|"generate-prestate")
            log_info "Running standalone prestate generation..."

            # Set up required variables for standalone execution
            SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
            ROLLUP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
            DEPLOYER_DIR="$ROLLUP_DIR/deployer"

            generate_challenger_prestate
            exit $?
            ;;
        *)
            log_error "Unknown command: $1"
            log_info "Available commands: prestate"
            exit 1
            ;;
    esac
fi

# Run main function if no arguments provided
main "$@"
