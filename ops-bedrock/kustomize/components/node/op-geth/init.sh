#!/bin/sh
set -eu

apk add --no-cache curl

datadir=${OP_GETH_DATA_DIR:-/db}
chaindata_dir="$datadir/geth/chaindata"
verbosity=${GETH_VERBOSITY:-3}
genesis_file_path="$datadir/genesis.json"

# Check to see if Geth's datadir has already been
# initialized. If it hasn't, download the genesis file
# and initialize the data directory with it.

if [ -d "$chaindata_dir" ]; then
  echo "Chain already initialized at $chaindata_dir."
else
	echo "$chaindata_dir missing, running init."

  if [ ! -f "$genesis_file_path" ]; then
    if [ -z "${OP_GETH_GENESIS_URL-}" ]; then
      echo "You must specify OP_GETH_GENESIS_URL during initialization."
      exit 1
    fi

    echo "Downloading genesis file from $OP_GETH_GENESIS_URL."
	  curl -o "$genesis_file_path" -L "$OP_GETH_GENESIS_URL"
	fi

	echo "Initializing genesis."
	geth --verbosity="$verbosity" init \
		--datadir="$datadir" \
		"$genesis_file_path"
fi