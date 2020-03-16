HOSTNAME='geth'
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

    geth account new --password /dev/null | grep -oh "0x[a-fA-F0-9]\{40\}" > root/.ethereum/address.txt;
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
