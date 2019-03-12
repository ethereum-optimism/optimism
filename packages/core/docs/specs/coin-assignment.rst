==================
Coin ID Assignment
==================
This explains the process by which assets deposited into our plasma chain are assigned unique identifiers.

You usually won't have to think about these things when you're interacting with a PG plasma chains.
Most of this is handled automagically by ``plasma-utils`` or ``plasma-js-lib``.
The exact byte-per-byte binary representations of all data structures for each structure can be found in our schemas_.

Coin IDs
========
The base unit of any asset on our chain is the "coin".
Just like coins in Plasma Cash, these coins are non-fungible.
Each coin is given a unique 16 byte identifier, ``Coin ID``.
Coin IDs are assigned to assets in deposit-order on a per-asset (ERC 20/ETH) basis.

Note that all assets in the chain share the same ID-space, even if they're different ERC20s or ETH.
This means that transactions across all asset classes (which we refer to as the ``tokenType`` or ``token``) share the same tree.
This decision was made primarily as an optimization to make the transaction tree as small as possible.
However, it's important because it introduces some added complexity to coin IDs.

Token Type
----------
The first 4 bytes of a coin's ID refer to the ``tokenType`` of a coin.
``tokenType`` is assigned to a given token depending on when that token was listed in the `smart contract`_.
We give ETH a ``tokenType`` of ``00000000``.

Token ID
--------
The next 12 bytes of the ID represents the actual ID of that specific token.
Note that two different coins can have the same "Token ID" as long as they have a different ``tokenType``. 

Denominations
=============
The base denomination of each asset is automatically drawn from the asset's smart contract.
If it's an ERC20, we look at the ``decimals`` variable. 
If no ``decimals`` is available, we set the denomination to "1".
The base denomination of ETH is ``wei``.

Examples
========

ETH Deposit
-----------
Let's say a user deposits ``2 wei`` into the plasma chain contract.
That user will be given spending rights for two coins, ``00000000000000000000000000000000`` and ``00000000000000000000000000000001``.
The total coins received per deposit is precisely ``(amount of token deposited)/(minimum token denomination)``.

ERC20 Deposit
-------------
Let's say that the ``tokenType`` for ``DAI`` is ``00000001` and the base denomination is ``0.1 DAI``.
If the first user to deposit ``DAi`` deposits ``0.5 DAI``, they'll recieve the coins ``0x00000001000000000000000000000000`` up to and including coin ``0x00000001000000000000000000000004``.

.. _schemas: https://github.com/plasma-group/plasma-utils/tree/master/src/serialization/schemas
.. _smart contract: https://github.com/plasma-group/plasma-contracts/blob/master/contracts/PlasmaChain.vy
