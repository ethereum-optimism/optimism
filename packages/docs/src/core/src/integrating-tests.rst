===================
Integrating the OVM
===================
If you're interested in testing your contracts running on the OVM, then you're in the right place!  Please note that the OVM is in alpha and, as such, you'll probably find some bugs.  If you do, please drop us a line or open a github issue!

There are two steps you need to take to get your contracts running on the ovm: using ``@eth-optimism/solc-transpiler`` in place of ``solc`` so your contracts can be transpiled into OVM-compatible versions, and using ``@eth-optimism/rollup-full-node`` as your web3 provider.

For reference, example integrations for both Truffle and Waffle can be found `in our monorepo`_ .

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

Using With Waffle
-----------------

To use the transpiler with ``ethereum-waffle``, set the ``solc.version`` configuration to ``""@eth-optimism/solc-transpiler"`` and ``compilerOptions.executionManagerAddress`` to ``"0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"``.

example waffle-config.json:

.. code-block:: none

  {
    "solcVersion": "@eth-optimism/solc-transpiler",
    "compilerOptions": {
      "executionManagerAddress": "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"
    }
  }
  
Using With Truffle
------------------

To use the transpiler with Truffle, set truffle's ``compilers.solc.version`` configuration to ``@eth-optimism/solc-transpiler``, and configure the ``EXECUTION_MANAGER_ADDRESS`` environment variable. 

Currently, Truffle does not provide a clean way to use custom chain IDs, so we have created the ``@eth-optimism/ovm-truffle-provider-wrapper`` library to seamlessly wrap your provider of choice to handle this.

example truffle-config.json:

.. code-block:: json

  const HDWalletProvider = require('truffle-hdwallet-provider');
  const ProviderWrapper = require("@eth-optimism/ovm-truffle-provider-wrapper");
  const mnemonic = 'candy maple cake sugar pudding cream honey rich smooth crumble sweet treat';

  // Set this to the desired Execution Manager Address -- required for the transpiler
  process.env.EXECUTION_MANAGER_ADDRESS = process.env.EXECUTION_MANAGER_ADDRESS || "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"

  module.exports = {
  /**
   * Note: this runs the OVM full node for the duration of the tests at `http://127.0.0.1:8545`
   *
   * To run tests:
   * $ truffle test ./truffle-tests/test-erc20.js --config truffle-config-ovm.js
   */
  networks: {
    test: {
      network_id: 108,
      provider: function() {
        return ProviderWrapper.wrapProviderAndStartLocalNode(new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10));
      },
      gasPrice: 0,
      gas: 9000000,
    },
    live_example: {
      provider: function () {
        return ProviderWrapper.wrapProvider(new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10));
      },
      gasPrice: 0,
      gas: 9000000,
    },
  },

  // Set default mocha options here, use special reporters etc.
  mocha: {
    timeout: 100000
  },

  compilers: {
    solc: {
      // Add path to the solc-transpiler
      version: "@eth-optimism/solc-transpiler",
    }
  }
}

As you can see in the above comments, you must spin up the rollup full node before running truffle tests.  To do this, with ``@eth-optimism/rollup-full-node`` installed, you can run:

.. code-block:: bash

  node rollup-full-node/build/src/exec/fullnode.js

Currently, ``rollup-full-node`` breaks Truffle's ``gasLimit`` and ``blockGasLimit``.  To avoid this, you can set both to ``undefined`` where they are used.

Integrating the OVM Full Node
==============================

To use your transpiled contracts, you need to use ``@eth-optimism/rollup-full-node`` as your web3 provider.  To do this, make sure it's installed:

.. code-block:: none

    yarn add @eth-optimism/rollup-full-node && yarn install

or

.. code-block:: none

    npm install --save @eth-optimism/rollup-full-node

To get your provider and some wallets:

.. code-block:: javascript

    const RollupFullNode = require("@eth-optimism/rollup-full-node")
    const provider = RollupFullNode.getMockProvider()
    const wallets = RollupFullNode.getWallets(provider)

.. _`in our monorepo`: https://github.com/ethereum-optimism/optimism-monorepo/tree/master/packages/examples