#!/usr/bin/env bash
# Usage: hack-partition.sh <index>/<total>
#
# Assigns every workspace package to one of <total> CI nodes by weighted LPT
# (longest-processing-time) bin-packing over scripts/hack-weights.txt, then
# prints node <index>'s (1-based) share, one entry per line:
#
#   <package>             run all of the package's checks on this node
#   <package> <i>/<k>     run shard i of the package's checks on this node
#                         (pass to `cargo hack -p <package> --partition i/k`)
#
# The assignment is a pure function of the workspace member list, the weights
# file, and <total>, so every node computes the identical split and each
# package (or shard) lands on exactly one node.
set -euo pipefail

partition=$1
node_index=${partition%/*}
node_total=${partition#*/}

if ! [[ "$node_index" =~ ^[0-9]+$ && "$node_total" =~ ^[0-9]+$ ]] \
  || [ "$node_index" -lt 1 ] || [ "$node_index" -gt "$node_total" ]; then
  echo "invalid partition '$partition': expected <index>/<total> with 1 <= index <= total" >&2
  exit 1
fi

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
weights_file="$script_dir/hack-weights.txt"

# One line per bin-packing item: "<weight> <package> <shard> <shards>". A
# package with a splits column becomes <splits> items of weight/splits each.
cargo metadata --no-deps --format-version 1 \
  | jq -r '.packages[].name' \
  | sort -u \
  | awk -v weights_file="$weights_file" '
      BEGIN {
        default_weight = 30
        while ((getline line < weights_file) > 0) {
          sub(/#.*/, "", line)
          n = split(line, field, /[ \t]+/)
          if (n < 2) continue
          weight[field[1]] = field[2]
          splits[field[1]] = (n >= 3 && field[3] > 1) ? field[3] : 1
        }
      }
      {
        pkg_weight = ($0 in weight) ? weight[$0] : default_weight
        pkg_splits = ($0 in splits) ? splits[$0] : 1
        for (shard = 1; shard <= pkg_splits; shard++)
          printf "%.3f %s %d %d\n", pkg_weight / pkg_splits, $0, shard, pkg_splits
      }' \
  | sort -k1,1nr -k2,2 -k3,3n \
  | awk -v node_index="$node_index" -v node_total="$node_total" '
      {
        item_weight = $1; pkg = $2; shard = $3; shards = $4

        # Lightest bin wins, lowest index breaks ties; bins already holding a
        # shard of this package are skipped so shards spread across nodes.
        best = 0
        for (bin = 1; bin <= node_total; bin++) {
          if (occupied[bin, pkg]) continue
          if (best == 0 || load[bin] < load[best]) best = bin
        }
        if (best == 0) {
          best = 1
          for (bin = 2; bin <= node_total; bin++) if (load[bin] < load[best]) best = bin
        }

        load[best] += item_weight
        occupied[best, pkg] = 1
        if (best != node_index) next
        if (shards > 1) { print pkg, shard "/" shards } else { print pkg }
      }'
