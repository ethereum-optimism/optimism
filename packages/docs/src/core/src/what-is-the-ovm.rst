===================
The OVM 101
===================

What is the OVM?
================

The Optimistic Virtual Machine (OVM) is a scalable form of the EVM designed for use in layer 2 (L2) systems.  You can think of it as a container environment like Docker or Xen: even though a containerized program on Docker runs on your local computer, it executes *as if on the machine defined by its container.* Similarly, OVM transactions which occurred on an L2 chain can be executed on an L1 chain *as if they are running on the L2 chain*.  This ability is the fundamental underpinning of `optimistic execution`_ , the `basis`_ for L2 scaling solutions: the L1 blockchain only needs to evaluate transactions in the pessimistic case of misbehavior.

The OVM functions as a drop-in replacement for the EVM.  This means that, for the first time, L2 chain can provide application developers with the same experience and features that L1 Ethereum has.  No hoops, no tricks--the Ethereum you know and love, ready to scale up with L2.

To learn more about how it works under the hood, you can check out our specifications section.

How do I use the OVM?
=====================

Because the OVM complies with the EVM, porting your system to use the OVM is intended to be as easy as possible.  Only two things have to be done: 1. Transpile your contracts to comply with the OVM's containerization interface.  To accomplish this, we provide ``@eth-optimism/solc-transpiler``: a plug-and-play replacement for the solidity compiler, ``solc``, which transpiles your smart contracts to work on the OVM.
2. Run the transpiled contracts in an OVM-enabled web3 provider.  To accomplish this, we provide ``@eth-optimism/rollup-full-node``: a plug-and-play Web3 provider which converts web3 calls to work on the OVM.

Check out our ERC20 `tutorial`_ if you'd like to try it in action!  Or learn to integrate it into an existing project `here`_.

.. _`basis`: https://twitter.com/ben_chain/status/1161425776929136641?s=20
.. _`optimistic execution`: https://plasma.group/optimistic-game-semantics.pdf
.. _`here`: ./integrating-tests.html
.. _`tutorial`: https://github.com/ethereum-optimism/erc20-example
