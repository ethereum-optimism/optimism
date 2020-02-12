==============================
Transpiler-``solc`` Wrapper
==============================

Our transpilation binaries and OVM Different tooling need to be seamlessly included in various standard Ethereum tools, including test tools. This outlines how this may happen for testing tools.

General Approach
=================

1. Hook into calls to the configured compiler
2. Call the configured compiler
3. Locate complier output or hook into post-compilation / pre-output step
4. Transpile the compiled bytecode
5. Output to the configured location

Compiler Integration
====================

Truffle
--------
As of version 5.0, Truffle supports [_Bring Your Own Compiler_](https://github.com/trufflesuite/truffle/releases/tag/v5.0.0-beta.0#bring-your-own-compiler) capabilities, allowing us to configure Truffle with a custom version of the [``solc-js``](https://github.com/ethereum/solc-js) package that should be used in compilation. 

Caveats:
--------

* Truffle assumes that this is the path to a version of the ``solc-js`` package
  * We will need to fork ``solc-js`` and make sure our executable has the same backward-compatible interface as the public-facing ``solc-js`` API
  * Our library, in turn, needs to make the compiler version (just ``solc-js``?) configurable to achieve functional parity with Truffle

Approach:
-----------

1. Fork ``solc-js`` and import transpiler
2. In compilation API functions, change the configured output location to a temp dir
3. Call the configured compiler on all input
5. Transpile the compiled bytecode from the output files in the temp dir
6. Output to the originally configured location


Waffle
=======
Waffle allows the configured ``solc-version`` to be a path to a local [``solc-js``](https://github.com/ethereum/solc-js) package, allowing us to configure Waffle with a custom version of the ``solc-js`` package that should be used in compilation.

Caveats:
----------

* Waffle assumes that this is the path to a version of the ``solc-js`` package
  * We will need to fork ``solc-js`` and make sure our executable has the same backward-compatible interface as the public-facing ``solc-js`` API
  * Our library, in turn, needs to make the compiler version (just ``solc-js``?) configurable to achieve functional parity with Waffle

Approach:
-----------

1. Fork ``solc-js`` and import transpiler
2. In compilation API functions, change the configured output location to a temp dir
3. Call the configured compiler on all input
5. Transpile the compiled bytecode from the output files in the temp dir
6. Output to the originally configured location

On-The-Fly
===========

In theory we could compile bytecode on-the-fly in our [Web3-compatible RPC Server](https://github.com/op-optimism/optimistic-rollup/blob/master/packages/rollup-full-node/src/app/fullnode-rpc-server.ts). 

This would allow test tools to function entirely as they otherwise would, _but it would make all compilation errors deployment errors_. This would not only be the case in testing, but also in live use. This may be fine, though.

Test Integration
==================

Our [Web3-compatible RPC Server](https://github.com/op-optimism/optimistic-rollup/blob/master/packages/rollup-full-node/src/app/fullnode-rpc-server.ts) can be used to handle all Web3 requests that should be sent to the OVM, handling test execution once the transpiled bytecode is created.