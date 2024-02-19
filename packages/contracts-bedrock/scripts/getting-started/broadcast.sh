#!/usr/bin/env bash

# This script is used to generate the four wallets that are used in the Getting
# Started quickstart guide on the docs site. Simplifies things for users
# slightly while also avoiding the need for users to manually copy/paste a
# bunch of stuff over to the environment file.

# Generate wallets
wallet1=$(cast wallet new)

# Grab wallet addresses
address1=$(echo "$wallet1" | awk '/Address/ { print $2 }')

# Grab wallet private keys
key1=$(echo "$wallet1" | awk '/Private key/ { print $3 }')

# Print out the environment variables to copy
echo "Copy the following into your .envrc file:"
echo
echo "# Broadcaster account"
echo "export GS_BROADCASTER_ADDRESS=$address1"
echo "export GS_BROADCASTER_PRIVATE_KEY=$key1"
