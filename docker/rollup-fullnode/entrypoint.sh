case $1 in
  "")
    npm run server:fullnode
    break
    ;;
  *)
    $1
    ;;
esac
