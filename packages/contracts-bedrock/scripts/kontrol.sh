#!/usr/bin/env bash
set -exuo pipefail
# Setup Kontrol 
cd packages/contracts-bedrock && ./test/kontrol/kontrol/run-kontrol.sh
