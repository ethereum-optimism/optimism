#!/bin/bash
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$REPO_ROOT/.deployer-output"
OP_DEPLOYER_BASE_CMD=("go" "run" "./cmd/op-deployer")
mkdir -p "$OUTPUT_DIR"
cd "$REPO_ROOT"

read_env_var() {
    local env_name="$1"
    local prompt="$2"
    local default="${3:-}"
    local hidden="${4:-}"
    local value="${!env_name}"
    
    if [ -z "$value" ] || [ "$value" == "null" ]; then
        if [ -n "$prompt" ]; then
            if [ "$hidden" == "-s" ]; then
                read -r -sp "$prompt" value
                echo ""
            else
                read -r -p "$prompt" value
            fi
        fi
        value="${value:-$default}"
    else
        echo -e "${GREEN}✓ Using $env_name${NC}" >&2
    fi
    echo "$value"
}

download_op_deployer_binary() {
    local version="$1"
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo -e "${RED}Unsupported arch: $arch${NC}" >&2; exit 1 ;;
    esac

    local name="op-deployer-${version}-${os}-${arch}"
    local url="https://github.com/ethereum-optimism/optimism/releases/download/op-deployer%2Fv${version}/${name}.tar.gz"
    local dest_dir
    dest_dir="$(mktemp -d "${TMPDIR:-/tmp}/op-deployer.XXXXXX")"
    local tar_path="$dest_dir/${name}.tar.gz"
    local extract_dir="$dest_dir/${name}"
    local bin_path="$extract_dir/op-deployer"

    echo -e "${BLUE}Downloading op-deployer ${version} (${os}/${arch})...${NC}"
    if ! curl -L --fail --retry 3 "$url" -o "$tar_path"; then
        echo -e "${RED}Download failed from:${NC} $url" >&2
        exit 1
    fi

    if ! tar -xzf "$tar_path" -C "$dest_dir"; then
        echo -e "${RED}Failed to extract:${NC} $tar_path" >&2
        exit 1
    fi

    if [ ! -x "$bin_path" ]; then
        echo -e "${RED}Binary not found after extract:${NC} $bin_path" >&2
        exit 1
    fi

    chmod +x "$bin_path"
    if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
        xattr -d com.apple.quarantine "$bin_path" 2>/dev/null || true
    fi

    OP_DEPLOYER_BASE_CMD=("$bin_path")
    echo -e "${GREEN}✓ Using downloaded binary:${NC} $bin_path"
}

select_verifier() {
    local context="$1"
    local prefill_type="${DEPLOYER_VERIFIER_TYPE:-}"
    local prefill_key="${DEPLOYER_VERIFIER_API_KEY:-}"
    
    echo ""
    echo -e "${YELLOW}Choose verifier(s) for $context:${NC}"
    echo "    1) Etherscan only"
    echo "    2) Blockscout only"
    echo "    3) Both Etherscan + Blockscout"
    echo ""
    
    local choice=""
    if [ -n "$prefill_type" ]; then
        if [ "$prefill_type" == "etherscan" ]; then
            choice="1"
        elif [ "$prefill_type" == "blockscout" ]; then
            choice="2"
        elif [[ "$prefill_type" == *"etherscan"* ]] && [[ "$prefill_type" == *"blockscout"* ]]; then
            choice="3"
        fi
        if [ -n "$choice" ]; then
            echo -e "${GREEN}  (Pre-filled from DEPLOYER_VERIFIER_TYPE: $prefill_type - auto-selecting option $choice)${NC}"
        fi
    fi
    
    if [ -z "$choice" ]; then
        read -r -p "Enter choice [1-3]: " choice
    else
        echo "  Using choice: $choice"
    fi
    
    local verifier_type=""
    local api_key="$prefill_key"
    
    case "$choice" in
        1)
            verifier_type="etherscan"
            if [ -z "$api_key" ]; then
                echo ""
                echo -e "${YELLOW}Etherscan API Key${NC}"
                echo "  Get one free at: https://etherscan.io/myapikey"
                echo ""
                read -r -p "Enter Etherscan API key: " api_key
            fi
            ;;
        2)
            verifier_type="blockscout"
            echo ""
            echo -e "${GREEN}✓ Blockscout verification selected${NC}"
            ;;
        3)
            verifier_type="etherscan,blockscout"
            if [ -z "$api_key" ]; then
                echo ""
                echo -e "${YELLOW}Etherscan API Key${NC}"
                echo "  Get one free at: https://etherscan.io/myapikey"
                echo ""
                read -r -p "Enter Etherscan API key: " api_key
            fi
            echo ""
            echo -e "${GREEN}✓ Dual verification: Etherscan + Blockscout${NC}"
            ;;
        *)
            echo -e "${RED}Invalid choice. Exiting.${NC}"
            exit 1
            ;;
    esac
    
    VERIFIER_TYPE="$verifier_type"
    ETHERSCAN_API_KEY="$api_key"
}

