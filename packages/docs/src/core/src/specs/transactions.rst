========================
Transactions Over Ranges
========================
Our plasma design introduces an new construction that allows users to transact over *ranges* of coins, rather than simply transacting single coins at a time.
This page explains how we achieve that.

Transfers
=========
A transaction consists of a specified block number and an array of Transfer_ objects.
These Transfer_ objects describe the actual details of the transaction, including which ranges are being sent and to whom.

From the schema_ in ``plasma-utils``:

.. code-block:: javascript

    const TransferSchema = new Schema({
     sender: {
       type: Address,
       required: true
     },
     recipient: {
       type: Address,
       required: true
     },
     token: {
       type: Number,
       length: 4,
       required: true
     },
     start: {
       type: Number,
       length: 12,
       required: true
     },
     end: {
       type: Number,
       length: 12,
       required: true
     }

(Note that ``length`` is in bytes)

We can see that each Transfer_ in a Transaction_ specifies a ``tokenType``, ``start``, ``end``, ``sender``, and ``recipient``.

Typed and Untyped Bounds
========================
One thing to note above is that the ``start`` and ``end`` values are *12 bytes* and *not 16 bytes* like the `Coin ID`_.
This is because these values are "untyped" - they don't take the `tokenType` into account.
We can calculate the "typed" values by concatenating the ``token`` field to either ``start`` or ``end``.
This design choice was made for API simplicity.

Atomic Transactions
===================
The Transaction_ schema contains an *array* of Transfer_ objects.
This means that a transaction can describe several transfers at the same time.
Multiple transfers in the same transaction are all atomically executed *if any only if* the *entire transaction* is included and valid.
This will form the basis for both decentralized exchange and `defragmentation`_ in later releases.

Serialization
=============
``plasma-utils`` implements a `custom serialization library`_ for the above data structures.
Both the JSON-RPC API and the smart contract use byte arrays as encoded by the serializer.

Our encoding scheme is quite simple.
The encoded version of a piece of data structure is the concatenation of each its values.
Because each value has a fixed number of bytes defined by a schema, we can decode by slicing off the appropriate number of bytes.
The choice to use a custom scheme instead of an existing one (like `RLP encoding`_) was made to reduce smart contract complexity.

For encodings which involve variable-sized arrays, like Transaction_ objects which contain 1 or more Transfer_ objects, we prepend a single byte that represents the number of array elements.

.. _schema: https://plasma-utils.readthedocs.io/en/latest/serialization.html#transferschema
.. _Transfer: https://plasma-utils.readthedocs.io/en/latest/models.html#transfer
.. _Transaction: https://plasma-utils.readthedocs.io/en/latest/models.html#SignedTransaction
.. _Coin ID: specs/coin-assignment.html
.. _defragmentation: TODO
.. _custom serialization library: https://plasma-utils.readthedocs.io/en/latest/serialization.html
.. _RLP encoding: https://github.com/ethereum/wiki/wiki/%5BEnglish%5D-RLP
