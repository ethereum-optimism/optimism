#!/bin/bash
export SEQUENCER_ADDRESS=0xabba
export SEQUENCER_DANGER_THRESHOLD=100 # 100 eth

export PROPOSER_ADDRESS=0xacdc
export PROPOSER_DANGER_THRESHOLD=200 # 200 eth

yarn ts-mocha src/*.spec.ts
