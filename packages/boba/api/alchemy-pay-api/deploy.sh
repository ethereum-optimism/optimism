#!/bin/bash

set -e

STAGE=${1:-dev}
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
LOGS_DIR="./logs"
mkdir -p "$LOGS_DIR"

# Log file paths
DEV_LOG="$LOGS_DIR/dev-deployments.log"
MAINNET_LOG="$LOGS_DIR/mainnet-deployments.log"

# Create log files if they don't exist
touch "$DEV_LOG" "$MAINNET_LOG"

log_message() {
  local stage=$1
  local message=$2
  local log_file

  if [ "$stage" == "mainnet" ]; then
    log_file="$MAINNET_LOG"
  else
    log_file="$DEV_LOG"
  fi

  echo "[$TIMESTAMP] $message" | tee -a "$log_file"
}

echo "Deploying Alchemy Pay URL Generator service..."
log_message "$STAGE" "Starting deployment for stage: $STAGE"

case $STAGE in
  "mainnet")
    if [ -z "$MAINNET_SECRET_KEY" ]; then
      error_msg="Error: MAINNET_SECRET_KEY environment variable is required for mainnet deployment"
      log_message "mainnet" "$error_msg"
      echo "$error_msg"
      exit 1
    fi

    # Set AWS profile for mainnet
    export AWS_PROFILE=mainnet
    log_message "mainnet" "Using AWS Profile: mainnet"

    # Copy mainnet environment and deploy
    log_message "mainnet" "Copying mainnet environment configuration..."
    cp env-mainnet.yml env.yml

    log_message "mainnet" "Starting serverless deployment..."
    if serverless -c serverless-mainnet.yml deploy 2>&1 | tee -a "$MAINNET_LOG"; then
      log_message "mainnet" "Deployment successful"
    else
      error_msg="Deployment failed"
      log_message "mainnet" "$error_msg"
      rm -rf env.yml
      rm -rf .serverless
      exit 1
    fi

    rm -rf env.yml
    rm -rf .serverless
    log_message "mainnet" "Cleanup completed"

    log_message "mainnet" "Mainnet deployment completed successfully!"
    ;;

  "dev")
    log_message "dev" "Deploying to Development..."

    # Set AWS profile for dev (or use default)
    export AWS_PROFILE=default
    log_message "dev" "Using AWS Profile: default"

    # Copy dev environment and deploy
    log_message "dev" "Copying development environment configuration..."
    cp env-dev.yml env.yml

    log_message "dev" "Starting serverless deployment..."
    if serverless deploy --stage dev 2>&1 | tee -a "$DEV_LOG"; then
      log_message "dev" "Deployment successful"
    else
      error_msg="Deployment failed"
      log_message "dev" "$error_msg"
      rm -rf env.yml
      rm -rf .serverless
      exit 1
    fi

    rm -rf env.yml
    rm -rf .serverless
    log_message "dev" "Cleanup completed"

    log_message "dev" "Development deployment completed successfully!"
    ;;

  *)
    error_msg="Invalid stage: $STAGE. Usage: ./deploy.sh [dev|mainnet]"
    log_message "$STAGE" "$error_msg"
    echo "$error_msg"
    exit 1
    ;;
esac

log_message "$STAGE" "Deployment finished for stage: $STAGE"

# Add logs directory to .gitignore if not already present
if ! grep -q "^logs/" .gitignore; then
  echo "logs/" >> .gitignore
  echo "Added logs/ to .gitignore"
fi
