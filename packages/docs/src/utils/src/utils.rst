=========
Utilities
=========
``plasma-utils.utils`` provides miscellaneous utilities often used when interacting with plasma chains.

.. code-block:: javascript

    const utils = require('plasma-utils').utils

-----------------------------------------------------------------------------

int32ToHex
==========

.. code-block:: javascript

    utils.int32ToHex(x)

Converts a 32 byte integer to a hex string.

----------
Parameters
----------

1. ``x`` - ``number``: A 32 byte integer.

-------
Returns
-------

``string``: The integer represented as a hex string.

-----------------------------------------------------------------------------

sleep
=====

.. code-block:: javascript

    utils.sleep(ms)

Sleeps for a number of milliseconds.

----------
Parameters
----------

1. ``ms`` - ``number``: Number of milliseconds to sleep.

-------
Returns
-------

``Promise``: A promise that resolves after the given number of ms.

-----------------------------------------------------------------------------

remove0x
========

.. code-block:: javascript

    utils.remove0x(str)

Removes "0x" from the start of a string, if it's present.

----------
Parameters
----------

1. ``str`` - ``string``: String to modify.

-------
Returns
-------

``string``: The string without a leading "0x".

-----------------------------------------------------------------------------

add0x
=====

.. code-block:: javascript

    utils.add0x(x)

Adds "0x" to the start of a string, if not already present.

----------
Parameters
----------

1. ``str`` - ``string``: String to modify.

-------
Returns
-------

``string``: The string with "0x".

-----------------------------------------------------------------------------

isString
========

.. code-block:: javascript

    utils.isString(str)

Checks if the input value is a string.

----------
Parameters
----------

1. ``str`` - ``any``: The thing that might be a string.

-------
Returns
-------

``boolean``: ``true`` if the input is a string, ``false`` otherwise.

-----------------------------------------------------------------------------

getRandomElement
================

.. code-block:: javascript

    utils.getRandomElement(arr)

Returns a random element from an array.

----------
Parameters
----------

1. ``arr`` - ``Array``: An array.

-------
Returns
-------

``any``: A random element from that array.

-----------------------------------------------------------------------------

getRandomAccount
================

.. code-block:: javascript

    utils.getRandomAccount()

Returns a random Ethereum account.

-------
Returns
-------

``any``: The Ethereum account.

-----------------------------------------------------------------------------

sign
====

.. code-block:: javascript

    utils.sign(data, key)

Signs a message with a private key.

----------
Parameters
----------

1. ``data`` - ``string``: Message to sign.
2. ``key`` - ``string``: Private key to sign with.

-------
Returns
-------

``string``: The signature.

-----------------------------------------------------------------------------

signatureToString
=================

.. code-block:: javascript

    utils.signatureToString(signature)

Converts a signature with v,r,s Buffers to a single hex string.

----------
Parameters
----------

1. ``signature`` - ``Object``: A signature object.

-------
Returns
-------

``string``: The signature as a hex string.

-----------------------------------------------------------------------------

stringToSignature
=================

.. code-block:: javascript

    utils.stringToSignature(signature)

Converts a hex string signature into an object with v,r,s Buffers.

----------
Parameters
----------

1. ``signature`` - ``string``: A signature string.

-------
Returns
-------

``Object``: A signature object with v,r,s as Buffers.

-----------------------------------------------------------------------------

getSequentialTxs
================

.. code-block:: javascript

    utils.getSequentialTxs(n, blockNum)

Generates sequential transactions in the same block.
Usually used for testing with mass amounts of transactions.

----------
Parameters
----------

1. ``n`` - ``number``: Number of transactions to generate.
2. ``blockNum`` - ``number``: Block in which the transactions will be included.

-------
Returns
-------

``Array<SignedTransaction>``: An array of SignedTransaction_ objects.

-----------------------------------------------------------------------------

getRandomTx
===========

.. code-block:: javascript

    utils.getRandomTx(blockNum, sender, recipient, numTransfers)

Generates a random transaction.
Usually used for testing.

----------
Parameters
----------

1. ``blockNum`` - ``number``: Block in which this transaction will be included.
2. ``sender`` - ``string``: Address of the sender.
3. ``recipient`` - ``string``: Address of the recipient.
4. ``numTransfers`` - ``number``: Number of transfers to generate.

-------
Returns
-------

``UnsignedTransaction``: An UnsignedTransaction_ object.


.. _SignedTransaction: models.html#signedtransaction
.. _UnsignedTransaction: models.html#unsignedtransaction
