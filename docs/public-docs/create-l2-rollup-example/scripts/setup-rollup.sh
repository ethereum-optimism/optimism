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
    for role in admin base_fee_vault_recipient l1_fee_vault_recipient sequencer_fee_vault_recipient system_config unsafe_block_signer operator_fee_vault_recipient batcher proposer challenger ; do
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

    # Operational signers (batcher, proposer) get REAL, dedicated keys.
    # These roles sign L1 transactions continuously, so they must NOT reuse the
    # deployer key — day-to-day operation should never touch the deployer. We
    # derive a real address with cast and persist the private key so the services
    # can sign with it (and so we can fund it below).
    for role in batcher proposer; do
        wallet=$(cast wallet new --json)
        private_key=$(echo "$wallet" | jq -r '.[0].private_key')
        address=$(echo "$wallet" | jq -r '.[0].address')
        echo "${private_key#0x}" > "${role}_private_key.txt"
        echo "$address" > "${role}_address.txt"
        chmod 600 "${role}_private_key.txt"
        log_info "Generated dedicated $role signer: $address"
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
    # custom intent: deploys a standalone superchain (fresh SuperchainConfig) rather
    # than joining the shared Sepolia superchain — avoids OPCM SuperchainConfigNeedsUpgrade
    # when using a release-candidate op-deployer/contracts version (e.g. v0.7.0-rc.1).
    op-deployer init \
        --l1-chain-id $L1_CHAIN_ID \
        --l2-chain-ids "$L2_CHAIN_ID_DECIMAL" \
        --workdir .deployer \
        --intent-type custom

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
    ADMIN_ADDR=$(cat addresses/admin_address.txt)

    # Owners and non-signing roles stay as the deployer for simplicity: the deployer
    # already controls the chain, and these roles do not sign L1 transactions during
    # normal operation. The batcher and proposer, however, keep their own dedicated
    # addresses (generated in generate_addresses and read above) and are funded
    # separately — so the chain operates without ever using the deployer key.
    DEPLOYER_ADDR=$(cast wallet address --private-key "0x${PRIVATE_KEY#0x}")
    ADMIN_ADDR="$DEPLOYER_ADDR"
    CHALLENGER_ADDR="$DEPLOYER_ADDR"
    SYSTEM_CONFIG_ADDR="$DEPLOYER_ADDR"
    UNSAFE_BLOCK_SIGNER_ADDR="$DEPLOYER_ADDR"

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
    # op-deployer v0.7.0-rc.1 requires the ProxyAdmin owners to be non-zero
    sed -i.bak "s|l1ProxyAdminOwner = .*|l1ProxyAdminOwner = \"$ADMIN_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|l2ProxyAdminOwner = .*|l2ProxyAdminOwner = \"$ADMIN_ADDR\"|" .deployer/intent.toml

    # custom configType: set the superchain roles (deploying our own superchain)
    sed -i.bak "s|SuperchainProxyAdminOwner = .*|SuperchainProxyAdminOwner = \"$ADMIN_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|SuperchainGuardian = .*|SuperchainGuardian = \"$ADMIN_ADDR\"|" .deployer/intent.toml
    sed -i.bak "s|Challenger = .*|Challenger = \"$CHALLENGER_ADDR\"|" .deployer/intent.toml

    # custom intent ships with zero eip1559 params — set standard OP values or the chain is invalid
    sed -i.bak "s|eip1559DenominatorCanyon = .*|eip1559DenominatorCanyon = 250|" .deployer/intent.toml
    sed -i.bak "s|eip1559Denominator = .*|eip1559Denominator = 50|" .deployer/intent.toml
    sed -i.bak "s|eip1559Elasticity = .*|eip1559Elasticity = 6|" .deployer/intent.toml

    # activate Karst (the U19 hardfork) at genesis
    if ! grep -q 'globalDeployOverrides' .deployer/intent.toml; then
        printf '\n[globalDeployOverrides]\n  l2GenesisKarstTimeOffset = "0x0"\n' >> .deployer/intent.toml
    fi
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

