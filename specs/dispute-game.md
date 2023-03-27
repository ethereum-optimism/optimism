# Dispute Game

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Attestation Dispute Game](#attestation-dispute-game)
  - [Why EIP-712](#why-eip-712)
  - [Offchain Actor](#offchain-actor)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Attestation Dispute Game

The attestation based dispute game is meant to be a dispute game based on social consensus that
can progressively decentralize over time with larger participant sets. When submitting an output
proposal is permissionless, a social quorum can be used to revert "invalid" output proposals.
A set of attestors is maintained, and [EIP-712](https://eips.ethereum.org/EIPS/eip-712) signatures
over canonical output roots can be used as attestations.

### Why EIP-712

It is important to use EIP-712 to decouple the originator of the transaction and the attestor. This
will allow a decentralized network of attestors that serve attestations to bots that are responsible
for ensuring that all output proposals submitted to the network will not allow for malicious withdrawals
from the bridge.

It is important to have replay protection to ensure that attestations cannot be used more than once.

### Offchain Actor

The offchain actor should be able to custody an attestation key as well as a transaction signing key.
The offchain actor is expected to watch for each output proposal that is submitted to the
`L2OutputOracle` and then check the value against the value returned from a trusted RPC endpoint.
If the trusted value does not match what was submitted to the chain, the actor is expected to submit
an EIP-712 signature to the `AttestationDisputeGame` contract. After a quorum of signatures are sent
to the contract, the `AttestationDisputeGame` will call the `L2OutputOracle` and remove the
malicious output proposal.

Longer term, the actor should be capible of calling out to an "Attestation API", so that it will no
longer be responsible for custodying the attestation key itself and instead can rely on public
infrastructure to get attestations.
