=========
DBService
=========

``DBService`` handles all interaction with the user's local database.
Currently, **all** services talk to `ChainService`_ to interact with the database instead of talking with ``DBService`` directly.

------------------------------------------------------------------------------

Backends
========

``plasma-core`` uses `key-value store`_ when storing information in the database.
``plasma-core`` provides serveral different backends for ``DBService`` depending on the user's preference.
``DBService`` supports:

1. ``EphemDBProvider``, an in-memory database (mostly for testing).
2. ``LevelDBProvider``, a wrapper for LevelDB_.

------------------------------------------------------------------------------

get
===

.. code-block:: javascript

    db.get(key)

Returns the value stored at the given key.

----------
Parameters
----------

1. ``key`` - ``String``: The key to query.

-------
Returns
-------

``any``: The value stored at that key.

------------------------------------------------------------------------------

set
===

.. code-block:: javascript

    db.set(key, value)

Stores a value at the given key.

----------
Parameters
----------

1. ``key`` - ``String``: The key to set.
2. ``value`` - ``any``: The value to store.

------------------------------------------------------------------------------

delete
======

.. code-block:: javascript

    db.delete(key)

Deletes the value at a given key.

----------
Parameters
----------

1. ``key`` - ``String``: The key to delete.

------------------------------------------------------------------------------

exists
======

.. code-block:: javascript

    db.exists(key)

Checks if a given key is set.

----------
Parameters
----------

1. ``key`` - ``String``: The key to check.

-------
Returns
-------

``boolean``: ``true`` if the key exists, ``false`` otherwise.


.. _ChainService: services/chain.html
.. _LevelDB: http://leveldb.org/
.. _key-value store: https://en.wikipedia.org/wiki/Key-value_database
