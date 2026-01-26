#!/usr/bin/env bash
set -euo pipefail

# Runs the file unchanged script for each of the Kontrol summary files.
# Update these hashes if you have changed the summary files deliberately.
# Use `openssl dgst -sha256` to generate the hash for a file.
./scripts/checks/check-file-unchanged.sh ./test/kontrol/proofs/utils/DeploymentSummary.sol cb2f1d7ff9a6fb878328650114af6a6864ed6e353e1d031b5704dee75270a4b5
./scripts/checks/check-file-unchanged.sh ./test/kontrol/proofs/utils/DeploymentSummaryCode.sol 8cef371441261e54161645036717d3ee21f66bd638a961da698c6858250e41bc
./scripts/checks/check-file-unchanged.sh ./test/kontrol/proofs/utils/DeploymentSummaryFaultProofs.sol 80e9caac4deba370d01dd642401ec2e8145a1e2ae981b7e423d4de188266f87b
./scripts/checks/check-file-unchanged.sh ./test/kontrol/proofs/utils/DeploymentSummaryFaultProofsCode.sol c4379d2232fab24bea2f55d25cef5de4d0629b8e04592f074f680a3141861d9e
