#!/usr/bin/env bash
set -euo pipefail

echo "> Deploying contracts to generate state diff (non-broadcast)"
forge script -vvv scripts/Deploy.s.sol:Deploy --sig 'runWithStateDiff()'
