=============
Serialization
=============

``plasma-utils.serialization`` provides utilities for encoding and decoding transactions.
It also provides tools for defining your own custom schemas using our encoding scheme.

-----------------------------------------------------------------------------

--------------
TransferSchema
--------------

.. code-block:: javascript

    const TransferSchema = new Schema({
      token: {
        type: ByteArray,
        length: 4,
        required: true
      },
      start: {
        type: ByteArray,
        length: 12,
        required: true
      },
      end: {
        type: ByteArray,
        length: 12,
        required: true
      },
      sender: {
        type: Address,
        required: true
      },
      recipient: {
        type: Address,
        required: true
      }
    })

A `Transfer` represents a portion of a transaction.
Each transaction is composed of one or more `Transfer`.
By allowing a transaction to support more than one transfer, we enable atomic swaps.

-----------------------------------------------------------------------------

---------------
SignatureSchema
---------------

.. code-block:: javascript

    const SignatureSchema = new Schema({
      v: {
        type: ByteArray,
        length: 1,
        required: true
      },
      r: {
        type: ByteArray,
        length: 32,
        required: true
      },
      s: {
        type: ByteArray,
        length: 32,
        required: true
      }
    })

A `Signature` is a simple representation of an ECDSA signature.

------------------------------------------------------------------------------

-------------------------
UnsignedTransactionSchema
-------------------------

.. code-block:: javascript

    const UnsignedTransactionSchema = new Schema({
      block: {
        type: Number,
        length: 4,
        required: true
      },
      transfers: {
        type: [TransferSchema]
      }
    })

An `UnsignedTransactionSchema` is composed of one or more `Transfer` objects.

------------------------------------------------------------------------------

-----------------------
SignedTransactionSchema
-----------------------

.. code-block:: javascript

    const SignedTransactionSchema = new Schema({
      block: {
        type: Number,
        length: 4,
        required: true
      },
      transfers: {
        type: [TransferSchema]
      },
      signatures: {
        type: [SignatureSchema]
      }
    })

A `SignedTransactionSchema` is composed of one or more `Transfer` objects and a `Signature` for each `Transfer`.
