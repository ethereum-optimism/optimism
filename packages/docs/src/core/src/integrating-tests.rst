===================
Integrating the OVM
===================
If you're interested in testing your contracts running on the OVM, then you're in the right place!  Please note that the OVM is in alpha and, as such, you'll probably find some bugs.  If you do, please drop us a line or open a github issue!

There are two steps you need to take to get your contracts running on the ovm: using ``@eth-optimism/solc-transpiler`` in place of ``solc`` so your contracts can be transpiled into OVM-compatible versions, and using ``@eth-optimism/rollup-full-node`` as your web3 provider.

Integrating the OVM Transpiler
==============================

Installing
-----------

Both ``truffle`` and ``ethereum-waffle`` allow you to specify a custom replacement for ``solc-js``.  First, you'll need to install ``@eth-optimism/solc-transpiler``:

.. code-block:: none

    yarn add @eth-optimism/solc-transpiler && yarn install

or

.. code-block:: none

    npm install --save @eth-optimism/solc-transpiler


``@eth-optimism/solc-transpiler`` Accepts the same compiler options as as ``solc``, with one additional option, ``executionManagerAddress``.  The Execution Manager is a smart contract implementing the OVM's containerization functionality.  If you are using an unmodified ``rollup-full-node``, the default execution manager address is ``0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA``.  More info on the execution manager can be found here. (link)

*With Waffle:*

To use the transpiler with ``ethereum-waffle``, set the ``solc.version`` configuration to ``""@eth-optimism/solc-transpiler"`` and ``compilerOptions.executionManagerAddress`` to ``"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"``.

example waffle-config.json:

.. code-block:: none

  {
    "solcVersion": "@eth-optimism/solc-transpiler",
    "compilerOptions": {
      "executionManagerAddress": "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"
    }
  }
  
*With Truffle:*

To use the transpiler with Truffle, set truffle's ``compilers.solc.version`` configuration to ``@eth-optimism/solc-transpiler``.

example truffle-config.json:

.. code-block:: none

  // Note, will need EXECUTION_MANAGER_ADDRESS environment variable set.
  // Default is "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"
  module.exports = {
    compilers: {
      solc: {
      // Add path to the solc-transpiler
      version: "@eth-optimism/solc-transpiler",
      }
    }
  }

Truffle does not expose compiler options, so the execution manager address must be passed in with an environment variable, called ``EXECUTION_MANAGER_ADDRESS``.  One easy way to do this is run tests with 

.. code-block:: none

    $   env EXECUTION_MANAGER_ADDRESS="0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA" [test command]


Currently, ``rollup-full-node`` breaks Truffle's ``gasLimit`` and ``blockGasLimit``.  To avoid this, you can set both to ``undefined`` where they are used.

Integrating the OVM Full Node
------------------------------

To use your transpiled contracts, you need to use ``@eth-optimism/rollup-full-node`` as your web3 provider.  To do this, make sure it's installed:

.. code-block:: none

    yarn add @eth-optimism/rollup-full-node && yarn install

or

.. code-block:: none

    npm install --save @eth-optimism/rollup-full-node


To get your provider and some wallets:

.. code-block:: none

    const RollupFullNode = require("@eth-optimism/rollup-full-node")
    const provider = RollupFullNode.getMockOVMProvider()
    const wallets = RollupFullNode.getWallets(provider)