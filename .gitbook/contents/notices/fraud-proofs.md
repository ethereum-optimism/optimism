# Prepare for Fraud Proofs on Boba Sepolia and Boba BNB Testnet 

Fraud proofs are important for making L2 more decentrailized, allowing more community involvement in validating the L2 state. These changes allow anyone to withdraw funds from Boba Layer 2 to L1without relying on any centralized actors.

## Prepare for Fault Proofs

Fault proofs are expected to go live for Boba Testnets (Boba Sepolia and Boba BNB Testnet) in mid-August.

### Changes for Testnet Withdrawals

* Withdrawal will require proving and finalizing based on the fault proof system.

* Withdrawal will require 7 days to finalize depending on the outcome of the dispute game. 

  > Boba Testnets withdrawals are no longer instant.

* The `DisputeGameFactory` will replace the `L2OutputOracle` contract for prosing output root statements, enhancing platform security and reliability.

* The valid proofs challenged by maliciously players can be delayed up to an anditional a few hours at a very high cost to the malicious actor.

### Changes for Users

If you submit your proof prior to the fraud proof upgrades, you must submit it again and wait an additional seven days before you can claim your withdrawal. You may consider waiting until after the upgrade is complete to submit your proof.

### Changes for Bridge Operators

Upgrade your bridge to support new L1 smart contracts.