build_verify_cmd() {
    local input_file="$1"
    local cmd=("${OP_DEPLOYER_BASE_CMD[@]}" "verify"
        "--l1-rpc-url" "$L1_RPC_URL"
        "--input-file" "$input_file"
        "--verifier" "$VERIFIER_TYPE"
        "--artifacts-locator" "embedded"
    )
    if [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && [ -n "$ETHERSCAN_API_KEY" ]; then
        cmd+=("--verifier-api-key" "$ETHERSCAN_API_KEY")
    fi
    echo "${cmd[@]}"
}

build_validate_cmd() {
    local workdir="$1"
    local cmd=("${OP_DEPLOYER_BASE_CMD[@]}" "validate" "auto"
        "--l1-rpc-url" "$L1_RPC_URL"
        "--workdir" "$workdir"
        "--fail"
    )
    if [ -n "${DEPLOYER_L2_CHAIN_ID:-}" ]; then
        cmd+=("${DEPLOYER_L2_CHAIN_ID}")
    fi
    echo "${cmd[@]}"
}

update_toml_field() {
    local file="$1"
    local field="$2"
    local value="$3"
    
    if grep -q "^$field = " "$file" 2>/dev/null; then
        if ! sed -i.bak "s|^$field = .*|$field = \"$value\"|g" "$file" 2>/dev/null; then
            if sed "s|^$field = .*|$field = \"$value\"|g" "$file" > "${file}.tmp" 2>/dev/null && [ -f "${file}.tmp" ]; then
                mv "${file}.tmp" "$file"
            fi
        fi
        rm -f "${file}.bak" 2>/dev/null
    else
        if ! sed -i.bak "/^configType = /a\\
$field = \"$value\"
" "$file" 2>/dev/null; then
            if sed "/^configType = /a\\
$field = \"$value\"
" "$file" > "${file}.tmp" 2>/dev/null && [ -f "${file}.tmp" ]; then
                mv "${file}.tmp" "$file"
            fi
        fi
        rm -f "${file}.bak" 2>/dev/null
    fi
}

select_runner() {
    local prefill="${DEPLOYER_RUNNER:-}"
    local prefill_choice=""
    if [ "$prefill" == "docker" ]; then
        prefill_choice="2"
        echo -e "${GREEN}  (Pre-filled from DEPLOYER_RUNNER: docker - auto-selecting option 2)${NC}"
    elif [ "$prefill" == "go" ]; then
        prefill_choice="1"
        echo -e "${GREEN}  (Pre-filled from DEPLOYER_RUNNER: go - auto-selecting option 1)${NC}"
    elif [ "$prefill" == "binary" ]; then
        prefill_choice="3"
        echo -e "${GREEN}  (Pre-filled from DEPLOYER_RUNNER: binary - auto-selecting option 3)${NC}"
    fi
    
    echo ""
    echo -e "${BLUE}How would you like to run op-deployer?${NC}"
    echo "  1) Local go run (default)"
    echo "  2) Docker image"
    echo "  3) Prebuilt binary (download from GitHub release)"
    echo ""
    
    local choice="$prefill_choice"
    if [ -z "$choice" ]; then
        read -r -p "Enter choice [1-3]: " choice
    fi
    
    case "$choice" in
        2)
            local default_tag="${DEPLOYER_IMAGE_TAG:-latest}"
            if [ -n "${DEPLOYER_IMAGE:-}" ]; then
                OP_DEPLOYER_IMAGE="$DEPLOYER_IMAGE"
            else
                read -r -p "Docker image tag [${default_tag}]: " selected_tag
                selected_tag="${selected_tag:-$default_tag}"
                OP_DEPLOYER_IMAGE="us-docker.pkg.dev/oplabs-tools-artifacts/images/op-deployer:${selected_tag}"
            fi
            OP_DEPLOYER_BASE_CMD=("docker" "run" "--rm" "-v" "$REPO_ROOT:$REPO_ROOT" "-w" "$REPO_ROOT" "$OP_DEPLOYER_IMAGE" "op-deployer")
            echo -e "${GREEN}✓ Using docker image:${NC} $OP_DEPLOYER_IMAGE"
            ;;
        3)
            local default_version="${DEPLOYER_VERSION:-0.5.1}"
            read -r -p "Release version [${default_version}]: " selected_version
            selected_version="${selected_version:-$default_version}"
            download_op_deployer_binary "$selected_version"
            ;;
        *)
            OP_DEPLOYER_BASE_CMD=("go" "run" "./cmd/op-deployer")
            echo -e "${GREEN}✓ Using local go run (./cmd/op-deployer)${NC}"
            ;;
    esac
}

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}   OP Deployer - Sepolia Deployment & Verification Script${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${BLUE}What would you like to do?${NC}"
echo "  1) Deploy superchain contracts (recommended for first deployment)"
echo "  2) Deploy implementation contracts (requires existing superchain deployment)"
echo "  3) Apply chain deployment (deploy L2 chain using intent.toml)"
echo "  4) Verify previous deployment (using state/bootstrap output file)"
echo "  5) Validate deployment (using state.json and chain-id)"
echo ""
read -r -p "Enter choice [1-5]: " DEPLOY_TYPE

echo ""
echo -e "${BLUE}━━━ Required Inputs ━━━${NC}"
echo ""

L1_RPC_URL=$(read_env_var "L1_RPC_URL" "Enter Sepolia RPC URL: ")

select_runner

if [ "$DEPLOY_TYPE" != "4" ] && [ "$DEPLOY_TYPE" != "5" ]; then
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
fi

if [ "$DEPLOY_TYPE" != "3" ] && [ "$DEPLOY_TYPE" != "5" ]; then
    echo ""
    echo -e "${YELLOW}Contract Verification${NC}"
    echo "  How would you like to verify contracts?"
    echo "    1) Auto-verify during deployment (--verify flag)"
    echo "    2) Verify after deployment using state file"
    echo "    3) Skip verification"
    echo ""
    read -r -p "Enter choice [1-3]: " VERIFY_METHOD_CHOICE
    
    AUTO_VERIFY=false
    POST_DEPLOY_VERIFY=false
    
    if [ "$VERIFY_METHOD_CHOICE" == "1" ]; then
        AUTO_VERIFY=true
        select_verifier "auto-verification"
    elif [ "$VERIFY_METHOD_CHOICE" == "2" ]; then
        POST_DEPLOY_VERIFY=true
        select_verifier "post-deployment verification"
    else
        echo ""
        echo -e "${YELLOW}✓ Verification skipped${NC}"
    fi
fi

case "$DEPLOY_TYPE" in
    1)
        echo ""
        echo -e "${BLUE}━━━ Superchain Configuration ━━━${NC}"
        echo ""
        echo -e "${YELLOW}Admin/Owner Addresses${NC}"
        echo "  You can use the same address for all roles for testing"
        echo ""
        
        PROXY_ADMIN_OWNER=$(read_env_var "DEPLOYER_PROXY_ADMIN_OWNER" "Superchain Proxy Admin Owner: ")
        PROTOCOL_VERSIONS_OWNER=$(read_env_var "DEPLOYER_PROTOCOL_VERSIONS_OWNER" "Protocol Versions Owner: ")
        GUARDIAN=$(read_env_var "DEPLOYER_GUARDIAN" "Guardian Address: ")
        
        OUTPUT_FILE="$OUTPUT_DIR/sepolia-superchain-$(date +%Y%m%d-%H%M%S).json"
        ;;
    2)
        echo ""
        echo -e "${BLUE}━━━ Implementation Configuration ━━━${NC}"
        echo ""
        echo -e "${YELLOW}Required: Existing Superchain Addresses${NC}"
        echo "  These should be from a previous superchain deployment"
        echo ""
        
        PROTOCOL_VERSIONS_PROXY=$(read_env_var "DEPLOYER_PROTOCOL_VERSIONS_PROXY" "Protocol Versions Proxy Address: ")
        SUPERCHAIN_CONFIG_PROXY=$(read_env_var "DEPLOYER_SUPERCHAIN_CONFIG_PROXY" "Superchain Config Proxy Address: ")
        SUPERCHAIN_PROXY_ADMIN=$(read_env_var "DEPLOYER_SUPERCHAIN_PROXY_ADMIN" "Superchain Proxy Admin Address: ")
        L1_PROXY_ADMIN_OWNER=$(read_env_var "DEPLOYER_L1_PROXY_ADMIN_OWNER" "L1 Proxy Admin Owner Address: ")
        CHALLENGER=$(read_env_var "DEPLOYER_CHALLENGER" "Challenger Address: ")
        
        OUTPUT_FILE="$OUTPUT_DIR/sepolia-implementations-$(date +%Y%m%d-%H%M%S).json"
        ;;
    3)
        echo ""
        echo -e "${BLUE}━━━ Apply Chain Deployment ━━━${NC}"
        echo ""
        
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
        WORKDIR=$(read_env_var "DEPLOYER_WORKDIR" "Enter workdir path [.deployer]: " ".deployer")
        
        if [ -z "${DEPLOYER_L2_CHAIN_ID:-}" ]; then
            echo ""
            echo -e "${YELLOW}L2 Chain ID${NC}"
            echo "  Example: 0xAA37DC (11155420 in decimal)"
            echo "  Options:"
            echo "    1) Enter a custom chain ID"
            echo "    2) Generate a random chain ID"
            echo ""
            read -r -p "Enter choice [1-2]: " CHAIN_ID_CHOICE
            
            if [ "$CHAIN_ID_CHOICE" == "2" ]; then
                if command -v openssl >/dev/null 2>&1; then
                    RANDOM_BYTES=$(openssl rand -hex 3)
                elif [ -c /dev/urandom ]; then
                    RANDOM_BYTES=$(head -c 3 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 6)
                else
                    RANDOM_BYTES=$(printf "%06X" $((RANDOM * 256 + RANDOM)))
                fi
                L2_CHAIN_ID="0x$(echo "$RANDOM_BYTES" | tr '[:lower:]' '[:upper:]')"
                echo ""
                echo -e "${GREEN}✓ Generated random L2 chain ID: $L2_CHAIN_ID${NC}"
            else
                echo ""
                read -r -p "Enter L2 chain ID: " L2_CHAIN_ID
            fi
        else
            L2_CHAIN_ID="$DEPLOYER_L2_CHAIN_ID"
            echo ""
            echo -e "${GREEN}✓ Using DEPLOYER_L2_CHAIN_ID: $L2_CHAIN_ID${NC}"
        fi
        
        L1_CHAIN_ID="${DEPLOYER_L1_CHAIN_ID:-11155111}"
        
        OPCM_ADDRESS=""
        INTENT_TYPE="standard"
        
        if [ -d "$OUTPUT_DIR" ]; then
            IMPL_FILE=$(find "$OUTPUT_DIR" -maxdepth 1 -name "sepolia-implementations-*.json" -type f -printf '%T@\t%p\n' 2>/dev/null | sort -rn | head -n 1 | cut -f2-)
            if [ -n "$IMPL_FILE" ] && [ -f "$IMPL_FILE" ]; then
                OPCM_ADDRESS=$(jq -r '.opcmAddress // .OPCMAddress // empty' "$IMPL_FILE" 2>/dev/null)
                if [ -n "$OPCM_ADDRESS" ] && [ "$OPCM_ADDRESS" != "null" ] && [ "$OPCM_ADDRESS" != "" ]; then
                    echo ""
                    echo -e "${GREEN}✓ Found bootstrap implementations output${NC}"
                    echo "  OPCM Address: $OPCM_ADDRESS"
                    echo "  Will use 'standard-overrides' intent type with your OPCM"
                    INTENT_TYPE="standard-overrides"
                fi
            fi
        fi
        
        if [ -z "$OPCM_ADDRESS" ] || [ "$OPCM_ADDRESS" == "null" ]; then
            echo ""
            echo -e "${YELLOW}OPCM Configuration${NC}"
            echo "  Standard intent uses pre-deployed OPCM from registry"
            echo "  If you deployed implementations (option 2), enter your OPCM address"
            echo "  Leave empty to use standard registry OPCM"
            echo ""
            read -r -p "OPCM Address (optional): " OPCM_INPUT
            if [ -n "$OPCM_INPUT" ] && [ "$OPCM_INPUT" != "" ]; then
                OPCM_ADDRESS="$OPCM_INPUT"
                INTENT_TYPE="standard-overrides"
            fi
        fi
        
        SKIP_INIT=false
        if [ -f "$WORKDIR/intent.toml" ]; then
            echo ""
            echo -e "${YELLOW}⚠️  intent.toml already exists in $WORKDIR${NC}"
            read -r -p "Re-initialize? This will overwrite existing files [y/N]: " REINIT
            [[ ! "$REINIT" =~ ^[Yy]$ ]] && SKIP_INIT=true
        fi
        
        if [ "$SKIP_INIT" != "true" ]; then
            echo ""
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${GREEN}📝 Initializing intent and state files...${NC}"
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            
            INIT_CMD=("${OP_DEPLOYER_BASE_CMD[@]}" "init"
                "--l1-chain-id" "$L1_CHAIN_ID"
                "--l2-chain-ids" "$L2_CHAIN_ID"
                "--workdir" "$WORKDIR"
                "--intent-type" "$INTENT_TYPE"
            )
            
            if ! "${INIT_CMD[@]}"; then
                echo ""
                echo -e "${RED}✗ Initialization failed!${NC}"
                exit 1
            fi
            
            echo ""
            echo -e "${GREEN}✓ Intent and state files created${NC}"
            
            if [ "$INTENT_TYPE" == "standard-overrides" ] && [ -n "$OPCM_ADDRESS" ] && [ -f "$WORKDIR/intent.toml" ]; then
                echo ""
                echo -e "${GREEN}✓ Setting OPCM address in intent.toml${NC}"
                update_toml_field "$WORKDIR/intent.toml" "opcmAddress" "$OPCM_ADDRESS"
                echo "  OPCM Address: $OPCM_ADDRESS"
            fi
            
            if [ -f "$WORKDIR/intent.toml" ]; then
                DEPLOYER_ADDRESS=$(cast wallet address --private-key "$PRIVATE_KEY" 2>/dev/null | tail -n 1 | tr -d ' ')
                
                REQUIRED_FIELDS=(
                    "systemConfigOwner:SystemConfigOwner"
                    "unsafeBlockSigner:UnsafeBlockSigner"
                    "batcher:Batcher"
                    "proposer:Proposer"
                    "baseFeeVaultRecipient:BaseFeeVaultRecipient"
                    "l1FeeVaultRecipient:L1FeeVaultRecipient"
                    "sequencerFeeVaultRecipient:SequencerFeeVaultRecipient"
                    "operatorFeeVaultRecipient:OperatorFeeVaultRecipient"
                )
                
                if grep -q 'useRevenueShare = true' "$WORKDIR/intent.toml" 2>/dev/null; then
                    if grep -q 'chainFeesRecipient = "0x0000000000000000000000000000000000000000"' "$WORKDIR/intent.toml" 2>/dev/null || ! grep -q 'chainFeesRecipient' "$WORKDIR/intent.toml" 2>/dev/null; then
                        REQUIRED_FIELDS+=("chainFeesRecipient:ChainFeesRecipient")
                    fi
                fi
                
                NEEDS_FIX=false
                for field_info in "${REQUIRED_FIELDS[@]}"; do
                    field_name="${field_info%%:*}"
                    if grep -q "$field_name = \"0x0000000000000000000000000000000000000000\"" "$WORKDIR/intent.toml" 2>/dev/null; then
                        NEEDS_FIX=true
                        break
                    fi
                done
                
                if [ "$NEEDS_FIX" == "true" ]; then
                    echo ""
                    echo -e "${YELLOW}Some required addresses are not set in intent.toml${NC}"
                    echo "  These addresses are required for chain deployment"
                    
                    USE_DEPLOYER_FOR_ALL=""
                    if [ -n "$DEPLOYER_ADDRESS" ]; then
                        echo -e "${GREEN}  Deployer address: $DEPLOYER_ADDRESS${NC}"
                        echo ""
                        read -r -p "Use deployer address for all required roles? [Y/n]: " USE_DEPLOYER_FOR_ALL
                    fi
                    
                    for field_info in "${REQUIRED_FIELDS[@]}"; do
                        field_name="${field_info%%:*}"
                        field_display="${field_info##*:}"
                        
                        FIELD_NEEDS_FIX=false
                        if grep -q "$field_name = \"0x0000000000000000000000000000000000000000\"" "$WORKDIR/intent.toml" 2>/dev/null; then
                            FIELD_NEEDS_FIX=true
                        elif [ "$field_name" == "chainFeesRecipient" ] && ! grep -q "$field_name" "$WORKDIR/intent.toml" 2>/dev/null; then
                            FIELD_NEEDS_FIX=true
                        fi
                        
                        if [ "$FIELD_NEEDS_FIX" == "true" ]; then
                            ADDRESS_TO_USE=""
                            
                            if [[ "$USE_DEPLOYER_FOR_ALL" != "n" && "$USE_DEPLOYER_FOR_ALL" != "N" ]] && [ -n "$DEPLOYER_ADDRESS" ]; then
                                ADDRESS_TO_USE="$DEPLOYER_ADDRESS"
                            else
                                echo ""
                                read -r -p "Enter $field_display address: " ADDRESS_TO_USE
                            fi
                            
                            if [ -n "$ADDRESS_TO_USE" ]; then
                                if grep -q "$field_name = " "$WORKDIR/intent.toml" 2>/dev/null; then
                                    if ! sed -i.bak "s/$field_name = \"0x0000000000000000000000000000000000000000\"/$field_name = \"$ADDRESS_TO_USE\"/g" "$WORKDIR/intent.toml" 2>/dev/null; then
                                        if sed "s/$field_name = \"0x0000000000000000000000000000000000000000\"/$field_name = \"$ADDRESS_TO_USE\"/g" "$WORKDIR/intent.toml" > "${WORKDIR}/intent.toml.tmp" 2>/dev/null && [ -f "${WORKDIR}/intent.toml.tmp" ]; then
                                            mv "${WORKDIR}/intent.toml.tmp" "$WORKDIR/intent.toml"
                                        fi
                                    fi
                                    rm -f "${WORKDIR}/intent.toml.bak" 2>/dev/null
                                    echo -e "${GREEN}✓ Set $field_display to: $ADDRESS_TO_USE${NC}"
                                else
                                    if grep -q 'useRevenueShare = true' "$WORKDIR/intent.toml" 2>/dev/null; then
                                        if ! sed -i.bak "/useRevenueShare = true/a\\
  $field_name = \"$ADDRESS_TO_USE\"
