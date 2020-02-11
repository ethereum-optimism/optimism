===============
OperatorService
===============

``OperatorService`` handles all interaction with the operator_.
This includes things like sending transactions and pulling any pending transactions.

------------------------------------------------------------------------------

getNextBlock
============

.. code-block:: javascript

    operator.getNextBlock()

Returns the next block that will be submitted.

-------
Returns
-------

``Promise<number>``: Next block number.

------------------------------------------------------------------------------

getEthInfo
==========

.. code-block:: javascript

    operator.getEthInfo()

Returns information about the smart contract.

-------
Returns
-------

``Promise<Object>``: Smart contract info.

------------------------------------------------------------------------------

getTransactions
===============

.. code-block:: javascript

    operator.getTransactions(address, startBlock, endBlock)

Returns a list of transactions received by an address between two blocks.

----------
Parameters
----------

1. ``address`` - ``string``: Address to query.
2. ``startBlock`` - ``number``: Block to query from.
3. ``endBlock`` - ``number``: Block to query to.

-------
Returns
-------

``Promise<Array>``: List of encoded transactions.

------------------------------------------------------------------------------

getTransaction
==============

.. code-block:: javascript

    operator.getTransaction(encoded)

Returns a transaction proof for a given transaction.

----------
Parameters
----------

1. ``encoded`` - ``string``: The encoded transaction.

-------
Returns
-------

``Promise<Object>``: Proof information for the transaction.

------------------------------------------------------------------------------

sendTransaction
===============

.. code-block:: javascript

    operator.sendTransaction(transaction)

Sends a SignedTransaction_ to the operator.

----------
Parameters
----------

1. ``transaction`` - ``string``: The encoded SignedTransaction_.

-------
Returns
-------

``Promise<string>``: The transaction receipt.

------------------------------------------------------------------------------

submitBlock
===========

.. code-block:: javascript

    operator.submitBlock()

Attempts to have the operator submit a new block.
Won't work if the operator is properly configured, but used for testing.


.. _operator: specs/operator.html
.. _transaction relay: TODO
.. _Transaction: specs/transactions.html#transaction-object
