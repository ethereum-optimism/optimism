case $1 in
  deploy_execution_manager)
    npm run deploy:execution-manager
    break
    ;;
  "")
    # Don't do anything by default
    break
    ;;
  *)
    $1
    ;;
esac
