=====================
Extending plasma-core
=====================
**Note:** This article is for developers who want to build their own full plasma nodes using ``plasma-core``.

------------------------------------------------------------------------------

As explained in `What is plasma-core?`_ ``plasma-core`` is just the *core* of our plasma node ecosystem.
It's not really a *full* plasma client.
In order to build out a full client that exposes all of the functionality a user will expect, you'll need to *extend* ``plasma-core``.

To this end, ``plasma-core`` is designed to be maximally extensible.
It tries to make as few decisions on your behalf as possible.
This means you can make a lot of different clients out of ``plasma-core``!
So far we've tested out creating a full `Node.js` node and a full node inside of a `Chrome extension`_!

What's missing
=============
``plasma-core`` is missing a few key features that a user might expect.
Here's a list of what's missing so that you know what to expect if you want to build a full plasma node.

Front-facing Services
---------------------
``plasma-core`` doesn't come with any sort of front-facing services (i.e. an HTTP server that handles JSON-RPC requests).
If you want users to be able to interact with your node software, you'll need to implement a service like this.
We left this to node developers because different types of nodes might handle this in completely different ways.
For example, the node that we're building inside of a `Chrome extension`_ talks to apps via Chrome's `native message passing interface`_ instead of over HTTP.
Front-facing services need to wrap and pipe calls into JSONRPCService_.

User interface
--------------
Because ``plasma-core`` doesn't provide any front-facing services, it also doesn't provide any sort of user interface.
As a node developer, you'll probably want to create some sort of simple interface that allows users to interact with the node.
This might take the form of a CLI that sends requests to an HTTP server or a local website that connects to the node.

Wallet Management
-----------------
Private key storage and transaction signing is not handled by ``plasma-core``.
``plasma-core`` only provides a mock wallet for testing that should **not** be used in production.
You will therefore have to implement your own WalletService_.

However, key management is hard and you probably shouldn't be building your own wallets.
We therefore recommend deferring this functionality to a user's Ethereum node.
This can be as easy as forwarding the necessary API calls (as described on the WalletService_ documentation page) to the Ethereum node.
``plasma-extension`` uses this method by forwarding all wallet-related activity to MetaMask_.

.. _What is plasma-core: what-is-plasma-core.html
.. _Chrome extension: https://plasma-extension.readthedocs.io/en/latest/
.. _native message passing interface: https://developer.chrome.com/apps/messaging
.. _JSONRPCService: services/jsonrpc.html
.. _WalletService: services/wallet.html
.. _MetaMask: https://metamask.io/
