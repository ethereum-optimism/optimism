case $1 in
  setup)
    geth init etc/rollup-fullnode.json;
    geth account new --password /dev/null | grep -oh "0x[a-fA-F0-9]\{40\}" > root/.ethereum/address.txt;
    break
    ;;
  "")
    geth --syncmode 'full' --rpc --rpcaddr 'localhost'  --rpcapi 'eth,net' --networkid 12 --nodiscover --nousb --allow-insecure-unlock --gasprice '1' --mine
    break
    ;;
  *)
    $1
    ;;
esac
