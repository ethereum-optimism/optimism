#!/bin/bash

mkdir -p ../log
yarn --cwd ../ run validator 2>&1 | tee ../log/validator.$(date '+%Y.%m.%d_%H.%M.%S')_$(uuidgen).log
