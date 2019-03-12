================================
Proof Structure and Verification
================================

What are History Proofs?
========================
Unlike traditional blockchain nodes, our full plasma nodes don't store every single transaction.
Instead, they only ever need to store information relevant to assets they own.
This means that when you're receiving an asset, you don't have all of the information necessary to know that the person sending that asset is, in fact, the true owner.
The ``sender`` needs to explicitly prove to the ``recipient`` that the ``sender`` actually owns the asset being sent.

Currently, the ``sender`` needs to give the ``recipient`` a list of every single transaction that has ever moved the asset around.
The ``sender`` also needs to provide proof that they're not omitting any transactions.
Once the ``recipient`` checks this proof, they can see the whole chain of custody for the asset and be convinced that the ``sender`` is the current owner.
Generally, we call this proof a "history proof" because it tells the ``recipient`` about the history of an asset.

This page describes the methodology ``plasma-core`` follows to verify these history proofs.

Proof Structure
===============
History proofs consist of a set of Deposits_ a long list of relevant Transactions_ with corresponding TransactionProofs_.
Here we'll discuss all of the various components of the proof.

Deposits
--------
Deposits_ form the beginning of each history proof.
An asset's history always starts from the point at which it was created.
When we're talking about a range_, we might need to provide more than one deposit.

Let's look at an example.
Imagine that a sender is trying to create a proof for the range ``(0, 100)``. 
The range ``(0, 25)`` was created in deposit #1, and the range ``(25, 100)`` was created in deposit #2. 
The sender **must** provide these two deposits as part of the proof.

Transaction Proofs
------------------
A TransactionProof_ contains all the necessary information to check the validity of a given Transaction_.
Namely, it is simply an array of TransferProof_ objects (described below).
A given TransactionProof_ is valid if and only if all its TransferProofs_ are valid.

Transfer Proofs
---------------
A TransferProof_ contains all the necessary information required to check that a specific Transfer_ inside of a Transaction_ is valid.
This includes:

* The Merkle tree branch that shows the transaction was included in a block.
* The position of the Merkle tree in which the transaction was included.
* The "parsed sum" for that transaction - a special value necessary to verify the Merkle proof.
* The transaction signature from the sender of the transfer.

Here's the schema taken right from ``plasma-utils``:

.. code-block:: javascript

    const TransferProofSchema = new Schema({
      parsedSum: {
        type: Number,
        length: 16
      },
      leafIndex: {
        type: Number,
        length: 16
      },
      signature: {
        type: SignatureSchema
      },
      inclusionProof: {
        type: [Bytes],
        length: 48
      }
    })

Note that the ``inclusionProof`` is a variable-length array whose size depends on the depth of the tree.

Proof Verification
==================
The process of verifying a proof for an incoming transaction involves applying each proof element to the current "verified" state, starting with the deposits.
If any proof element doesn't result in a valid state transition, we simply ignore that element and go onto the next.
At the very end, we check that each of the transfers in the incoming transaction is part of the verified state.

Snapshot Objects
----------------
We keep track of the current owner of a range using an object called a Snapshot_.
Quite simply, a Snapshot_ represents the verified owner of a range at a block:

.. code-block:: json

    {
      start: number,
      end: number,
      block: number,
      owner: address
    }

Checking for Exits
------------------
Before doing anything else, the verifier **must** check that the ranges being received have no pending or finalized exits.
If any part of the received ranges have pending or finalized exits, the transaction should be rejected.

Applying Deposits
-----------------
Every received range has to come from a corresponding deposit.
A deposit record consists of its ``token``, ``start``, ``end``, ``depositer``, and ``blockNumber``.

For each deposit record, the verifier **must** double-check with Ethereum to verify that the claimed deposit did indeed occur.
The verifier must then add a verified Snapshot_ for each valid deposit, where ``snapshot.owner = deposit.depositer``.

Applying Transactions
---------------------
Next, the verifier must apply all given TransactionProofs_ and update the set of verified Snapshots_ accordingly.
For each Transaction_ and corresponding TransactionProof_, the verifier **must** first perform the following validation steps:

1. Check that the transaction encoding is well-formed.
2. For each Transfer_ in the Transaction_:
  1. Check that the Transfer_ has a corresponding Signature_ created by ``transfer.sender``.
  2. Check that the Transfer_ was included in the plasma block using the ``inclusionProof``, ``leafIndex``, and ``parsedSum``.
  3. Calculate the ``implicitStart`` and ``implicitEnd`` of the Transfer, and verify that ``implicitStart <= transfer.start < transfer.end <= implicitEnd``.

If any of the above checks fail, the transaction **must** be ignored and the verifier should continue onto the next transaction.

If all of the checks are successful, the verifier **must** apply each Transfer_ to the verified state:

1. For each Transfer_ in the Transaction_, do the following:
  1. Break the Transfer_ into *implicit* components (``[implicitStart, typedStart], [typedEnd, implicitEnd]``) and *explicit* components (``[typedStart, typedEnd]``).
  2. For each component:
    1. Find all verified Snapshots_ that overlap with the component.
    2. For each Snapshot_ that overlaps:
      2. Remove the Snapshot_ from the verified state.
      3. Split the Snapshot_ into overlapping and non-overlapping components.
      4. Re-insert any non-overlapping components into the verified state.
      5. If ``snapshot.block === transaction.blockNumber - 1`` and ``snapshot.owner === component.sender || component.implicit``:
        1. Increment ``snapshot.block``.
        2. Set ``snapshot.owner = transfer.sender``.
      6. Insert the overlapping snapshot back into the verified state.

Verifying Transactions
----------------------
Once all Deposits_ and Transactions_ have been applied to the verified state, the verifier can check the validity of the incoming transaction.
The verifier **must** check that for each Transfer_ in the Transaction_, there exists some Snapshot_ in the verified state such that:

1. ``snapshot.owner === transfer.recipient``.
2. ``snapshot.start <= transfer.typedStart``.
3. ``snapshot.end >= transfer.typedEnd``.

If this condition is true for each Transfer_ in the Transaction_, the proof can be accepted.

.. _Deposits: TODO
.. _Transfer: https://plasma-utils.readthedocs.io/en/latest/models.html#transfer
.. _Transaction: https://plasma-utils.readthedocs.io/en/latest/models.html#signedtransaction
.. _Transactions: https://plasma-utils.readthedocs.io/en/latest/models.html#signedtransaction
.. _TransferProof: TODO
.. _TransferProofs: TODO
.. _TransactionProof: TODO
.. _TransactionProofs: TODO
.. _Snapshot: TODO
.. _Snapshots: TODO
.. _plasma-utils: https://plasma-utils.readthedocs.io/en/latest/index.html
.. _range: specs/transactions.html#ranges
