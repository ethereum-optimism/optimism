#!/usr/bin/env bash
set -euo pipefail

required_tests=(
  TestFlashblocksStream
  TestFlashblocksTransfer
  TestSDMDisabledNoRefunds
  TestSDMOptInIsInertOnStockOpReth
  TestSDMFixturePayloadReceiptAndAccounting
  TestSDMFixtureOperatorOptInControlsProduction
  TestSDMPostExecBlockDerivesAndChainProgresses
  TestSDMPostExecBlockDerivesOnIsolatedVerifier
  TestSDMPostExecSpanCrossesInteropBoundary
  TestSDMActivatesAtLagoonBoundary
  TestInteropSingleChainFaultProofsWithSDM
)

for test_name in "${required_tests[@]}"; do
  if ! grep -R -q --include='*_test.go' "^func ${test_name}(gt \\*testing.T)" op-acceptance-tests/tests; then
    echo "required SDM fixture acceptance test is missing: $test_name" >&2
    exit 1
  fi
done

echo "all required SDM fixture acceptance test names are present"
