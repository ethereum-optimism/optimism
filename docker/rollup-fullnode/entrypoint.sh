case $1 in
  "")
    ping geth
    #npm run server:fullnode
    break
    ;;
  *)
    $1
    ;;
esac
