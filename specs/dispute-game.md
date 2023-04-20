# Dispute Game

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Attestation Dispute Game](#attestation-dispute-game)
  - [Smart Contract Implementation](#smart-contract-implementation)
    - [Attestation Structure](#attestation-structure)
  - [Why EIP-712](#why-eip-712)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Attestation Dispute Game

The output attestation based dispute game shifts the current permissioned output proposal process
to a permissionless, social-consensus based architecture that can progressively decentralize over
time by increasing the size of the signer set. In this "game," output proposals can be submitted
permissionlessly. To prevent "invalid output proposals," a social quorum can revert an output proposal
when an invalid one is discovered. The set of signers is maintained in the `SystemConfig` contract,
and these signers will issue [EIP-712](https://eips.ethereum.org/EIPS/eip-712) signatures
over canonical output roots and the `l2BlockNumber`s they commit to as attestations. To learn more,
see the [DisputeGame Interface Spec](./dispute-game-interface.md).

In the above language, an "invalid output proposal" is defined as an output proposal that represents
a non-canonical state of the L2 chain.

### Smart Contract Implementation

The `AttestationDisputeGame` should implement the `IDisputeGame` interface and also be able to call
out to the `L2OutputOracle`. It is expected that the `L2OutputOracle` will grant permissions to
`AttestationDisputeGame` contracts to call its `deleteL2Outputs` function at the *specific* `l2BlockNumber`
that is embedded in the `AttestationDisputeGame`'s `extraData`.

The `AttestationDisputeGame` should be configured with a quorum ratio at deploy time. It should also
maintain a set of attestor accounts, which is fetched by the `SystemConfig` contract and snapshotted
at deploy time. This snapshot is necessary to have a fixed upper bound on resolution cost, which in
turn gives a fix cost for the necessary bond attached to output proposals.

The ability to add and remove attestor accounts should be enabled by a single immutable
account that controls the `SystemConfig`. It should be impossible to remove accounts such that quorum
is not able to be reached. It is ok to allow accounts to be added or removed in the middle of an
open challenge, as it will not affect the `signerSet` that exists within open challenges.

A challenge is created when an alternative output root for a given `l2BlockNumber` is presented to the
`DisputeGameFactory` contract. Multiple challenges should be able to run in parallel.

For simplicity, the `AttestationDisputeGame` does not need to track what output proposals are
committed to as part of the attestations. It only needs to check that the attested output root
is different than the proposed output root. If this is not checked, then it will be possible
to remove output proposals that are in agreement with the attestations and create a griefing vector.

#### Attestation Structure

The EIP-712 [typeHash](https://eips.ethereum.org/EIPS/eip-712#rationale-for-typehash) should be
defined as the following:

```solidity
TYPE_HASH = keccak256("Dispute(bytes32 outputRoot,uint256 l2BlockNumber)");
```

The components for the `typeHash` are as follows:

- `outputRoot` - The **correct** output root that commits to the given `l2BlockNumber`. This should be a
  positive attestation where the `rootClaim` of the `AttestationDisputeGame` is the **correct** output root
  for the given `l2BlockNumber`.
- `l2BlockNumber` - The L2 block number that the `outputRoot` commits to. The `outputRoot` should commit
  to the entirety of the L2 state from genesis up to and including this `l2BlockNumber`.

### Why EIP-712

It is important to use EIP-712 to decouple the originator of the transaction and the attestor. This
will allow a decentralized network of attestors that serve attestations to bots that are responsible
for ensuring that all output proposals submitted to the network will not allow for malicious withdrawals
from the bridge.

It is important to have replay protection to ensure that attestations cannot be used more than once.
