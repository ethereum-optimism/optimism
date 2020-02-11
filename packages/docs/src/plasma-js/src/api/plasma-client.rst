============
PlasmaClient
============

``PlasmaClient`` handles interaction with plasma clients that implement the `PG JSON-RPC Calls`_

.. code-block:: javascript

    const PlasmaClient = require('@pigi/plasma-js')

    // Connects automatically to http://localhost:9898
    const plasma = new PlasmaClient()

------------------------------------------------------------------------------

getAccounts
===========

.. code-block:: javascript

    plasma.getAccounts()

Returns the list of available accounts.

-------
Returns
-------

``Promise<Array>``: List of addresses controlled by the node.

-------
Example
-------

.. code-block:: javascript

    const accounts = await plasma.getAccounts()
    console.log(accounts)
    > [ '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
        '0x946E85B7C74a89f2710BEa6Cb83d4c1AEA40682F',
        '0xbF699b0d6e59B865d74D9D1714A407f6516B0F60' ]

------------------------------------------------------------------------------

getBalances
===========

.. code-block:: javascript

    plasma.getBalances(address)

Returns all token balances for an address.
Balances are returned as BigNum.

----------
Parameters
----------

1. ``address`` - ``string``: Address to return balances for.

-------
Returns
-------

``Promise<Object>``: A mapping of token IDs to account balances.

-------
Example
-------

.. code-block:: javascript

    const balances = await plasma.getBalances('0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6')
    console.log(balances)
    > { '0': <BN: 64> }

------------------------------------------------------------------------------

getExits
========

.. code-block:: javascript

    plasma.getExits(address)

Returns all active exits for an address.

----------
Parameters
----------

1. ``address`` - ``string``: Address to return exits for.

-------
Returns
-------

``Promise<Array>``: List of exits.

-------
Example
-------

.. code-block:: javascript

    const exits = await plasma.getExits('0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6')
    console.log(exits)
    > [ { token: '0',
          start: '384',
          end: '3e8',
          id: '0',
          block: '3a5b57',
          exiter: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
          completed: true,
          finalized: false } ]

------------------------------------------------------------------------------

getTransaction
==============

.. code-block:: javascript

    plasma.getTransaction(hash)

Returns a transaction given its hash.

----------
Parameters
----------

1. ``hash`` - ``string``: Hash of the transaction to return.

-------
Returns
-------

``Promise<SignedTransaction>``: Transaction with the given hash.

-------
Example
-------

.. code-block:: javascript

    const transaction = await plasma.getTransaction('0xae5ac607d29c6d38a63db00550160b5ca3b51ec9b3ede8dcb5755b60700aecfe')
    console.log(transaction)
    > SignedTransaction {
        schema:
          Schema {
            unparsedFields:
              { block: [Object], transfers: [Object], signatures: [Object] },
                fields:
                  { block: [SchemaNumber],
                    transfers: [Schema],
                    signatures: [Schema] } },
        block: <BN: 389e>,
        transfers:
          [ { sender: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6',
              recipient: '0x946E85B7C74a89f2710BEa6Cb83d4c1AEA40682F',
              token: <BN: 0>,
              start: <BN: 3e8>,
              end: <BN: 44c> } ],
        signatures:
          [ { v: <Buffer 1b>,
              r: <Buffer 07 78 c7 ba a3 df 5e 4d 39 ff 4a 17 63 f5 53 84 4a 30 b5 47 1a 75 71 06 f7 a5 f7 e2 f7 00 91 1b>,
              s: <Buffer 53 94 5b 03 2d fb a3 4d 9f 59 13 a1 06 ff 09 0e 88 b6 19 4b 27 74 9a c4 e9 31 17 2c 0c b7 6e d1> } ] }

------------------------------------------------------------------------------

getBlock
========

.. code-block:: javascript

    plasma.getBlock(block)

Returns the hash of the plasma block with the given number.

----------
Parameters
----------

1. ``block`` - ``number``: Number of the block to query.

-------
Returns
-------

``Promise<string>``: Hash of the block with that number.

-------
Example
-------

.. code-block:: javascript

    const block = await plasma.getBlock(15)
    console.log(block)
    > 0x26e5955d5db3d1fb3fd4142fbf91daa9d8f6b58f0612c6e52eee79ca7755b004

------------------------------------------------------------------------------

getCurrentBlock
===============

