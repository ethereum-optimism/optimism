==============
JSONRPCService
==============

``JSONRPCService`` handles incoming JSON-RPC method calls.
A full list of methods is documented at the `JSON-RPC Methods Specification`_.
Note that ``JSONRPCService`` does **not** expose any form of external interface (such as an HTTP server).
Full nodes should implement these services so that users can interact with the node.
For more information about these external services, see our document on `extending plasma-core`_.

------------------------------------------------------------------------------

getAllMethods
=============

.. code-block:: javascript

    jsonrpc.getAllMethods()

Returns all available RPC methods.

-------
Returns
-------

``Object``: All subdispatcher methods as a single object in the form ``{ name: methodref }``.

------------------------------------------------------------------------------

getMethod
=========

.. code-block:: javascript

    jsonrpc.getMethod(name)

Returns a method by its name.

----------
Parameters
----------

1. ``name`` - ``string``: Name of the method.

-------
Returns
-------

``Function``: A reference to the method with that name.

------------------------------------------------------------------------------

handle
======

.. code-block:: javascript

    jsonrpc.handle(method, params = [])

Calls a method with the given name and params.

----------
Parameters
----------

1. ``method`` - ``string``: Name of the method to call.
2. ``params`` - ``Array``: An array of parameters.

-------
Returns
-------

``Promise<any>``: The result of the method call.

------------------------------------------------------------------------------

handleRawRequest
================

.. code-block:: javascript

    jsonrpc.handleRawRequest(request)

Handles a raw `JSON-RPC request`_.

----------
Parameters
----------

1. ``request`` - ``Object``: A JSON-RPC `request object`_.

-------
Returns
-------

``Promise<Object>``: A JSON-RPC `response object`_.


.. _JSON-RPC Methods Specification: specs/jsonrpc.html
.. _extending plasma-core: extending-plasma-core.html
.. _JSON-RPC request: https://www.jsonrpc.org/specification#request_object
.. _request object: https://www.jsonrpc.org/specification#request_object
.. _response object: https://www.jsonrpc.org/specification#response_object
