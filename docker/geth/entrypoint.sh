#!/bin/sh

# PASSED IN FROM ENV
#HOSTNAME=

KEYSTORE_PATH="${VOLUME_PATH}/keystore"
SEALER_PRIVATE_KEY_PATH="${VOLUME_PATH}/sealer_private_key.txt"
PRIVATE_KEY_PATH="${VOLUME_PATH}/private_key.txt"
ADDRESS_PATH="${VOLUME_PATH}/address.txt"
SEALER_ADDRESS_PATH="${VOLUME_PATH}/sealer_address.txt"
INITIAL_BALANCE='0x200000000000000000000000000000000000000000000000000000000000000'
GENISIS_PATH='etc/rollup-fullnode.json'
NETWORK_ID=108
PORT=8545

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
  EXTRA_DATA=`jq -r '.extraData' $GENISIS_PATH | sed "s/\\$SEALER_ADDRESSES/$SEALER_ADDRESS_BYTES/g"`
  tmp=$(mktemp)
  jq --arg address $ADDRESS_BYTES --arg balance $INITIAL_BALANCE '.alloc += {($address): {balance: $balance}}' $GENISIS_PATH > $tmp
  mv $tmp $GENISIS_PATH
  jq --arg extraData $EXTRA_DATA '.extraData = $extraData' $GENISIS_PATH > $tmp
  mv $tmp $GENISIS_PATH
}


case $1 in
  setup)
    if test -d "$KEYSTORE_PATH"; then
      echo "Setup has already been run"
      exit 1
    fi
    generate_private_key > $SEALER_PRIVATE_KEY_PATH
    import_private_key $SEALER_PRIVATE_KEY_PATH > $SEALER_ADDRESS_PATH
    generate_private_key > $PRIVATE_KEY_PATH
    import_private_key $PRIVATE_KEY_PATH > $ADDRESS_PATH
    generate_geneisis `cat $SEALER_ADDRESS_PATH` `cat $ADDRESS_PATH`

    geth --datadir $VOLUME_PATH --nousb --verbosity 0 init $GENISIS_PATH 2> /dev/null;
    echo "Setup Complete"
    echo "Sealer Address: 0x`cat $SEALER_PRIVATE_KEY_PATH`"
    echo "Account Address: 0x`cat $PRIVATE_KEY_PATH`"
    break
    ;;
  "")
    geth --datadir $VOLUME_PATH --syncmode 'full' --rpc --rpcaddr $HOSTNAME  --rpcvhosts=$HOSTNAME --rpcapi 'eth,net' --rpcport $PORT --networkid $NETWORK_ID --nodiscover --nousb --allow-insecure-unlock -unlock `cat $SEALER_ADDRESS_PATH` --password /dev/null --gasprice '1' --mine
    break
    ;;
  *)
    $1
    ;;
esac