.. code-block:: javascript

    plasma.getCurrentBlock()

Returns the number of the most recently submitted block.

-------
Returns
-------

``Promise<number>``: Last submitted block number.

-------
Example
-------

.. code-block:: javascript

    const currentBlock = await plasma.getCurrentBlock()
    console.log(currentBlock)
    > 5442

------------------------------------------------------------------------------

getNextBlock
============

.. code-block:: javascript

    plasma.getNextBlock()

Returns the number of the plasma block that will be submitted next.

-------
Returns
-------

``Promise<number>``: Next plasma block number.

-------
Example
-------

.. code-block:: javascript

    const nextBlock = await plasma.getNextBlock()
    console.log(nextBlock)
    > 5443

------------------------------------------------------------------------------

getTokenId
==========

.. code-block:: javascript

    plasma.getTokenId(tokenAddress)

Returns the `token ID`_ of the token at the given contract address.

----------
Parameters
----------

1. ``tokenAddress`` - ``string``: Address of the contract that represents the token.

-------
Returns
-------

``Promise<string>``: The token's ID.

-------
Example
-------

.. code-block:: javascript

    const tokenId = await plasma.getTokenId('0xf88ce35b57e37cda8a8520f1a290b7edef532d95)
    console.log(tokenId)
    > 1

------------------------------------------------------------------------------

createAccount
=============

.. code-block:: javascript

    plasma.createAccount()

Creates a new account.

-------
Returns
-------

``Promise<string>``: Address of the created account.

-------
Example
-------

.. code-block:: javascript

    const account = await plasma.createAccount()
    console.log(account)
    > 0x8508c8aCA521512D4695eCF6976d2e8D2666a46d

------------------------------------------------------------------------------

sign
====

.. code-block:: javascript

    plasma.sign(address, data)

Signs a message with a given account.

----------
Parameters
----------

1. ``address`` - ``string``: Address of the account to sign with.
2. ``data`` - ``string``: Message to sign.

-------
Returns
-------

``Promise<Object>``: An `Ethereum signature object`_.

-------
Example
-------

.. code-block:: javascript

    const signature = await plasma.sign('0x8508c8aCA521512D4695eCF6976d2e8D2666a46d', 'Hello!)
    console.log(siganture)
    > { message: 'Hello!',
        messageHash: '0x52b6437db56d87f5991d7c173cf11b9dd0f9fb083260bef1bf0c338042bc398c',
        v: '0x1c',
        r: '0x47de6cc9f808658d643c3fd4a79be725627f719e6604d86f7b6356f3bdb81ed3',
        s: '0x4e18918c4b0a60dfa2ce3ee623c815b90b4eb30f5a83bae5b89778ff0aa742af',
        signature: '0x47de6cc9f808658d643c3fd4a79be725627f719e6604d86f7b6356f3bdb81ed34e18918c4b0a60dfa2ce3ee623c815b90b4eb30f5a83bae5b89778ff0aa742af1c' }

------------------------------------------------------------------------------

deposit
=======

.. code-block:: javascript

    plasma.deposit(token, amount, address)

Deposits an amount of a given token for an address.

----------
Parameters
----------

1. ``token`` - ``string``: ID or address of the token to be deposited.
2. ``amount`` - ``number``: Amount to be deposited.
3. ``address`` - ``string``: Address to use to deposit.

-------
Returns
-------

``Promise<EthereumTransaction>``: An Ethereum transaction object.

-------
Example
-------

.. code-block:: javascript

    const depositTx = await plasma.deposit('1', 5000, '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6')
    console.log(depositTx)
    > { blockHash: '0x7cea9161cbf7265c2d19392888703e56f84987da8734bffd5bd6351f2098a0e0',
        blockNumber: 3824629,
        contractAddress: null,
        cumulativeGasUsed: 938742,
        from: '0x1e3a4a2edec2b3568b5ad0656ec3b48d9c699db6',
        gasUsed: 108968,
        logsBloom: '0x00000000000000000000002000000000000000000000000000000000000400000000000000000000010000000002000000000000000000000000000000000000008000000000000000000008000001040000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001100000000000000000000200000000000000000802000000000200000000000000000000000000000000420000000000000000000000000000000000000000000100000000000100000000040000000000',
        status: '0x1',
        to: '0x888c238f821fd7e62460f029adbc388aa3143932',
        transactionHash: '0x3aa45e3e674d11329e207782d30ba9893c5d34deece2dee8bfc1047ffa8f614c',
        transactionIndex: 6,
        events:
          { '0':
            { address: '0xF88Ce35b57e37Cda8a8520f1a290B7edef532D95',
              blockHash: '0x7cea9161cbf7265c2d19392888703e56f84987da8734bffd5bd6351f2098a0e0',
              blockNumber: 3824629,
              logIndex: 7,
              removed: false,
              transactionHash: '0x3aa45e3e674d11329e207782d30ba9893c5d34deece2dee8bfc1047ffa8f614c',
              transactionIndex: 6,
              id: 'log_1ab6673b',
              returnValues: {},
              signature: null,
              raw: [Object] },
            DepositEvent:
              { address: '0x888C238f821fD7e62460F029ADbC388aa3143932',
                blockHash: '0x7cea9161cbf7265c2d19392888703e56f84987da8734bffd5bd6351f2098a0e0',
                blockNumber: 3824629,
                logIndex: 8,
                removed: false,
                transactionHash: '0x3aa45e3e674d11329e207782d30ba9893c5d34deece2dee8bfc1047ffa8f614c',
                transactionIndex: 6,
                id: 'log_a5079148',
                returnValues: [Object],
                event: 'DepositEvent',
                signature: '0x7a9ec4e041f302c44606a6b6c9f3ab369e99b054e8582f4fc4d6f39240cfc810',
                raw: [Object] } } }

------------------------------------------------------------------------------

pickRanges
==========

.. code-block:: javascript

    plasma.pickRanges(address, token, amount)

Picks the best ranges to make a transaction.

----------
Parameters
----------

1. ``address`` - ``string``: Address to transact from.
2. ``token`` - ``string``: ID or address of token to send.
3. ``amount`` - ``number``: Amount to be sent.

-------
Returns
-------

``Promise<Array>``: An array of Range_ objects.

-------
Example
-------

.. code-block:: javascript

    const ranges = await plasma.pickRanges('0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6', '1', 5000)
    console.log(ranges)
    > [ { token: '1',
          start: '0',
          end: '1388',
          owner: '0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6' } ]

------------------------------------------------------------------------------

sendRawTransaction
==================

.. code-block:: javascript

    plasma.sendRawTransaction(transaction)

Sends an encoded and signed transaction to the operator.
If you're looking for an easier way to send transactions, look at ``sendTransaction`` below.

----------
Parameters
----------

1. ``transaction`` - ``string``: The encoded signed transaction.

-------
Returns
-------

``Promise<string>``: A transaction receipt.

-------
Example
-------

.. code-block:: javascript

    const receipt = await plasma.sendRawTransaction('0000389e011E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6946E85B7C74a89f2710BEa6Cb83d4c1AEA40682F000000000000000000000000000003e800000000000000000000044c011b0778c7baa3df5e4d39ff4a1763f553844a30b5471a757106f7a5f7e2f700911b53945b032dfba34d9f5913a106ff090e88b6194b27749ac4e931172c0cb76ed1')
    console.log(receipt)
    > 0000389e011E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6946E85B7C74a89f2710BEa6Cb83d4c1AEA40682F000000000000000000000000000003e800000000000000000000044c011b0778c7baa3df5e4d39ff4a1763f553844a30b5471a757106f7a5f7e2f700911b53945b032dfba34d9f5913a106ff090e88b6194b27749ac4e931172c0cb76ed1

------------------------------------------------------------------------------

sendTransaction
===============

.. code-block:: javascript

    plasma.sendTransaction(from, to, token, amount)

The method that most people should use to make transactions.
Wraps ``sendRawTransaction`` and automatically calculates the best ranges for a given transaction.
Also handles formatting and signing the transaction.

----------
Parameters
----------

1. ``from`` - ``string``: Address to send from.
2. ``to`` - ``string``: Address to send to.
3. ``token`` - ``string``: ID or address of the token to send.
4. ``amount`` - ``number``: Amount of the token to send.

-------
Returns
-------

``Promise<string>``: A transaction receipt.

-------
Example
-------

.. code-block:: javascript

    const receipt = await plasma.sendTransaction('0x82A978B3f5962A5b0957d9ee9eEf472EE55B42F1', '0x7d577a597B2742b498Cb5Cf0C26cDCD726d39E6e', '0', 50)
    console.log(receipt)
    > 000000030182A978B3f5962A5b0957d9ee9eEf472EE55B42F17d577a597B2742b498Cb5Cf0C26cDCD726d39E6e000000000000000000000000000013ba0000000000000000000013ec0101570685e98f44cc642ff081bc0314108cd5982d1b9b4646adc688ffd3960609f50fe33c24d949deb321280376b42118ca85ac6288f26a1aa1f038191a48b08b6e

------------------------------------------------------------------------------

startExit
=========

.. code-block:: javascript

    plasma.startExit(address, token, amount)

Starts exits for a user to withdraw a certain amount of a given token.
Will automatically select the right ranges to withdraw and submit more than one exit if necessary.

----------
Parameters
----------

1. ``address`` - ``string``: Address to submit exits for.
2. ``token`` - ``string``: ID or address of the token to exit.
3. ``amount`` - ``number``: Amount of the token to withdraw.

-------
Returns
-------

``Promise<Array>``: Ethereum transaction hash for each exit.

-------
Example
-------

.. code-block:: javascript

    const exitTxs = await plasma.startExit('0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6', '1', 5000)
    console.log(exitTxs)
    > [ '0xfdb32a468624233010d9648e8231327d0ff9794cc8b722c2a8539d90cb5af20c' ]

------------------------------------------------------------------------------

finalizeExits
=============

.. code-block:: javascript

    plasma.finalizeExits(address)

Finalizes all available exits for an address.
Will not finalize any exits that are still in their challenge period or have already been finalized.

----------
Parameters
----------

1. ``address`` - ``string``: Address to finalize exits for.

-------
Returns
-------

``Promise<Array>``: Ethereum transaction hash for each finalization.

-------
Example
-------

.. code-block:: javascript

    const finalizeTxs = await plasma.finalizeExits('0x1E3a4a2edec2b3568B5Ad0656ec3b48d9C699dB6')
    console.log(finalizeTxs)
    > [ '0xac8840c6ba3a948590e07f95b52647d81005de2c4f161be63f060da926a40350' ]

------------------------------------------------------------------------------

listToken
=========

.. code-block:: javascript

    plasma.listToken(tokenAddress)

Lists a new token so that it can be deposited.

----------
Parameters
----------

1. ``tokenAddress`` - ``string``: Address of the token to be deposited.

-------
Returns
-------

``Promise<EthereumTransaction>``: The transaction result.

-------
Example
-------

.. code-block:: javascript

    const listingTx = await plasma.listToken('0xf88ce35b57e37cda8a8520f1a290b7edef532d95')
    console.log(listingTx)
    > { blockHash: '0x114e62f5e92e50ed941f5ad0d63f04ad90d9677613a4897bdbc5a6f5d3774700',
        blockNumber: 3824586,
        contractAddress: null,
        cumulativeGasUsed: 676722,
        from: '0x1e3a4a2edec2b3568b5ad0656ec3b48d9c699db6',
        gasUsed: 92449,
        logsBloom: '0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001040000000000000000000000000000100000000000000000000020000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
        status: '0x1',
        to: '0x888c238f821fd7e62460f029adbc388aa3143932',
        transactionHash:
        '0x35c88e3b51109dd446319318a3be285d065943203ebb8780bb1f9717f72a051d',
        transactionIndex: 6,
        events:
          { ListingEvent:
            { address: '0x888C238f821fD7e62460F029ADbC388aa3143932',
              blockHash: '0x114e62f5e92e50ed941f5ad0d63f04ad90d9677613a4897bdbc5a6f5d3774700',
              blockNumber: 3824586,
              logIndex: 8,
              removed: false,
              transactionHash: '0x35c88e3b51109dd446319318a3be285d065943203ebb8780bb1f9717f72a051d',
              transactionIndex: 6,
              id: 'log_53a1c942',
              returnValues: [Object],
              event: 'ListingEvent',
              signature: '0x80ed85783ee3285a2a09339e1e9f1c0b2a3aa05240c97e1a741ac6347a2aca11',
              raw: [Object] } } }

.. _PG JSON-RPC Calls: https://docs.plasma.group/projects/core/en/latest/src/specs/jsonrpc.html
.. _token ID: TODO
.. _Ethereum signature object: https://web3js.readthedocs.io/en/1.0/web3-eth-accounts.html#id14
.. _Range: TODO
