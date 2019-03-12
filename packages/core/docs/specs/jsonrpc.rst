==============
JSON-RPC Calls
==============

pg_getBalance
=============
.. code-block:: javascript

    pg_getBalance

Returns the balance of a specific account.

----------
Parameters
----------

1. ``address`` - ``string``: Address of the account to query.

-------
Returns
-------

``Array``: A list of token balances in the form `(token, balance)`.

------------------------------------------------------------------------------

pg_getBlock
===========
.. code-block:: javascript

    pg_getBlock

Pulls the hash of the block at a specific height.

----------
Parameters
----------

1. ``block`` - ``number``: Number of the block to query.

-------
Returns
-------

``string``: The block hash.

------------------------------------------------------------------------------

pg_getTransaction
=================
.. code-block:: javascript

    pg_getTransaction

Pulls information about a specific transaction.

----------
Parameters
----------

1. ``hash`` - ``string``: The hash of the transaction.

-------
Returns
-------

``SignedTransaction``: The specified transaction.

------------------------------------------------------------------------------

pg_sendTransaction
==================
.. code-block:: javascript

    pg_sendTransaction

Sends a transaction to the node to be processed.

----------
Parameters
----------

1. ``transaction`` - ``Object``:
    * ``from`` - ``string``: Address from which the transaction was sent.
    * ``to`` - ``string``: Address to which the transaction was sent.
    * ``token`` - ``string``: ID of the token to be sent.
    * ``value`` - ``number``: Value of tokens to be sent.

-------
Returns
-------

``string``: The transaction receipt.

------------------------------------------------------------------------------

pg_sendRawTransaction
=====================
.. code-block:: javascript

    pg_sendRawTransaction

Sends an encoded SignedTransaction_ to the node to be processed.

----------
Parameters
----------

1. ``transaction`` - ``string``: Encoded signed transaction.

-------
Returns
-------

``string``: The transaction receipt.

------------------------------------------------------------------------------

pg_getHeight
============
.. code-block:: javascript

    pg_getHeight

Returns the current plasma block height.

-------
Returns
-------

``number``: The current block height.

------------------------------------------------------------------------------

pg_getRecentTransactions
========================
.. code-block:: javascript

    pg_getRecentTransactions

Returns the most recent transactions.
Because there are a *lot* of transactions in each block, this method is paginated.

----------
Parameters
----------

1. ``start`` - ``number``: Start of the range of recent transactions to return.
2. ``end`` - ``number``: End of range of recent transactions to return.

-------
Returns
-------

``Array<SignedTransaction>``: A list of SignedTransaction_ objects.

------------------------------------------------------------------------------

pg_getAccounts
==============
.. code-block:: javascript

    pg_getAccounts

Returns a list of all available accounts.

-------
Returns
-------

``Array<string>``: A list of account addresses.

------------------------------------------------------------------------------

pg_getTransactionsByAddress
===========================
.. code-block:: javascript

    pg_getTransactionsByAddress

Returns the latest transactions by an address.
This method is paginated and requires a ``start`` and ``end``.
Limited to a total of **25** transactions at a time.

----------
Parameters
----------

1. ``address - ``string``: The address to query.
2. ``start`` - ``number``: Start of the range of recent transactions to return.
3. ``end`` - ``number``: End of range of recent transactions to return.

-------
Returns
-------

``Array<SignedTransaction>``: A list of SignedTransaction_ objects.

.. _SignedTransaction: https://plasma-utils.readthedocs.io/en/latest/models.html#signedtransaction
