#!/bin/sh

# RPC endpoint for the migration rehearsal network
export ETH_RPC_URL="https://mainnet-l1-rehearsal.optimism.io/"
# export ETH_RPC_URL="localhost:8545"

# Default HH key
HH_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
# Default HH addr
HH_ADDR="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

# SystemDictator contract (deployed by Maurelian on 2023-05-09)
MSD="0x49149a233de6E4cD6835971506F47EE5862289c1"

# ProxyAdmin contract
PROXY_ADMIN="0x43cA9bAe8dF108684E5EAaA720C25e1b32B0A075"

# AddressManager contract
ADDRESS_MANAGER="0xdE1FCfB0851916CA5101820A69b13a4E276bd81F"

# ResolvedDelegateProxy contract
RESOLVED_DELEGATE_PROXY="0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1"

# L1ChugSplashProxy contract
# use `setOwner(address)` for this one
L1_CHUG_SPLASH_PROXY="0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1"

# Proxy contract
# use `changeAdmin(address)` for this one
PROXY="0x5a7749f83b81B301cAb5f48EB8516B986DAef23D"

# OptimismPortal proxy
PORTAL_PROXY="0x59C4e2c6a6dC27c259D6d067a039c831e1ff4947"

# Check existing owners (should all be $HH_ADDR)
# cast call $PROXY_ADMIN "owner()(address)"
# cast call $ADDRESS_MANAGER "owner()(address)" 
# cast admin $L1_CHUG_SPLASH_PROXY 
# cast admin $PROXY

# ---
# Transfer ownership
# ---
# cast send --private-key $HH_KEY $PROXY_ADMIN "transferOwnership(address)" $MSD
# cast send --private-key $HH_KEY $ADDRESS_MANAGER "transferOwnership(address)" $MSD
# cast send --private-key $HH_KEY $L1_CHUG_SPLASH_PROXY "setOwner(address)" $MSD
# cast send --private-key $HH_KEY $PROXY "changeAdmin(address)" $MSD

# ---
# Execute Phase 1
# ---
# cast send --private-key $HH_KEY $MSD "phase1()"

# updateDynamicConfig signature
SIG="updateDynamicConfig((uint256,uint256),bool)"
# Encode calldata
CALLDATA=$(cast abi-encode $SIG "(17377105,1685641931)" true)
# Grab the selector
SELECTOR=$(cast sig $SIG)
# Prepare full payload
PAYLOAD=$(cast --concat-hex $SELECTOR $CALLDATA)

# Sanity check calldata
# cast pretty-calldata $PAYLOAD

# ---
# Update dynamic config
# ---
# cast send --private-key $HH_KEY $MSD $PAYLOAD

# ---
# !!!POINT OF NO RETURN!!!
# Execute phase 2
# ---
# cast send --private-key $HH_KEY $MSD "phase2()"

# ---
# Unpause the portal
# ---
# cast send --private-key $HH_KEY $PORTAL_PROXY "unpause()"

# ---
# Unpause Portal with CallForwarder
# ---

# Fetch portal guardian
# cast call $PORTAL_PROXY "GUARDIAN()(address)"

SIG="forward(address,bytes)"
FORWARD_SIG=$(cast sig $SIG)
CALLDATA=$(cast abi-encode $SIG $PORTAL_PROXY $(cast sig "unpause()"))
PAYLOAD=$(cast --concat-hex $FORWARD_SIG $CALLDATA)

# Sanity check calldata
# cast pretty-calldata $PAYLOAD

# Send unpause tx from multisig
# cast send "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A" $PAYLOAD --private-key $HH_KEY

# Check if paused
# cast call $PORTAL_PROXY "paused()(bool)"

# Check bytecode of multisig
# cast code "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A" --rpc-url https://mainnet-l1-rehearsal.optimism.io
