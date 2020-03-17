case $1 in
  deploy_execution_manager)
    yarn run deploy:execution-manager production
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
