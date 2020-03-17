HOSTNAME='geth'
CONFIG_DIR='root/.ethereum/'
PRIVATE_KEY_PATH="${CONFIG_DIR}/private_key.txt"
ADDRESS_PATH="${CONFIG_DIR}/address.txt"

# https://gist.github.com/miguelmota/3793b160992b4ea0b616497b8e5aee2f
generate_private_key()
{
  PRIVATE_KEY_PATH=$1
  openssl ecparam -name secp256k1 -genkey -noout |
    openssl ec -text -noout  |
      grep priv -A 3 |
        tail -n +2 |
          tr -d '\n[:space:]:' |
            sed 's/^00//' |
              (echo -n "0x" && cat)> $PRIVATE_KEY_PATH
  echo "done"
}


case $1 in
  setup)
    # If geth is initialized with the mounted volume
    # it fails with the following error:
    # Fatal: Failed to open database: resource temporarily unavailable
    # Instead we initialize it in a temporary directory and copy th results
    mkdir -p /root/tmp
    geth --datadir /root/tmp --nousb --pcscdpath /dev/null --verbosity 5 init etc/rollup-fullnode.json;
    rm -rf /root/.ethereum/*
    mv /root/tmp/* /root/.ethereum

    generate_private_key $PRIVATE_KEY_PATH
    geth account import root/.ethereum/private_key.txt
    geth account new --password /dev/null | grep -oh "0x[a-fA-F0-9]\{40\}" > $ADDRESS_PATH;
    break
    ;;
  "")
    echo "Starting Geth on port 8546"
    geth --syncmode 'full' --rpc --rpcaddr $HOSTNAME  --rpcvhosts=$HOSTNAME --rpcapi 'eth,net' --rpcport 8546 --networkid 12 --nodiscover --nousb --allow-insecure-unlock --gasprice '1' --mine
    break
    ;;
  *)
    $1
    ;;
esac

