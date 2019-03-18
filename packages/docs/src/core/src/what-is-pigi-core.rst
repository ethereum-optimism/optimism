===================
What is @pigi/core?
===================
``@pigi/core`` contains core modules that most of our other plasma chain clients use.
It handles the functionality that's relatively universal to all other clients.
For example, ``@pigi/core`` handles things like watching Ethereum and handling new transactions.
A full list of modules provided by ``@pigi/core`` is documented in our architecture_ page.

``@pigi/core`` is **not** a full plasma node.
It doesn't have key features - for example, it doesn't have a way for users to actually communicate with the app!
If you're looking for a full plasma node with all of those user-friendly features check out `@pigi/client`_.
``@pigi/client`` uses ``@pigi/core`` under the hood.

.. _`@pigi/client`: https://github.com/plasma-group/pigi/tree/master/packages/client
