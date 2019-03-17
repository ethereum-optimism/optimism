============
ChainService
============

``ChainService`` does most of the heavy lifting when it comes to receiving and sending transactions.
This service handles processing new transactions and computing the latest state.

------------------------------------------------------------------------------

getBalances
===========

.. code-block:: javascript

    chain.getBalances(address)

Returns a list of balances for a user.

----------
Parameters
----------

1. ``address`` - ``string``: Address of the user to query.

-------
Returns
-------

``Promise<Object>``: An object where keys are tokens and values are the balances.

-------
Example
-------

.. code-block:: javascript

    const balances = await chain.getBalances(address)
    console.log(balances)
    > { '0': '1194501', '1': '919ff01' }

------------------------------------------------------------------------------

addDeposits
===========

.. code-block:: javascript

    chain.addDeposits(deposits)

Applies a series of deposits to the state.

----------
Parameters
----------

1. ``deposits`` - ``Array<Deposit>``: An array of Deposit_ objects to apply.

------------------------------------------------------------------------------

getExitsWithStatus
==================

.. code-block:: javascript

    chain.getExitsWithStatus(address)

Returns any exits started by a specific user.
Identifies exits that are finalized or ready to be finalized.

----------
Parameters
----------

1. ``address`` - ``string``: Address of the user to query.

-------
Returns
-------

``Array<Exit>``: An array of Exits_ started by the user.

------------------------------------------------------------------------------

addExit
=======

.. code-block:: javascript

    chain.addExit(exit)

Applies an exit to the local state.
Internally, sends the exited range to the zero address.

----------
Parameters
----------

1. ``exit`` - ``Exit``: An Exit_ to apply.

------------------------------------------------------------------------------

pickRanges
==========

.. code-block:: javascript

    chain.pickRanges(address, token, amount)

Picks the best ranges_ to use for a transaction.

----------
Parameters
----------

1. ``address`` - ``string``: Address sending the transaction.
2. ``token`` - ``BigNum``: ID_ of the token being sent.
3. ``amoun`` - ``BigNum``: Amount of the token being sent.

-------
Returns
-------

``Array<Range>``: Best ranges for the transaction.

------------------------------------------------------------------------------

pickTransfers
=============

.. code-block:: javascript

    chain.pickTransfers(address, token, amount)

Picks the best Transfers_ to use for an exit.
This is currently necessary because of a `quirk in how we're processing exits`_.

----------
Parameters
----------

1. ``address`` - ``string``: Address sending the transaction.
2. ``token`` - ``BigNum``: ID_ of the token being sent.
3. ``amoun`` - ``BigNum``: Amount of the token being sent.

-------
Returns
-------

``Array<Range>``: Best ranges for the transaction.

------------------------------------------------------------------------------

startExit
=========

.. code-block:: javascript

    chain.startExit(address, token, amount)

Attempts to start an exit for a user.
May submit more than one exit if neccessary to withdraw the entire amount.

----------
Parameters
----------

1. ``address`` - ``string``: Account to withdraw from.
2. ``token`` - ``BigNum``: ID of the token to exit.
3. ``amount`` - ``BigNum``: Amount to exit.

-------
Returns
-------

``Array<String>``: An array of Ethereum transaction hashes.

------------------------------------------------------------------------------

finalizeExits
=============

.. code-block:: javascript

    chain.finalizeExits(address)

Attempts to finalize all pending exits for an account.

----------
Parameters
----------

1. ``address`` - ``string``: Address to finalize exits for.

-------
Returns
-------

``Array<String>``: An array of Etheruem transaction hashes.

------------------------------------------------------------------------------

sendTransaction
===============

.. code-block:: javascript

    chain.sendTransaction(transaction)

Sends a transaction to the operator.

----------
Parameters
----------

1. ``transaction`` - ``Transaction``: Transaction_ to be sent.

-------
Returns
-------

``string``: The transaction receipt.

------------------------------------------------------------------------------

loadState
=========

.. code-block:: javascript

    chain.loadState()

Loads the current head state as a SnapshotManager_.

-------
Returns
-------

``SnapshotManager``: The current head state.

------------------------------------------------------------------------------

saveState
=========

.. code-block:: javascript

    chain.saveState(stateManager)

Saves the current head state from a SnapshotManager_.

----------
Parameters
----------

1. ``stateManager`` - ``SnapshotManager``: A SnapshotManager_ to save.


.. _Deposit: TODO
.. _Exit: TODO
.. _Exits: TODO
.. _Transfers: TODO
.. _Transaction: specs/transactions.html#transaction-object
.. _Proof: specs/proofs.html#proof-object
.. _ranges: TODO
.. _ID: TODO
.. _SnapshotManager: TODO
.. _`quirk in how we're processing exits`: TODO
