#!/bin/bash

mkdir -p log
yarn run aggregator 2>&1 | tee log/aggregator.$(date '+%Y.%m.%d_%H.%M.%S')_$(uuidgen).log
