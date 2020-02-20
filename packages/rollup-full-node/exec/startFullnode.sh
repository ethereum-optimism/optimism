#!/bin/bash

mkdir -p ../log
SCRIPT_DIR=$(dirname $0)
yarn --cwd ${SCRIPT_DIR}/.. run server:fullnode 2>&1 | tee ../log/fullnode.$(date '+%Y.%m.%d_%H.%M.%S')_$(uuidgen).log