# Fund the operational signers (batcher, proposer) from the deployer
fund_operators() {
    log_info "Funding operational signers from the deployer..."

    BATCHER_ADDR=$(cat "$DEPLOYER_DIR/addresses/batcher_address.txt")
    PROPOSER_ADDR=$(cat "$DEPLOYER_DIR/addresses/proposer_address.txt")

    # Sepolia ETH sent to each signer. The batcher posts batches every channel and
    # the proposer posts output roots on its interval, so both need ongoing L1 gas.
    # Override with OPERATOR_FUND_AMOUNT in your .env if you want more headroom.
    FUND_AMOUNT=${OPERATOR_FUND_AMOUNT:-0.1ether}

    for entry in "batcher:$BATCHER_ADDR" "proposer:$PROPOSER_ADDR"; do
        role="${entry%%:*}"
        addr="${entry#*:}"
        log_info "Funding $role ($addr) with $FUND_AMOUNT..."
        cast send "$addr" \
            --value "$FUND_AMOUNT" \
            --private-key "0x${PRIVATE_KEY#0x}" \
            --rpc-url "$L1_RPC_URL" \
            --confirmations 1
        balance=$(cast balance "$addr" --rpc-url "$L1_RPC_URL")
        log_info "$role L1 balance: $balance wei"
    done

    log_success "Operational signers funded"
}

# Zero the permissioned (game type 1) init bond
zero_permissioned_bond() {
    log_info "Setting permissioned (game type 1) init bond to 0..."

    # The DisputeGameFactory enforces initBonds[gameType] on EVERY game creation,
    # permissioned included — bonds are not exclusive to the permissionless game.
    # op-deployer sets a non-zero bond for the permissioned game, which would force
    # us to refund the proposer a fresh bond before every proposal (it otherwise
    # starves after one game). On a single-operator permissioned demo chain there is
    # no untrusted party to bond against, so we set it to 0 and the proposer runs
    # freely. The factory owner is the deployer.
    DGF_ADDR=$(jq -r '.opChainDeployments[0].DisputeGameFactoryProxy' "$DEPLOYER_DIR/.deployer/state.json")

    cast send "$DGF_ADDR" 'setInitBond(uint32,uint256)' 1 0 \
        --private-key "0x${PRIVATE_KEY#0x}" \
        --rpc-url "$L1_RPC_URL" \
        --confirmations 1 > /dev/null

    log_success "Permissioned init bond set to 0"
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

    # Use the dedicated batcher key (not the deployer key) so batch submission
    # signs as the batcher role configured in the system config.
    BATCHER_KEY=$(cat "$DEPLOYER_DIR/addresses/batcher_private_key.txt")

    # Create .env file with OP_BATCHER prefixed variables
    cat > .env << EOF
OP_BATCHER_L2_ETH_RPC=http://op-reth:8545
OP_BATCHER_ROLLUP_RPC=http://op-node:8547
OP_BATCHER_PRIVATE_KEY=$BATCHER_KEY
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

    # Use the dedicated proposer key (not the deployer key) so output proposals
    # sign as the proposer role.
    PROPOSER_KEY=$(cat "$DEPLOYER_DIR/addresses/proposer_private_key.txt")

    # Create .env file with OP_PROPOSER prefixed variables
    cat > .env << EOF
OP_PROPOSER_GAME_FACTORY_ADDRESS=$GAME_FACTORY_ADDR
# A fresh op-deployer chain ships permissioned: the OptimismPortal respects game
# type 1 (PermissionedDisputeGame) and only that impl is registered in the
# DisputeGameFactory. The proposer must therefore propose game type 1 — type 0
# (permissionless) has no implementation on this chain.
OP_PROPOSER_PRIVATE_KEY=$PROPOSER_KEY
OP_PROPOSER_POLL_INTERVAL=20s
OP_PROPOSER_GAME_TYPE=1
OP_PROPOSER_PROPOSAL_INTERVAL=3600s
EOF

    log_success "Proposer setup complete"
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
    fund_operators
    zero_permissioned_bond
    generate_config
    setup_sequencer
    setup_batcher
    setup_proposer

    log_success "OP Stack L2 Rollup deployment complete!"
    log_info "Run 'docker-compose up -d' to start the rollup (op-reth, op-node, batcher, proposer)."
    log_info "Your chain is permissioned (game type 1) and finalizes via the proposer's uncontested"
    log_info "dispute games. To add a fault-proof challenger or move to permissionless fault proofs,"
    log_info "see https://docs.optimism.io/chain-operators/tutorials/migrating-permissionless"
}

main "$@"
