#!/usr/bin/env bash

set -e

dir=$(dirname "$0")

echo "Validating deployment configurations...\n"

for config in $dir/../deploy-config/*.json
do
  echo "Found file: $config\n"
  git diff --exit-code $config
done

echo "Deployment configs in $dir/../deploy-config validated!\n"
