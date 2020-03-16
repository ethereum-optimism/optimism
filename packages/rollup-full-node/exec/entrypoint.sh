case $1 in
  deploy_execution_manager)
    npm run server:fullnode
    break
    ;;
  "")
    npm run server:fullnode
    break
    ;;
  *)
    $1
    ;;
esac
