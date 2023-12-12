#!/usr/bin/env bash
echo "$1"
jq '.frames[] | {timestamp, inclusion_block}' "$1"
jq '.batches[]|.Timestamp' "$1"
