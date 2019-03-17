=======
Logging
=======

``plasma-utils.logging`` exposes a few logging helpers.
These helpers make it possible to log messages in different contexts, e.g. in development or production.

API
===
ConsoleLogger
-------------

.. code-block: javascript

    new ConsoleLogger()

Creates a new `ConsoleLogger`, which simply wraps `console.log`.

----------
Parameters
----------

N/A

-------
Returns
-------

``ConsoleLogger``: The ``ConsoleLogger`` instance.

-----------------------------------------------------------------------------

~~~~~~~~~~~
= Methods =
~~~~~~~~~~~

-----------------------------------------------------------------------------

.. code-block: javascript

    consoleLogger.log(message)

Logs a new message to the console.

----------
Parameters
----------

1. ``message`` - ``String``: The message to be logged.

-----------------------------------------------------------------------------

DebugLogger
-----------

.. code-block: javascript

    new DebugLogger()

Creates a new `DebugLogger`, which wraps the debug_ NPM library.
The `DebugLogger` generally has better formatting than the `ConsoleLogger`.

----------
Parameters
----------

N/A

-------
Returns
-------

``DebugLogger``: The ``DebugLogger`` instance.

-----------------------------------------------------------------------------

~~~~~~~~~~~
= Methods =
~~~~~~~~~~~

-----------------------------------------------------------------------------

.. code-block: javascript

    debugLogger.log(message)

Logs a new message to the console, with extra formatting.

----------
Parameters
----------

1. ``message`` - ``String``: The message to be logged.

-----------------------------------------------------------------------------

.. _debug: https://www.npmjs.com/package/debug
