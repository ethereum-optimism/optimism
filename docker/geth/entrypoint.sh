#!/bin/sh

## Passed in from environment variables:
# HOSTNAME=
# PORT=8545
# NETWORK_ID=108
CLEAR_DATA_FILE_PATH="${VOLUME_PATH}/.clear_data_key_${CLEAR_DATA_KEY}"

if [[ -n "$CLEAR_DATA_KEY" && ! -f "$CLEAR_DATA_FILE_PATH" ]]; then
  echo "Detected change in CLEAR_DATA_KEY. Purging data."
  rm -rf ${VOLUME_PATH}/*
  rm -rf ${VOLUME_PATH}/.clear_data_key_*
  echo "Local data cleared from '${VOLUME_PATH}/*'"
  echo "Contents of volume dir: $(ls -alh $VOLUME_PATH)"
  touch $CLEAR_DATA_FILE_PATH
fi

echo "Starting Geth..."
## Command to kick off geth
geth --dev --datadir $VOLUME_PATH --rpc --rpcaddr $HOSTNAME --rpcvhosts=* --rpcport $PORT --networkid $NETWORK_ID --rpcapi 'eth,net' --gasprice '0' --targetgaslimit '4294967295' --nousb --gcmode=archive
