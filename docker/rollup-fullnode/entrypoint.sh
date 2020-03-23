#/bin/sh

case $1 in
  "")
    cd packages/rollup-fullnode && yarn run server:fullnode
    break
    ;;
  test)
    yarn test
    ;;
  *)
    $1
    ;;
esac
