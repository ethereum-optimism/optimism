================
ContractProvider
================

``ContractProvider`` is a wrapper that interacts with the plasma chain smart contract.

------------------------------------------------------------------------------

address
=======

.. code-block:: javascript

    contract.address

Returns the contract's address.

-------
Returns
-------

``string``: Address of the connected contract.

------------------------------------------------------------------------------

hasAddress
==========

.. code-block:: javascript

    contract.hasAddress

Whether or not the contract has an address.

-------
Returns
-------

``boolean``: ``true`` if the contract is ready to be used, ``false`` otherwise.

------------------------------------------------------------------------------

ready
=====

.. code-block:: javascript

    contract.ready

Whether or not the contract is ready to be used.

-------
Returns
-------

``boolean``: ``true`` if hte contract is ready, ``false`` otherwise.

------------------------------------------------------------------------------

web3
====

.. code-block:: javascript

    contract.web3

Returns the web3 instance being used by the contract.

-------
Returns
-------

``Web3``: Contract's web3 instance.

------------------------------------------------------------------------------

plasmaChainName
===============

.. code-block:: javascript

    contract.plasmaChainName

Name of the plasma chain this contract is connected to.

-------
Returns
-------

``string``: Plasma chain name.

------------------------------------------------------------------------------

checkAccountUnlocked
====================

.. code-block:: javascript

    contract.checkAccountUnlocked(address)

Checks whether an account is unlocked and attempts to unlock it if not.

----------
Parameters
----------

1. ``address`` - ``string``: Address of the account to check.

------------------------------------------------------------------------------

getBlock
========

.. code-block:: javascript

    contract.getBlock(block)

Queries the hash of a given block.

----------
Parameters
----------

1. ``block`` - ``number``: Number of the block to query.

-------
Returns
-------

``Promise<string>``: Root hash of the block with that number.

------------------------------------------------------------------------------

getNextBlock
============

.. code-block:: javascript

    contract.getNextBlock()

Returns the number of the next block that will be submitted.

-------
Returns
-------

``Promise<number>``: Next block number.

------------------------------------------------------------------------------

getCurrentBlock
===============

.. code-block:: javascript

    contract.getCurrentBlock()

Returns the number of the last block to be submitted.

-------
Returns
-------

``Promise<number>``: Last block number.

------------------------------------------------------------------------------

getOperator
===========

.. code-block:: javascript

    contract.getOperator()

Returns the address of the operator.

-------
Returns
-------

``Promise<string>``: Plasma chain operator address.

------------------------------------------------------------------------------

getTokenAddress
===============

.. code-block:: javascript

    contract.getTokenAddress(token)

Returns the address for a given token ID.

----------
Parameters
----------

1. ``token`` - ``string``: A token ID.

-------
Returns
-------

``Promise<string>``: Address of the contract for that token.

------------------------------------------------------------------------------

listToken
=========

.. code-block:: javascript

    contract.listToken(tokenAddress)

Lists a token with the given address so that it can be deposited.

----------
Parameters
----------

1. ``tokenAddress`` - ``string``: Address of the token to list.

-------
Returns
-------

``EthereumTransaction``: The Ethereum transaction result.

------------------------------------------------------------------------------

getChallengePeriod
==================

.. code-block:: javascript

    contract.getChallengePeriod()

Returns the current challenge period in number of blocks.

-------
Returns
-------

``Promise<number>``: Challenge period.

------------------------------------------------------------------------------

getTokenId
==========

.. code-block:: javascript

    contract.getTokenId(tokenAddress)

Gets the token ID for a specific token.

----------
Parameters
----------

1. ``tokenAddress`` - ``string``: Token contract address.

-------
Returns
-------

``Promise<string>``: ID of the token.

------------------------------------------------------------------------------

depositValid
============

.. code-block:: javascript

    contract.depositValid(deposit)

Checks whether a Deposit_ actually exists.
Used when checking transaction proofs.

----------
Parameters
----------

1. ``deposit`` - ``Deposit``: A Deposit_ to validate.

-------
Returns
-------

``boolean``: ``true`` if the deposit exists, ``false`` otherwise.

------------------------------------------------------------------------------

deposit
=======

.. code-block:: javascript

    contract.deposit(address, token, amount)

Deposits some value of a token to the plasma smart contract.

----------
Parameters
----------

1. ``address`` - ``string``: Address to deposit with.
1. ``token`` - ``string``: Address of the token to deposit.
2. ``amount`` - ``number``: Amount to deposit.

-------
Returns
-------

``EthereumTransaction``: An Ethereum transaction receipt.

------------------------------------------------------------------------------

startExit
=========

.. code-block:: javascript

    contract.startExit(block, token, start, end, owner)

Starts an exit for a user.
Exits can only be started on *transfers*, meaning you need to specify the block in which the transfer was received.

----------
Parameters
----------

1. ``block`` - ``BigNum``: Block in which the transfer was received.
2. ``token`` - ``BigNum``: Token to be exited.
3. ``start`` - ``BigNum``: Starts of the range received in the transfer.
4. ``end`` - ``BigNum``: End of the range received in the transfer.
5. ``owner`` - ``string``: Address to withdraw from.

-------
Returns
-------

``EthereumTransaction``: Exit transaction receipt.

------------------------------------------------------------------------------

finalizeExit
============

.. code-block:: javascript

    contract.finalizeExit(exitId, exitableEnd, owner)

Finalizes an exit for a user.

----------
Parameters
----------

1. ``exitId`` - ``string``: ID of the exit to finalize.
2. ``exitableEnd`` - ``BigNum``: The "exitable end" for that exit.
3. ``owner`` - ``string``: Address that owns the exit.

-------
Returns
-------

``EthereumTransaction``: Finalization transaction receipt.

------------------------------------------------------------------------------

submitBlock
===========

.. code-block:: javascript

    contract.submitBlock(hash)

Submits a block with the given hash.
Will only work if the operator's account is unlocked and available to the node.

----------
Parameters
----------

1. ``hash`` - ``string``: Hash of the block to submit.

-------
Returns
-------

``EthereumTransaction``: Block submission transaction receipt.


.. _Deposit: TODO
