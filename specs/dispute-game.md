# Dispute Game

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Attestation Dispute Game](#attestation-dispute-game)
  - [Smart Contract Implementation](#smart-contract-implementation)
    - [Attestation Structure](#attestation-structure)
  - [Why EIP-712](#why-eip-712)
  - [Offchain Actor](#offchain-actor)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Attestation Dispute Game

The attestation based dispute game is meant to be a dispute game based on social consensus that
can progressively decentralize over time with larger participant sets. When submitting an output
proposal is permissionless, a social quorum can be used to revert "invalid" output proposals.
A set of attestors is maintained and [EIP-712](https://eips.ethereum.org/EIPS/eip-712) signatures
over canonical output roots can be used as attestations.

### Smart Contract Implementation

The `AttestationDisputeGame` should implement the dispute game interface and also be able to call
out to the disputable interface. It is expected that a contract that implements the disputable
interface will have permissions such that the `AttestationDisputeGame` has rights to alter its state.

The `AttestationDisputeGame` should be configured with a quorum ratio at deploy time. It should also
maintain a set of attestor accounts. The ability to add and remove attestor accounts should be
enabled by a single immutable account. It should be impossible to remove accounts such that quorum
is not able to be reached. It is ok to allow accounts to be added or removed in the middle of an
open challenge.

A challenge is opened when an EIP-712 based attestation is presented to the contract and the signer
is in the set of attestors. Multiple challenges should be able to run in parallel.

For simplicity, the `AttestationDisputeGame` does not need to track what output proposals are
committed to as part of the attestations, it only needs to check that the attested value is
different than the proposed value. If this is not checked, then it will be possible to remove
outputs that are in agreement with the attestations and create a griefing vector.

#### Attestation Structure

The EIP-712 [typeHash](https://eips.ethereum.org/EIPS/eip-712#rationale-for-typehash) should be
defined as the following:

```solidity
TYPE_HASH = keccak256("Dispute(bytes32 outputRoot,uint256 l2BlockNumber)");
```

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
