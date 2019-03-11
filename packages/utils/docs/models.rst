======
Models
======

``plasma-utils`` provides several different "models" that represent the various common data structures we use.
These models implement the schemas_ that we use in serialization.

It's pretty simple to import all of the available models:

.. code-block:: javascript

    const utils = require('plasma-utils')
    const models = utils.serialization.models

-----------------------------------------------------------------------------

Transfer
========

A ``Transfer`` is the basic component of every transaction.
Every transaction has one or more transfers.

.. code-block:: javascript

    const Transfer = models.Transfer

    const transfer = new Transfer({
      token: 0,
      start: 0,
      end: 100,
      sender: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
      recipient: '0x8508c8aCA521512D4695eCF6976d2e8D2666a46d'
    })

-----------------------------------------------------------------------------

----------
Properties
----------

-----------------------------------------------------------------------------

token
~~~~~

.. code-block:: javascript

    transfer.token

ID of the token being transferred.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The token ID.

-----------------------------------------------------------------------------

start
~~~~~

.. code-block:: javascript

    transfer.start

Start of the range being transferred.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The range start.

-----------------------------------------------------------------------------

typedStart
~~~~~~~~~~

.. code-block:: javascript

    transfer.typedStart

The "typed" start of the range being transferred.
Calculated by concatenating ``token`` and ``start``.
Primarily used for calculating state updates in ``plasma-core``.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The typed start.

-----------------------------------------------------------------------------

end
~~~

.. code-block:: javascript

    transfer.end

End of the range being transferred.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The range end.

-----------------------------------------------------------------------------

typedEnd
~~~~~~~~

.. code-block:: javascript

    transfer.typedEnd

The "typed" end of the range being transferred.
Calculated by concatenating ``token`` and ``end``.
Primarily used for calculating state updates in ``plasma-core``.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The typed end.

-----------------------------------------------------------------------------

sender
~~~~~~

.. code-block:: javascript

    transfer.sender

Address of the user sending the transfer.

~~~~~~~
Returns
~~~~~~~

``string``: Sender address.

-----------------------------------------------------------------------------

recipient
~~~~~~~~~

.. code-block:: javascript

    transfer.recipient

Address of the user receiving the transfer.

~~~~~~~
Returns
~~~~~~~

``string``: Recipient address.

-----------------------------------------------------------------------------

encoded
~~~~~~~

.. code-block:: javascript

    transfer.encoded

The encoded version of the transfer according to the rules in our schemas_.

~~~~~~~
Returns
~~~~~~~

``string``: The encoded transfer.

-----------------------------------------------------------------------------

UnsignedTransaction
===================

An ``UnsignedTransaction`` contains transfers and a block number, but no signatures.

.. code-block:: javascript

    const UnsignedTransaction = models.UnsignedTransaction

    const unsigned = new UnsignedTransaction({
      block: 123,
      transfers: [
        {
          token: 0,
          start: 0,
          end: 100,
          sender: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
          recipient: '0x8508c8aCA521512D4695eCF6976d2e8D2666a46d'
        }
      ]
    })

-----------------------------------------------------------------------------

----------
Properties
----------

-----------------------------------------------------------------------------

block
~~~~~

.. code-block:: javascript

    unsigned.block

The block in which this transaction was included.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The transaction block number.

-----------------------------------------------------------------------------

transfers
~~~~~~~~~

.. code-block:: javascript

    unsigned.transfers

A list of ``Transfers`` that make up this transaction.

~~~~~~~
Returns
~~~~~~~

``Array<Transfer>``: A list of transfers.

-----------------------------------------------------------------------------

encoded
~~~~~~~

.. code-block:: javascript

    unsigned.encoded

The hex-encoded version of this transaction.

~~~~~~~
Returns
~~~~~~~

``string``: Encoded transaction.

-----------------------------------------------------------------------------

hash
~~~~

.. code-block:: javascript

    unsigned.hash

The keccak256 (Ethereum's SHA3) hash of the encoded transaction.

~~~~~~~
Returns
~~~~~~~

``string``: Hash of the transaction.

-----------------------------------------------------------------------------

SignedTransaction
=================

An ``SignedTransaction`` contains transfers, and a block number, and a signature for each transfer.

.. code-block:: javascript

    const SignedTransaction = models.SignedTransaction

    const signed = new SignedTransaction({
      block: 123,
      transfers: [
        {
          token: 0,
          start: 0,
          end: 100,
          sender: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
          recipient: '0x8508c8aCA521512D4695eCF6976d2e8D2666a46d'
        }
      ],
      signatures: [
        {
          v: '0x1b',
          r: '0xd693b532a80fed6392b428604171fb32fdbf953728a3a7ecc7d4062b1652c042',
          s: '0x24e9c602ac800b983b035700a14b23f78a253ab762deab5dc27e3555a750b354'
        }
      ]
    })

-----------------------------------------------------------------------------

----------
Properties
----------

-----------------------------------------------------------------------------

block
~~~~~

.. code-block:: javascript

    signed.block

The block in which this transaction was included.

~~~~~~~
Returns
~~~~~~~

``BigNum``: The transaction block number.

-----------------------------------------------------------------------------

transfers
~~~~~~~~~

.. code-block:: javascript

    signed.transfers

A list of ``Transfers`` that make up this transaction.

~~~~~~~
Returns
~~~~~~~

``Array<Transfer>``: A list of transfers.

-----------------------------------------------------------------------------

signatures
~~~~~~~~~~

.. code-block:: javascript

    signed.signatures

A list of ``Signatures`` on this transaction.
There should be one signature for each transfer, where the signature is from the sender of the transfer.

~~~~~~~
Returns
~~~~~~~

``Array<Signature>``: A list of signatures.

-----------------------------------------------------------------------------

encoded
~~~~~~~

.. code-block:: javascript

    signed.encoded

The hex-encoded version of this transaction.

~~~~~~~
Returns
~~~~~~~

``string``: Encoded transaction.

-----------------------------------------------------------------------------

hash
~~~~

.. code-block:: javascript

    signed.hash

The keccak256 (Ethereum's SHA3) hash of the encoded *unsigned* version of this transaction.
Effectively the same as casting this transaction to an ``UnsignedTransaction`` and getting the hash.

~~~~~~~
Returns
~~~~~~~

``string``: Hash of the *unsigned* version of this transaction.

-----------------------------------------------------------------------------

-------
Methods
-------

-----------------------------------------------------------------------------

checkSigs
~~~~~~~~~

.. code-block:: javascript

    signed.checkSigs()

Checks that the signatures on the transaction are valid.

~~~~~~~
Returns
~~~~~~~

``boolean``: ``true`` if the transaction is valid, ``false`` otherwise.

.. _schemas: serialization.html