" "$WORKDIR/intent.toml" 2>/dev/null; then
                                            if sed "/useRevenueShare = true/a\\
  $field_name = \"$ADDRESS_TO_USE\"
" "$WORKDIR/intent.toml" > "${WORKDIR}/intent.toml.tmp" 2>/dev/null && [ -f "${WORKDIR}/intent.toml.tmp" ]; then
                                                mv "${WORKDIR}/intent.toml.tmp" "$WORKDIR/intent.toml"
                                            fi
                                        fi
                                        rm -f "${WORKDIR}/intent.toml.bak" 2>/dev/null
                                        echo -e "${GREEN}✓ Added $field_display: $ADDRESS_TO_USE${NC}"
                                    else
                                        if ! sed -i.bak "/\[\[chains\]\]/,/^\[\[/ { /operatorFeeVaultRecipient = /a\\
  $field_name = \"$ADDRESS_TO_USE\"
}" "$WORKDIR/intent.toml" 2>/dev/null; then
                                            echo -e "${YELLOW}⚠️  Could not automatically add $field_display. Please add it manually to intent.toml${NC}"
                                        fi
                                        rm -f "${WORKDIR}/intent.toml.bak" 2>/dev/null
                                    fi
                                fi
                            fi
                        fi
                    done
                fi
            fi
        fi
        
        AUTO_VALIDATE_ENABLED=false
        if [ -n "${DEPLOYER_AUTO_VALIDATE:-}" ]; then
            AUTO_VALIDATE_ENABLED=true
            echo ""
            echo -e "${GREEN}✓ Auto-validation enabled${NC}"
        else
            echo ""
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${YELLOW}🔍 Validate after deployment?${NC}"
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            read -r -p "Validate deployment after apply? [y/N]: " VALIDATE_AFTER
            [[ "$VALIDATE_AFTER" =~ ^[Yy]$ ]] && AUTO_VALIDATE_ENABLED=true
        fi
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}Ready to apply chain deployment!${NC}"
        echo ""
        echo "  RPC URL: $L1_RPC_URL"
        echo "  Workdir: $WORKDIR"
        echo "  L1 Chain ID: $L1_CHAIN_ID"
        echo "  L2 Chain ID: $L2_CHAIN_ID"
        if [ "$AUTO_VALIDATE_ENABLED" == "true" ]; then
            echo -e "  Validation: ${GREEN}Enabled (auto-detect version and chain ID)${NC}"
        else
            echo -e "  Validation: ${YELLOW}Disabled${NC}"
        fi
        echo ""
        echo -e "${YELLOW}⚠️  This will deploy chain contracts to Sepolia and consume ETH for gas!${NC}"
        echo ""
        read -r -p "Continue? [y/N]: " CONFIRM
        
        if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
            echo -e "${RED}Deployment cancelled.${NC}"
            exit 0
        fi
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}🚀 Starting apply...${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${YELLOW}Note:${NC} If you see 'ReadImplementationAddresses' errors, this usually means:"
        echo "  - Implementation contracts need to be deployed first (run option 2: bootstrap implementations)"
        echo "  - Or there's an issue with the OPCM contract configuration"
        echo ""
        
        APPLY_CMD=("${OP_DEPLOYER_BASE_CMD[@]}" "apply"
            "--l1-rpc-url" "$L1_RPC_URL"
            "--workdir" "$WORKDIR"
            "--private-key" "$PRIVATE_KEY"
            "--deployment-target" "live"
        )
        
        [ "$AUTO_VALIDATE_ENABLED" == "true" ] && APPLY_CMD+=("--validate" "auto")
        
        if "${APPLY_CMD[@]}"; then
            echo ""
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
            echo -e "${GREEN}✓ Apply successful!${NC}"
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
            echo ""
            echo "  State file: $WORKDIR/state.json"
            echo "  Intent file: $WORKDIR/intent.toml"
        else
            echo ""
            echo -e "${RED}✗ Apply failed!${NC}"
            echo ""
            echo "Check the error messages above for details."
            exit 1
        fi
        exit 0
        ;;
    4)
        echo ""
        echo -e "${BLUE}━━━ Verification Configuration ━━━${NC}"
        echo ""
        
        STATE_FILE=$(read_env_var "DEPLOYER_STATE_FILE" "Enter path to state/bootstrap output file: ")
        
        if [ ! -f "$STATE_FILE" ]; then
            echo -e "${RED}✗ File not found: $STATE_FILE${NC}"
            exit 1
        fi
        
        select_verifier "verification"
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}Ready to verify!${NC}"
        echo ""
        echo "  RPC URL: $L1_RPC_URL"
        echo "  Input file: $STATE_FILE"
        echo -e "  Verification: ${GREEN}$VERIFIER_TYPE${NC}"
        echo ""
        read -r -p "Continue? [y/N]: " CONFIRM
        
        if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
            echo -e "${RED}Verification cancelled.${NC}"
            exit 0
        fi
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}🔍 Starting verification...${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        
        read -ra VERIFY_CMD <<< "$(build_verify_cmd "$STATE_FILE")"
        
        if "${VERIFY_CMD[@]}"; then
            echo ""
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
            echo -e "${GREEN}✓ Verification complete!${NC}"
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
            echo ""
            [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && echo "  Etherscan: https://sepolia.etherscan.io/"
            [[ "$VERIFIER_TYPE" == *"blockscout"* ]] && echo "  Blockscout: https://eth-sepolia.blockscout.com/"
        else
            echo ""
            echo -e "${RED}✗ Verification failed!${NC}"
            echo ""
            echo "Check the error messages above for details."
            exit 1
        fi
        exit 0
        ;;
    5)
        echo ""
        echo -e "${BLUE}━━━ Validation Configuration ━━━${NC}"
        echo ""
        
        WORKDIR=$(read_env_var "DEPLOYER_WORKDIR" "Enter workdir path [.deployer]: " ".deployer")
        
        if [ ! -d "$WORKDIR" ]; then
            echo -e "${RED}✗ Directory not found: $WORKDIR${NC}"
            exit 1
        fi
        
        STATE_FILE="$WORKDIR/state.json"
        if [ ! -f "$STATE_FILE" ]; then
            echo -e "${RED}✗ state.json not found in: $WORKDIR${NC}"
            exit 1
        fi
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}Ready to validate!${NC}"
        echo ""
        echo "  RPC URL: $L1_RPC_URL"
        echo "  Workdir: $WORKDIR"
        echo "  Version: Auto-detect from state.json"
        echo "  Chain ID: Auto-detect from state.json"
        echo ""
        read -r -p "Continue? [y/N]: " CONFIRM
        
        if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
            echo -e "${RED}Validation cancelled.${NC}"
            exit 0
        fi
        
        echo ""
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}🔍 Starting validation...${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        
        read -ra VALIDATE_CMD <<< "$(build_validate_cmd "$WORKDIR")"
        
        if "${VALIDATE_CMD[@]}"; then
            echo ""
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
            echo -e "${GREEN}✓ Validation passed!${NC}"
            echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
        else
            echo ""
            echo -e "${RED}✗ Validation failed!${NC}"
            echo ""
            echo "Check the error messages above for details."
            exit 1
        fi
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice. Exiting.${NC}"
        exit 1
        ;;
esac

if [ "$DEPLOY_TYPE" == "1" ] || [ "$DEPLOY_TYPE" == "2" ]; then
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Ready to deploy!${NC}"
    echo ""
    echo "  RPC URL: $L1_RPC_URL"
    echo "  Output file: $OUTPUT_FILE"
    if [ "$AUTO_VERIFY" == "true" ]; then
        echo -e "  Verification: ${GREEN}Auto-verify during deployment${NC} ($VERIFIER_TYPE)"
    elif [ "$POST_DEPLOY_VERIFY" == "true" ]; then
        echo -e "  Verification: ${GREEN}Post-deployment using state file${NC} ($VERIFIER_TYPE)"
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
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}🚀 Starting deployment...${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    
    CMD=("${OP_DEPLOYER_BASE_CMD[@]}")
    
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
    
    if [ "$AUTO_VERIFY" == "true" ] && [ -n "$VERIFIER_TYPE" ]; then
        CMD+=("--verify" "--verifier" "$VERIFIER_TYPE")
        [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && [ -n "$ETHERSCAN_API_KEY" ] && CMD+=("--verifier-api-key" "$ETHERSCAN_API_KEY")
    fi
    
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
            jq -r 'to_entries[] | "  \(.key): \(.value)"' "$OUTPUT_FILE" 2>/dev/null || cat "$OUTPUT_FILE"
            echo ""
        fi
        
        if [ "$AUTO_VERIFY" == "true" ] && [ -n "$VERIFIER_TYPE" ]; then
            echo -e "${GREEN}✓ Contracts verified during deployment${NC}"
            [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && echo "  Etherscan: https://sepolia.etherscan.io/"
            [[ "$VERIFIER_TYPE" == *"blockscout"* ]] && echo "  Blockscout: https://eth-sepolia.blockscout.com/"
        elif [ "$POST_DEPLOY_VERIFY" == "true" ] && [ -n "$VERIFIER_TYPE" ]; then
            echo ""
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${YELLOW}🔍 Verifying contracts using bootstrap output...${NC}"
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            
            read -ra VERIFY_CMD <<< "$(build_verify_cmd "$OUTPUT_FILE")"
            
            if "${VERIFY_CMD[@]}"; then
                echo ""
                echo -e "${GREEN}✓ Verification complete!${NC}"
                [[ "$VERIFIER_TYPE" == *"etherscan"* ]] && echo "  Etherscan: https://sepolia.etherscan.io/"
                [[ "$VERIFIER_TYPE" == *"blockscout"* ]] && echo "  Blockscout: https://eth-sepolia.blockscout.com/"
            else
                echo ""
                echo -e "${YELLOW}⚠  Verification had some issues (check output above)${NC}"
            fi
        else
            echo -e "${YELLOW}ℹ  Verification was skipped${NC}"
        fi
        
        if [ "$DEPLOY_TYPE" != "1" ] && [ "$DEPLOY_TYPE" != "2" ]; then
            if [ -f ".deployer/state.json" ] || [ -f "state.json" ]; then
                WORKDIR_FOR_VALIDATION=""
                [ -f ".deployer/state.json" ] && WORKDIR_FOR_VALIDATION=".deployer"
                [ -f "state.json" ] && WORKDIR_FOR_VALIDATION="."
                
                AUTO_VALIDATE_ENABLED=false
                
                if [ -n "${DEPLOYER_AUTO_VALIDATE:-}" ]; then
                    AUTO_VALIDATE_ENABLED=true
                    echo ""
                    echo -e "${GREEN}✓ Auto-validation enabled (will auto-detect version and chain ID from state.json)${NC}"
                else
                    echo ""
                    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
                    echo -e "${YELLOW}🔍 Would you like to validate the deployment?${NC}"
                    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
                    echo ""
                    echo "  Validation checks that the deployment matches standard configuration"
                    echo "  Version and chain ID will be auto-detected from state.json"
                    echo ""
                    read -r -p "Validate deployment? [y/N]: " VALIDATE_NOW
                    [[ "$VALIDATE_NOW" =~ ^[Yy]$ ]] && AUTO_VALIDATE_ENABLED=true
                fi
                
                if [ "$AUTO_VALIDATE_ENABLED" == "true" ]; then
                    echo ""
                    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
                    echo -e "${GREEN}🔍 Running validation...${NC}"
                    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
                    echo ""
                    
                    read -ra VALIDATE_CMD <<< "$(build_validate_cmd "$WORKDIR_FOR_VALIDATION")"
                    
                    if "${VALIDATE_CMD[@]}"; then
                        echo ""
                        echo -e "${GREEN}✓ Validation passed!${NC}"
                    else
                        echo ""
                        echo -e "${YELLOW}⚠  Validation found issues (check output above)${NC}"
                    fi
                fi
            fi
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
            [ -n "$VERIFIER_TYPE" ] && echo "export DEPLOYER_VERIFIER_TYPE=\"$VERIFIER_TYPE\""
            [ -n "$ETHERSCAN_API_KEY" ] && echo "export DEPLOYER_VERIFIER_API_KEY=\"$ETHERSCAN_API_KEY\""
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
fi
