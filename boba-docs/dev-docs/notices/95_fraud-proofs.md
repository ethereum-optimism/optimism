# Fraud Proofs on Boba Network

Fraud proofs are important for making L2 more decentralized, allowing more community involvement in validating the L2 state. These changes allow anyone to submit or challenge a state root, an essential part of the process used for withdrawing funds from L2 to L1.

## Withdrawal Process

* Withdrawals require proving and finalizing based on the fault proof system.

* Withdrawals require 7 days to finalize depending on the outcome of the dispute game.

* The `PermissionedDisputeGame` (via `DisputeGameFactory`) is used for proposing output root statements, enhancing platform security and reliability.

* Valid proofs challenged by malicious players can be delayed for up to a few additional hours, incurring a very high cost to the malicious actor.
