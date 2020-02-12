===============
Troubleshooting
===============

The OVM is an alpha.  In trying it out, you're helping us scale Ethereum and pave the way for a new generation of Layer 2 systems.  It also means you'll probably run into some bugs!  If you do, please reach out to us and create an issue on our `monorepo`_. https://github.com/ethereum-optimism/optimism-monorepo

While you do so, here is a collection of tips and notes that may help you figure out what's going on.

Limitations
-----------
Some features of the Ethereum are not yet implemented, or just don't make sense to have, in the OVM.  Check out our limitations section (link) to get more information to check if this is why you're running into issues.

Logging
-------
We use the npm package``debug`` for logging.  To get a better sense of what might be breaking, you can run
.. code-block:: none

  env DEBUG="debug:*,error:*" [test command]

in your terminal.

Getting Wallets
---------------

``rollup-full-node`` provides an RPC-based provider, and does not always allow you to `getWallets()`.  Instead, use the `getWallets()` function exported by ``rollup-full-node`` instead.

_`monorepo`: https://github.com/ethereum-optimism/optimism-monorepo