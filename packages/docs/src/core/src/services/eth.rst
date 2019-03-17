==========
ETHService
==========

``ETHService`` handles Ethereum-related requests that aren't contract calls.
For contract calls, see ContractService_.

------------------------------------------------------------------------------

getBalance
==========

.. code-block:: javascript

    eth.getBalance(address)

Returns the ETH balance of an address.
Queries the main chain, not the plasma chain.

----------
Parameters
----------

1. ``address`` - ``string``: Address to query.

-------
Returns
-------

``Promise<BigNum>``: The balance of the address.

------------------------------------------------------------------------------

getCurrentBlock
===============

.. code-block:: javascript

    eth.getCurrentBlock()

Returns the latest Ethereum block number.

-------
Returns
-------

``Promise<number>``: Latest ETH block number.


.. _ContractService: services/contract.html
