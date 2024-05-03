#!/bin/bash

# Function to manage a specific dependency
manage_dependency() {
  dependency=$1
  operation=$2

  case $dependency in
    "foundry")
      script_prefix="foundry"
      ;;
    "abigen")
      script_prefix="abigen"
      ;;
    "slither")
      script_prefix="slither"
      ;;
    *)
      echo "Unknown dependency: $dependency"
      exit 1
      ;;
  esac

  case $operation in
    "install")
      pnpm run install:$script_prefix
      ;;
    "verify")
      pnpm run check:$script_prefix
      ;;
    "print")
      pnpm run print:$script_prefix
      ;;
    *)
      echo "Unknown operation: $operation"
      exit 1
      ;;
  esac
}

# Main script logic
if [ "$#" -eq 0 ]; then
  # If no args, manage all dependencies
  pnpm run install:foundry
  pnpm run check:foundry
  pnpm run print:foundry

  pnpm run install:abigen
  pnpm run check:abigen
  pnpm run print:abigen

  pnpm run install:slither
  pnpm run check:slither
  pnpm run print:slither

  # Exit with a success status code
  exit 0
else
  # Manage specific dependency based on arguments
  manage_dependency "$1" "$2"
fi
