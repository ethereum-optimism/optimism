=======
ChainDB
=======

``ChainDB`` handles chain-related database calls, like accessing block headers or transactions.

------------------------------------------------------------------------------

getTransaction
==============

.. code-block:: javascript

    chaindb.getTransaction(hash)

Returns the transaction with the given hash.

----------
Parameters
----------

1. ``hash`` - ``string``: Hash of the transaction to query.

-------
Returns
-------

``Object``: A Transaction_ object.

------------------------------------------------------------------------------

setTransaction
==============

.. code-block:: javascript

    chaindb.setTransaction(transaction)

Adds a SignedTransaction_ to the database.

----------
Parameters
----------

1. ``transaction`` - ``SignedTransaction``: Transaction to store.

------------------------------------------------------------------------------

hasTransaction
==============

.. code-block:: javascript

    chaindb.hasTransaction(hash)

Checks if the database has a specific transaction.

----------
Parameters
----------

1. ``hash`` - ``string``: Hash of the transaction to check.

-------
Returns
-------

``boolean``: ``true`` if the database has the transaction, ``false`` otherwise.

------------------------------------------------------------------------------

getLatestBlock
==============

.. code-block:: javascript

    chaindb.getLatestBlock()

Returns the number of the last stored block.

-------
Returns
-------

``number``: Latest block number.

------------------------------------------------------------------------------

setLatestBlock
==============

.. code-block:: javascript

    chaindb.setLatestBlock(block)

Sets the latest block number.
Will only set if ``block`` actually is later than the latest.

----------
Parameters
----------

1. ``block`` - ``number``: Latest block number.

------------------------------------------------------------------------------

getBlockHeader
==============

.. code-block:: javascript

    chaindb.getBlockHeader(block)

Returns the header of the block with the given number.

----------
Parameters
----------

1. ``block`` - ``number``: Number of the block to query.

-------
Returns
-------

``string``: A block hash.

------------------------------------------------------------------------------

addBlockHeader
==============

.. code-block:: javascript

    chaindb.addBlockHeader(block, header)

Stores a block header.

----------
Parameters
----------

1. ``block`` - ``number``: Number of the block to store.
2. ``header`` - ``string``: Hash of the given block.

------------------------------------------------------------------------------

addBlockHeaders
===============

.. code-block:: javascript

    chaindb.addBlockHeaders(blocks)

Adds multiple block headers to the database simultaneously.
More efficient than adding several block headers with ``addBlockHeader``.

----------
Parameters
----------

1. ``blocks`` - ``Array<Block>``: An array of Block_ objects.

------------------------------------------------------------------------------

getDeposits
===========

.. code-block:: javascript

    chaindb.getDeposits(address)

Returns a list of known deposits for an address.

----------
Parameters
----------

1. ``address`` - ``string``: Address to query.

-------
Returns
-------

``Array<Deposit>``: A list of Deposit_ objects for that address.

------------------------------------------------------------------------------

getExits
========

.. code-block:: javascript

    chaindb.getExits(address)

Returns the list of known exits for an address.

----------
Parameters
----------

1. ``address`` - ``string``: Address to query.

-------
Returns
-------

``Array<Exit>``: A list of Exit_ objects for that address.

------------------------------------------------------------------------------

addExit
=======

.. code-block:: javascript

    chaindb.addExit(exit)

Adds an Exit_ to the database.

----------
Parameters
----------

1. ``exit`` - ``Exit``: Exit_ to add to the database.

------------------------------------------------------------------------------

addExitableEnd
==============

.. code-block:: javascript

    chaindb.addExitableEnd(token, end)

Adds an "exitable end" to the database.
See `this article`_ for more information.

----------
Parameters
----------

1. ``token`` - ``BigNum``: Token of the range.
2. ``end`` - ``BigNum``: End of the range.

------------------------------------------------------------------------------

addExitableEnds
===============

.. code-block:: javascript

    chaindb.addExitableEnds(exitables)

Adds several "exitable ends" to the database.
More efficient than calling ``addExitableEnd`` multiple times.

----------
Parameters
----------

1. ``exitables`` - ``Array<{ BigNum, BigNum }>``: An array of objects with a ``token`` and ``end``.

------------------------------------------------------------------------------

getExitableEnd
==================

.. code-block:: javascript

    chaindb.getExitableEnd(token, end)

Returns the correct "exitable end" for a range.

----------
Parameters
----------

1. ``token`` - ``BigNum``: Token of the range.
2. ``end`` - ``BigNum``: End of the range.

-------
Returns
-------

``BigNum``: The exitable end.

------------------------------------------------------------------------------

markExited
==========

.. code-block:: javascript

    chaindb.markExited(range)

Marks a specific range as "exited".

----------
Parameters
----------

1. ``range`` - ``Range``: Range_ to mark as exited.

------------------------------------------------------------------------------

checkExited
===========

.. code-block:: javascript

    chaindb.checkExited(range)

Checks if a Range_ is marked as exited.

----------
Parameters
----------

1. ``range`` - ``Range``: Range_ to check.

-------
Returns
-------

``boolean``: ``true`` if the range is exited, ``false`` otherwise.

------------------------------------------------------------------------------

markFinalized
=============

.. code-block:: javascript

    chaindb.markFinalized(exit)

Marks an exit as finalized.

----------
Parameters
----------

1. ``exit`` - ``Exit``: Exit_ to mark as finalized.

------------------------------------------------------------------------------

checkFinalized
==============

.. code-block:: javascript

    chaindb.checkFinalized(exit)

Checks if an exit is marked as finalized.

----------
Parameters
----------

1. ``exit`` - ``Exit``: Exit to check.

-------
Returns
-------

``boolean``: ``true`` if the exit is finalized, ``false`` otherwise.

------------------------------------------------------------------------------

getState
========

.. code-block:: javascript

    chaindb.getState()

Returns the latest head state.

-------
Returns
-------

``Array<Snapshot>``: The head state as a list of Snapshots_.

------------------------------------------------------------------------------

setState
========

.. code-block:: javascript

    chaindb.setState(state)

Sets the latest head state.

----------
Parameters
----------

1. ``state`` - ``Array<Snapshot>``: A list of snapshots that represent the state.

------------------------------------------------------------------------------

getTypedValue
=============

.. code-block:: javascript

    chaindb.getTypedValue(token, value)

Returns the "typed" version of a start or end.
See our `explanation of coin IDs`_ for more information.

----------
Parameters
----------

1. ``token`` - ``BigNum``: Token ID.
2. ``value`` - ``BigNum``: Value to type.

-------
Returns
-------

``string``: The typed value.


.. _SignedTransaction: TODO
.. _Block: TODO
.. _Deposit: TODO
.. _Exit: TODO
.. _`this article`: https://github.com/plasma-group/plasma-contracts/issues/44
.. _Range: TODO
.. _Snapshots: TODO
.. _`explanation of coin IDs`: TODO
