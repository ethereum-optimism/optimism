#!/usr/bin/env bash
# Shared compilation/verdict configuration. Source this from contracts-bedrock.
: "${CI_BRANCH:?CI_BRANCH must name the tested branch}"
CONTRACT_FEATURE="${CONTRACT_FEATURE:-main}"
case "${CONTRACT_FEATURE}" in
  main) FEATURE_ENV="" ;;
  CUSTOM_GAS_TOKEN) FEATURE_ENV=SYS_FEATURE__CUSTOM_GAS_TOKEN ;;
  OPTIMISM_PORTAL_INTEROP) FEATURE_ENV=DEV_FEATURE__OPTIMISM_PORTAL_INTEROP ;;
  ZK_DISPUTE_GAME) FEATURE_ENV=DEV_FEATURE__ZK_DISPUTE_GAME ;;
  *) echo "Unknown standard contract feature: ${CONTRACT_FEATURE}" >&2; exit 1 ;;
esac
for name in $(compgen -e); do
  case "${name}" in FOUNDRY_* | DAPP_* | DEV_FEATURE__* | SYS_FEATURE__*) unset "${name}" ;; esac
done
export CONTRACT_FEATURE
if [[ "${CI_BRANCH}" == develop ]]; then export FOUNDRY_PROFILE=ci; else export FOUNDRY_PROFILE=liteci; fi
if [[ -n "${FEATURE_ENV}" ]]; then export "${FEATURE_ENV}=true"; fi
export FORK_TEST=false L2_FORK_TEST=false L2CM_ACTIVATION_TEST=false
unset ETH_RPC_URL ETH_RPC_JWT ETH_RPC_HEADERS ETHERSCAN_API_KEY MAINNET_RPC_URL \
  FORK_RPC_URL FORK_BLOCK_NUMBER L2_FORK_RPC_URL L2_FORK_BLOCK_NUMBER
