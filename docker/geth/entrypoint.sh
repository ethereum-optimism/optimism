HOSTNAME='geth'
CONFIG_DIR='/root/.ethereum'
SEALER_PRIVATE_KEY_PATH="${CONFIG_DIR}/sealer_private_key.txt"
PRIVATE_KEY_PATH="${CONFIG_DIR}/private_key.txt"
ADDRESS_PATH="${CONFIG_DIR}/address.txt"
SEALER_ADDRESS_PATH="${CONFIG_DIR}/sealer_address.txt"
INITIAL_BALANCE='0x200000000000000000000000000000000000000000000000000000000000000'
GENISIS_PATH='etc/rollup-fullnode.json'
NETWORK_ID=108
PORT=8545

## Generates a private key provided a path
# https://gist.github.com/miguelmota/3793b160992b4ea0b616497b8e5aee2f
generate_private_key()
{
  openssl ecparam -name secp256k1 -genkey -noout |
    openssl ec -text -noout  |
      grep priv -A 3 |
        tail -n +2 |
          tr -d '\n[:space:]:' |
            sed 's/^00//'
}

## Generates a geneneis file with a prefunded account
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
  cp $GENISIS_PATH $CONFIG_DIR
}


case $1 in
  setup)
    generate_private_key > $SEALER_PRIVATE_KEY_PATH
    geth account import --password /dev/null $SEALER_PRIVATE_KEY_PATH |
      grep -oh "[a-fA-F0-9]\{40\}" | (echo -n "0x" && cat)  > $SEALER_ADDRESS_PATH;
    generate_private_key > $PRIVATE_KEY_PATH
    geth account import --password /dev/null $PRIVATE_KEY_PATH | grep -oh "[a-fA-F0-9]\{40\}" | (echo -n "0x" && cat) > $ADDRESS_PATH;
#(echo -n "0x" && cat) |
    generate_geneisis `cat $SEALER_ADDRESS_PATH` `cat $ADDRESS_PATH`
    echo "Miner address: `cat $SEALER_ADDRESS_PATH`"
    echo "User address: `cat $ADDRESS_PATH`"

    # If geth is initialized with the mounted volume
    # it fails with the following error:
    # Fatal: Failed to open database: resource temporarily unavailable
    # Instead we initialize it in a temporary directory and copy th results
    mkdir -p /root/tmp
    geth --datadir /root/tmp --nousb --verbosity 0 init $GENISIS_PATH;
    rm -rf /root/.ethereum/geth
    mv /root/tmp/geth /root/.ethereum
    break
    ;;
  "")
    geth --syncmode 'full' --rpc --rpcaddr $HOSTNAME  --rpcvhosts=$HOSTNAME --rpcapi 'eth,net' --rpcport $PORT --networkid $NETWORK_ID --nodiscover --nousb --allow-insecure-unlock -unlock `cat $SEALER_ADDRESS_PATH` --password /dev/null --gasprice '1' --mine
    break
    ;;
  *)
    $1
    ;;
esac

