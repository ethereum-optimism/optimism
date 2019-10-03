#!/bin/bash

mkdir -p log
yarn run aggregator 2>&1 | tee log/aggregator.$(uuidgen).log
