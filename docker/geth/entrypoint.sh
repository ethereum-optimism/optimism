#!/bin/sh

## Passed in from environment variables:
# HOSTNAME=
# PORT=8545
# NETWORK_ID=108
SEALER_PRIVATE_KEY_PATH="${VOLUME_PATH}${SEALER_PRIVATE_KEY_PATH_SUFFIX}"
PRIVATE_KEY_PATH="${VOLUME_PATH}${PRIVATE_KEY_PATH_SUFFIX}"
ADDRESS_PATH="${VOLUME_PATH}${ADDRESS_PATH_SUFFIX}"
SEALER_ADDRESS_PATH="${VOLUME_PATH}${SEALER_ADDRESS_PATH_SUFFIX}"
SETUP_RUN_PATH="${VOLUME_PATH}${SETUP_RUN_PATH_SUFFIX}"
CLEAR_DATA_FILE_PATH="${VOLUME_PATH}/.clear_data_key_${CLEAR_DATA_KEY}"

## Generates an Ethereum private key
# Source: https://gist.github.com/miguelmota/3793b160992b4ea0b616497b8e5aee2f
generate_private_key()
{
  openssl ecparam -name secp256k1 -genkey -noout 2> /dev/null |
    openssl ec -text -noout 2> /dev/null |
      grep priv -A 3 |
        tail -n +2 |
          tr -d '\n[:space:]:' |
            sed 's/^00//'
}

## Imports a private key into geth and returns the address
import_private_key()
{
  geth --datadir $VOLUME_PATH account import --password /dev/null $1 2> /dev/null |
    grep -oh "[a-fA-F0-9]\{40\}" | (echo -n "0x" && cat)
}

## Generates a geneneis file with one authrorized sealer and
## one prefunded account.
generate_geneisis()
{
  SEALER_ADDRESS=$1
  ADDRESS=$2
  ADDRESS_BYTES=`echo $ADDRESS | sed 's/^0x//'`
  SEALER_ADDRESS_BYTES=`echo $SEALER_ADDRESS | sed 's/^0x//'`
  EXTRA_DATA=`jq -r '.extraData' $GENESIS_PATH | sed "s/\\$SEALER_ADDRESSES/$SEALER_ADDRESS_BYTES/g"`
  tmp=$(mktemp)
  jq --arg address $ADDRESS_BYTES --arg balance $INITIAL_BALANCE '.alloc += {($address): {balance: $balance}}' $GENESIS_PATH > $tmp
  mv $tmp $GENESIS_PATH
  jq --arg extraData $EXTRA_DATA '.extraData = $extraData' $GENESIS_PATH > $tmp
  mv $tmp $GENESIS_PATH
}

if [[ -n "$CLEAR_DATA_KEY" && ! -f "$CLEAR_DATA_FILE_PATH" ]]; then
  echo "Detected change in CLEAR_DATA_KEY. Purging data."
  rm -rf ${VOLUME_PATH}/*
  rm -rf ${VOLUME_PATH}/.clear_data_key_*
  echo "Local data cleared from '${VOLUME_PATH}/*'"
  echo "Contents of volume dir: $(ls -alh $VOLUME_PATH)"
  touch $CLEAR_DATA_FILE_PATH
fi

## One-time configuration to be run only on first startup
if [ ! -f $SETUP_RUN_PATH ]; then
  echo "Generating keys..."

  generate_private_key > $SEALER_PRIVATE_KEY_PATH
  import_private_key $SEALER_PRIVATE_KEY_PATH > $SEALER_ADDRESS_PATH

  if [ -z "$PRIVATE_KEY" ]; then
    echo "\nGENERATING PRIVATE KEY!! We most likely don't want to do this.\n"
    generate_private_key > $PRIVATE_KEY_PATH
  else
    echo "Reading private key from env"
    echo "$PRIVATE_KEY" | sed 's/^0x//' > $PRIVATE_KEY_PATH
  fi

  import_private_key $PRIVATE_KEY_PATH > $ADDRESS_PATH

  generate_geneisis `cat $SEALER_ADDRESS_PATH` `cat $ADDRESS_PATH`

  echo "Ran Setup" > $SETUP_RUN_PATH

  echo "Setup Complete"
  echo "Account Address: 0x`cat $ADDRESS_PATH`"
else
  echo "FOUND EXISTING GETH DATA"
fi

echo "Starting Geth..."
## Command to kick off geth
geth --dev --datadir $VOLUME_PATH --rpc --rpcaddr $HOSTNAME --rpcvhosts=* --rpcport $PORT --networkid $NETWORK_ID --rpcapi 'eth,net' --gasprice '0' --targetgaslimit '4294967295'
