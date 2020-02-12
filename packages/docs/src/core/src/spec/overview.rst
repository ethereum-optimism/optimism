=============================
OVM Overview and Architecture
=============================

The core functionality of the OVM is to run transactions in such a way that they are "pure" or "deterministic"--that is, no matter what time in the future a dispute about them is triggered on layer 1, the output of the computation is the same--*no matter what the state of the L1.

To accomplish this, there are two critical smart contracts: the Execution Manager, and the Purity Checker.

Execution Manager
-----------------

The execution manager serves as a state and execution container for exeecuting OVM transactions.  To run an OVM transaction, it is sent as calldata an execution manager (running either on or off-chain).  Before passing this transaction to the execution manager, you can set all properties of the OVM state: contract storage, what contracts exist, the timestamp, and so on.  By configuring the execution manager to a particular state before running an OVM transaction through it, we are able to simulate how the transaction *was supposed to behave* given that configured state.  For example, in an optimistic rollup fraud proof, if block ``N+1`` with transaction ``T`` was fraudulent, we are able to configure the execution manager with the previous ``stateN`` (``executionManager.setContext(stateN)``), call ``executionManager.executeTransaction(T)``, and compare the output to the fraudulent ``stateN+1``.

The execution manager interfaces with "code contracts," which are contracts compatible with the OVM's container interface.  These contracts are EVM contracts which might be on L1 or L2.  So instead of usnig their built-in opcodes for state, they call the execution manager to do it to the OVM state.  For instance, in L1, the EVM opcode ``SSTORE`` is used during an ERC20 transfer to update the sender and recipients' balance.  In the OVM, the ERC20 code contract instead calls ``executionManager.ovmSSTORE`` to update the OVM's virtualized state instead.

.. raw:: html

   <img src="../../_static/images/execution-manager.png" alt="The Execution Manager">


Purity Checker
--------------

To ensure that the execution of an OVM transaction is deterministic between L1 and L2, we must enforce that **only** the container interface described above is used.  To accomplish this, we have a "purity checker."  The purity checker analyzes the low-level assembly bytecode of an EVM contract to tell the execution manager whether the code conforms to the OVM interface.  If it does not, then the execution manager does not allow such a contract to be created or used in a fraud proof.

.. raw:: html

   <img src="../../_static/images/purity-checker.png" alt="The Execution Manager">


Transpiler
----------

Because smart contracts are not normally compiled to comply with any containerization interface, we have a transpiler which takes low-level EVM assembly, detects the usage of any stateful opcodes, and converts them into calls to the relevant Execution Manager method.  There's a lot more that goes into doing that, which you can read about below.