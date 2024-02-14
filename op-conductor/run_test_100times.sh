#!/bin/bash

set -e

for i in {1..100}; do
  echo "======================="
  echo "Running iteration $i"
  if ! gotestsum -- -run 'TestControlLoop' ./... --count=1 --timeout=5s -race; then
    echo "Test failed"
    exit 1
  fi
done
