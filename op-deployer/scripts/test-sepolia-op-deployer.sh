#!/bin/bash
set -e

# OP Deployer - Sepolia Deployment & Verification Script
############ This script is only for testing ############
# Use it at your own risk.
#
# This script supports environment variables to avoid repetitive input:
#
# Required:
#   L1_RPC_URL               - Sepolia RPC endpoint
#   DEPLOYER_PRIVATE_KEY     - Private key (must have Sepolia ETH)
#
# Optional:
#   DEPLOYER_VERIFIER_TYPE      - Verifier(s): etherscan, blockscout, or etherscan,blockscout
#   DEPLOYER_VERIFIER_API_KEY   - API key for Etherscan (if using etherscan verifier)
#
# For superchain deployment (step 1):
#   DEPLOYER_PROXY_ADMIN_OWNER       - Superchain proxy admin owner address
#   DEPLOYER_PROTOCOL_VERSIONS_OWNER - Protocol versions owner address  
#   DEPLOYER_GUARDIAN                - Guardian address
#
# For implementations deployment (step 2):
#   DEPLOYER_PROTOCOL_VERSIONS_PROXY - Protocol versions proxy address (from step 1)
#   DEPLOYER_SUPERCHAIN_CONFIG_PROXY - Superchain config proxy address (from step 1)
#   DEPLOYER_SUPERCHAIN_PROXY_ADMIN  - Superchain proxy admin address (from step 1)
#   DEPLOYER_L1_PROXY_ADMIN_OWNER    - L1 proxy admin owner address
#   DEPLOYER_CHALLENGER              - Challenger address
#
# After step 1, the script outputs export commands for easy step 2 deployment.

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}   OP Deployer - Sepolia Deployment & Verification Script${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$REPO_ROOT/.deployer-output"
mkdir -p "$OUTPUT_DIR"

cd "$REPO_ROOT"

echo -e "${BLUE}What would you like to deploy?${NC}"
echo "  1) Superchain contracts (recommended for first deployment)"
echo "  2) Implementation contracts (requires existing superchain deployment)"
echo ""
read -r -p "Enter choice [1-2]: " DEPLOY_TYPE

echo ""
echo -e "${BLUE}━━━ Required Inputs ━━━${NC}"
echo ""

# Technically we could use a mainnet L1 here, this script is just for testing
if [ -z "$L1_RPC_URL" ]; then
    echo -e "${YELLOW}Sepolia RPC URL${NC}"
    echo "  Examples:"
    echo "    - https://sepolia.infura.io/v3/YOUR_KEY"
    echo "    - https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY"
    echo "    - https://rpc.sepolia.org (public, may be slow)"
    echo ""
    read -r -p "Enter Sepolia RPC URL: " L1_RPC_URL
else
    echo -e "${GREEN}✓ Using L1_RPC_URL from environment${NC}"
fi

if [ -z "$DEPLOYER_PRIVATE_KEY" ]; then
    echo ""
    echo -e "${YELLOW}Private Key${NC}"
    echo "  ⚠️  This account must have Sepolia ETH (~0.1-0.2 ETH recommended)"
    echo "  ⚠️  Never use mainnet keys or keys with real funds!"
    echo ""
    read -r -sp "Enter private key (hidden): " PRIVATE_KEY
    echo ""
else
    PRIVATE_KEY="$DEPLOYER_PRIVATE_KEY"
    echo ""
    echo -e "${GREEN}✓ Using DEPLOYER_PRIVATE_KEY from environment${NC}"
fi

if [ -z "$DEPLOYER_VERIFIER_TYPE" ]; then
    echo ""
    echo -e "${YELLOW}Contract Verification${NC}"
    echo "  Choose verifier(s) to use (or press Enter to skip verification):"
    echo "    1) Etherscan only"
    echo "    2) Blockscout only"
    echo "    3) Both Etherscan + Blockscout (recommended)"
    echo ""
    read -r -p "Enter choice [1-3 or Enter to skip]: " VERIFIER_CHOICE

    VERIFIER_TYPE=""
    ETHERSCAN_API_KEY="${DEPLOYER_VERIFIER_API_KEY}"

    if [ "$VERIFIER_CHOICE" == "1" ]; then
        VERIFIER_TYPE="etherscan"
        if [ -z "$ETHERSCAN_API_KEY" ]; then
            echo ""
            echo -e "${YELLOW}Etherscan API Key${NC}"
            echo "  Get one free at: https://etherscan.io/myapikey"
            echo ""
            read -r -p "Enter Etherscan API key: " ETHERSCAN_API_KEY
        fi
    elif [ "$VERIFIER_CHOICE" == "2" ]; then
        VERIFIER_TYPE="blockscout"
        echo ""
        echo -e "${GREEN}✓ Blockscout verification selected (no API key required)${NC}"
    elif [ "$VERIFIER_CHOICE" == "3" ]; then
        VERIFIER_TYPE="etherscan,blockscout"
        if [ -z "$ETHERSCAN_API_KEY" ]; then
            echo ""
            echo -e "${YELLOW}Etherscan API Key${NC}"
            echo "  Get one free at: https://etherscan.io/myapikey"
            echo ""
            read -r -p "Enter Etherscan API key: " ETHERSCAN_API_KEY
        fi
        echo ""
        echo -e "${GREEN}✓ Dual verification: Etherscan + Blockscout${NC}"
    fi
else
    VERIFIER_TYPE="$DEPLOYER_VERIFIER_TYPE"
    ETHERSCAN_API_KEY="${DEPLOYER_VERIFIER_API_KEY}"
    echo ""
    echo -e "${GREEN}✓ Using DEPLOYER_VERIFIER_TYPE from environment: $VERIFIER_TYPE${NC}"
fi

if [ "$DEPLOY_TYPE" == "1" ]; then
    echo ""
    echo -e "${BLUE}━━━ Superchain Configuration ━━━${NC}"
    echo ""
    echo -e "${YELLOW}Admin/Owner Addresses${NC}"
    echo "  You can use the same address for all roles for testing"
    echo ""
    
    if [ -z "$DEPLOYER_PROXY_ADMIN_OWNER" ]; then
        read -r -p "Superchain Proxy Admin Owner: " PROXY_ADMIN_OWNER
    else
        PROXY_ADMIN_OWNER="$DEPLOYER_PROXY_ADMIN_OWNER"
        echo -e "${GREEN}✓ Using DEPLOYER_PROXY_ADMIN_OWNER: $PROXY_ADMIN_OWNER${NC}"
    fi
    
    if [ -z "$DEPLOYER_PROTOCOL_VERSIONS_OWNER" ]; then
        read -r -p "Protocol Versions Owner: " PROTOCOL_VERSIONS_OWNER
    else
        PROTOCOL_VERSIONS_OWNER="$DEPLOYER_PROTOCOL_VERSIONS_OWNER"
        echo -e "${GREEN}✓ Using DEPLOYER_PROTOCOL_VERSIONS_OWNER: $PROTOCOL_VERSIONS_OWNER${NC}"
    fi
    
    if [ -z "$DEPLOYER_GUARDIAN" ]; then
        read -r -p "Guardian Address: " GUARDIAN
    else
        GUARDIAN="$DEPLOYER_GUARDIAN"
        echo -e "${GREEN}✓ Using DEPLOYER_GUARDIAN: $GUARDIAN${NC}"
    fi
    
    OUTPUT_FILE="$OUTPUT_DIR/sepolia-superchain-$(date +%Y%m%d-%H%M%S).json"
    
elif [ "$DEPLOY_TYPE" == "2" ]; then
    echo ""
    echo -e "${BLUE}━━━ Implementation Configuration ━━━${NC}"
    echo ""
    echo -e "${YELLOW}Required: Existing Superchain Addresses${NC}"
    echo "  These should be from a previous superchain deployment"
    echo ""
    
    if [ -z "$DEPLOYER_PROTOCOL_VERSIONS_PROXY" ] || [ "$DEPLOYER_PROTOCOL_VERSIONS_PROXY" == "null" ]; then
        read -r -p "Protocol Versions Proxy Address: " PROTOCOL_VERSIONS_PROXY
    else
        PROTOCOL_VERSIONS_PROXY="$DEPLOYER_PROTOCOL_VERSIONS_PROXY"
        echo -e "${GREEN}✓ Using DEPLOYER_PROTOCOL_VERSIONS_PROXY: $PROTOCOL_VERSIONS_PROXY${NC}"
    fi
    
    if [ -z "$DEPLOYER_SUPERCHAIN_CONFIG_PROXY" ] || [ "$DEPLOYER_SUPERCHAIN_CONFIG_PROXY" == "null" ]; then
        read -r -p "Superchain Config Proxy Address: " SUPERCHAIN_CONFIG_PROXY
    else
        SUPERCHAIN_CONFIG_PROXY="$DEPLOYER_SUPERCHAIN_CONFIG_PROXY"
        echo -e "${GREEN}✓ Using DEPLOYER_SUPERCHAIN_CONFIG_PROXY: $SUPERCHAIN_CONFIG_PROXY${NC}"
    fi
    
    if [ -z "$DEPLOYER_SUPERCHAIN_PROXY_ADMIN" ] || [ "$DEPLOYER_SUPERCHAIN_PROXY_ADMIN" == "null" ]; then
        read -r -p "Superchain Proxy Admin Address: " SUPERCHAIN_PROXY_ADMIN
    else
        SUPERCHAIN_PROXY_ADMIN="$DEPLOYER_SUPERCHAIN_PROXY_ADMIN"
        echo -e "${GREEN}✓ Using DEPLOYER_SUPERCHAIN_PROXY_ADMIN: $SUPERCHAIN_PROXY_ADMIN${NC}"
    fi
    
    if [ -z "$DEPLOYER_L1_PROXY_ADMIN_OWNER" ]; then
        read -r -p "L1 Proxy Admin Owner Address: " L1_PROXY_ADMIN_OWNER
    else
        L1_PROXY_ADMIN_OWNER="$DEPLOYER_L1_PROXY_ADMIN_OWNER"
        echo -e "${GREEN}✓ Using DEPLOYER_L1_PROXY_ADMIN_OWNER: $L1_PROXY_ADMIN_OWNER${NC}"
    fi
    
    if [ -z "$DEPLOYER_CHALLENGER" ]; then
        read -r -p "Challenger Address: " CHALLENGER
    else
        CHALLENGER="$DEPLOYER_CHALLENGER"
        echo -e "${GREEN}✓ Using DEPLOYER_CHALLENGER: $CHALLENGER${NC}"
    fi
    
    OUTPUT_FILE="$OUTPUT_DIR/sepolia-implementations-$(date +%Y%m%d-%H%M%S).json"
else
    echo -e "${RED}Invalid choice. Exiting.${NC}"
    exit 1
fi

# Confirmation
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Ready to deploy!${NC}"
echo ""
echo "  RPC URL: $L1_RPC_URL"
echo "  Output file: $OUTPUT_FILE"
if [ -n "$VERIFIER_TYPE" ]; then
    echo -e "  Verification: ${GREEN}Enabled${NC} ($VERIFIER_TYPE)"
else
    echo -e "  Verification: ${YELLOW}Disabled${NC}"
fi
echo ""
echo -e "${YELLOW}⚠️  This will deploy contracts to Sepolia and consume ETH for gas!${NC}"
echo ""
read -r -p "Continue? [y/N]: " CONFIRM

if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo -e "${RED}Deployment cancelled.${NC}"
    exit 0
fi

# Build the command
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🚀 Starting deployment...${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

CMD=("go" "run" "./cmd/op-deployer")

if [ "$DEPLOY_TYPE" == "1" ]; then
    CMD+=(
        "bootstrap" "superchain"
        "--l1-rpc-url" "$L1_RPC_URL"
        "--private-key" "$PRIVATE_KEY"
        "--outfile" "$OUTPUT_FILE"
        "--superchain-proxy-admin-owner" "$PROXY_ADMIN_OWNER"
        "--protocol-versions-owner" "$PROTOCOL_VERSIONS_OWNER"
        "--guardian" "$GUARDIAN"
    )
else
    CMD+=(
        "bootstrap" "implementations"
        "--l1-rpc-url" "$L1_RPC_URL"
        "--private-key" "$PRIVATE_KEY"
        "--outfile" "$OUTPUT_FILE"
        "--protocol-versions-proxy" "$PROTOCOL_VERSIONS_PROXY"
        "--superchain-config-proxy" "$SUPERCHAIN_CONFIG_PROXY"
        "--superchain-proxy-admin" "$SUPERCHAIN_PROXY_ADMIN"
        "--l1-proxy-admin-owner" "$L1_PROXY_ADMIN_OWNER"
        "--challenger" "$CHALLENGER"
        "--mips-version" "8"
    )
fi

if [ -n "$VERIFIER_TYPE" ]; then
    CMD+=(
        "--verify"
        "--verifier" "$VERIFIER_TYPE"
    )
    if [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && [ -n "$ETHERSCAN_API_KEY" ]; then
        CMD+=(
            "--verifier-api-key" "$ETHERSCAN_API_KEY"
        )
    fi
fi

# Execute the deployment
if "${CMD[@]}"; then
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✓ Deployment successful!${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${GREEN}Output saved to:${NC} $OUTPUT_FILE"
    echo ""
    
    if [ -f "$OUTPUT_FILE" ]; then
        echo -e "${BLUE}Deployed Addresses:${NC}"
        cat "$OUTPUT_FILE" | jq -r 'to_entries[] | "  \(.key): \(.value)"' 2>/dev/null || cat "$OUTPUT_FILE"
        echo ""
    fi
    
    if [ -n "$VERIFIER_TYPE" ]; then
        echo -e "${GREEN}✓ Contracts verified${NC}"
        if [[ "$VERIFIER_TYPE" == *"etherscan"* ]]; then
            echo "  Etherscan: https://sepolia.etherscan.io/"
        fi
        if [[ "$VERIFIER_TYPE" == *"blockscout"* ]]; then
            echo "  Blockscout: https://eth-sepolia.blockscout.com/"
        fi
    else
        echo -e "${YELLOW}ℹ  Run the verify command later to verify contracts:${NC}"
        echo ""
        echo "  go run ./cmd/op-deployer verify \\"
        echo "    --l1-rpc-url $L1_RPC_URL \\"
        echo "    --input-file $OUTPUT_FILE \\"
        echo "    --verifier etherscan,blockscout \\"
        echo "    --verifier-api-key YOUR_ETHERSCAN_API_KEY \\"
        echo "    --artifacts-locator embedded"
        echo ""
        echo "  (Verifies on both Etherscan and Blockscout)"
    fi
    
    if [ "$DEPLOY_TYPE" == "1" ] && [ -f "$OUTPUT_FILE" ]; then
        echo ""
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}📋 Copy these exports for implementations deployment:${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        
        PROTOCOL_VERSIONS_PROXY=$(jq -r '.protocolVersionsProxyAddress // .ProtocolVersionsProxyAddress' "$OUTPUT_FILE" 2>/dev/null)
        SUPERCHAIN_CONFIG_PROXY=$(jq -r '.superchainConfigProxyAddress // .SuperchainConfigProxyAddress' "$OUTPUT_FILE" 2>/dev/null)
        SUPERCHAIN_PROXY_ADMIN=$(jq -r '.proxyAdminAddress // .ProxyAdminAddress' "$OUTPUT_FILE" 2>/dev/null)
        
        echo "# Environment variables for next deployment"
        echo "export L1_RPC_URL=\"$L1_RPC_URL\""
        echo "export DEPLOYER_PRIVATE_KEY=\"$PRIVATE_KEY\""
        if [ -n "$VERIFIER_TYPE" ]; then
            echo "export DEPLOYER_VERIFIER_TYPE=\"$VERIFIER_TYPE\""
        fi
        if [ -n "$ETHERSCAN_API_KEY" ]; then
            echo "export DEPLOYER_VERIFIER_API_KEY=\"$ETHERSCAN_API_KEY\""
        fi
        echo "export DEPLOYER_PROTOCOL_VERSIONS_PROXY=\"$PROTOCOL_VERSIONS_PROXY\""
        echo "export DEPLOYER_SUPERCHAIN_CONFIG_PROXY=\"$SUPERCHAIN_CONFIG_PROXY\""
        echo "export DEPLOYER_SUPERCHAIN_PROXY_ADMIN=\"$SUPERCHAIN_PROXY_ADMIN\""
        echo "export DEPLOYER_L1_PROXY_ADMIN_OWNER=\"$PROXY_ADMIN_OWNER\""
        echo "export DEPLOYER_CHALLENGER=\"$GUARDIAN\"  # Using guardian as challenger"
        echo ""
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}Then run: ./scripts/test-sepolia-op-deployer.sh${NC}"
        echo -e "${GREEN}And select option 2 (Implementation contracts)${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    fi
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    
else
    echo ""
    echo -e "${RED}✗ Deployment failed!${NC}"
    echo ""
    echo "Check the error messages above for details."
    echo "Common issues:"
    echo "  - Insufficient Sepolia ETH in deployer account"
    echo "  - Invalid RPC URL"
    echo "  - Invalid private key format"
    echo "  - Network connectivity issues"
    echo ""
    exit 1
fi

