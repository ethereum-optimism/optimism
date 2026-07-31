# shellcheck shell=bash
# Source from CI recipes that need historical L1 state.
export SEPOLIA_RPC_URL="${SEPOLIA_RPC_URL:-${OP_CI_SEPOLIA_L1_ARCHIVE_RPC_URL:?OP_CI_SEPOLIA_L1_ARCHIVE_RPC_URL must be set}}"
export MAINNET_RPC_URL="${MAINNET_RPC_URL:-${OP_CI_MAINNET_L1_ARCHIVE_RPC_URL:?OP_CI_MAINNET_L1_ARCHIVE_RPC_URL must be set}}"
