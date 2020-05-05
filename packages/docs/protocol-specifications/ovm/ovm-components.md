# Architecture and Core Components of the OVM

The core functionality of the OVM is to run L2 transactions on L1 in such a way
that they are "pure" or "deterministic"--that is, no matter what time in
the future a dispute about them is triggered on layer 1, the output of
the computation is the same--\*no matter what the state of the L1.

To accomplish this, there are three critical components:

1. **Execution Manager:** provides a safe, sandboxed execution environment for OVM contracts to run in.
2. **Safety Checker:** contract which can check that OVM contracts will not try to escape the Execution Manager's sandbox.
3. **Transpiler:** converts EVM contracts to safe OVM contracts which will not escape the execution sandbox.

## Execution Manager

The Execution Manager serves as a state and execution sandbox for
executing OVM transactions and is where OVM transactions are always processed.  Instead of directy accessing state variables like contract storage, the block number, and so on, OVM contracts may only access them through the execution manager.  Thus, by configuring a particular state before running an OVM transaction through it, we are able to simulate how the transaction *was supposed to behave* given the L2 state. 

For example, in an optimistic rollup fraud proof, if
block `N+1` with transaction `T` was fraudulent, we are able to
configure the execution manager with the previous `stateN`
(`executionManager.setContext(stateN)`), call
`executionManager.executeTransaction(T)`, and compare the output to the
fraudulent `stateN+1`.

The execution manager interfaces with "code contracts," which are EVM
contracts that can only use the OVM's container interface.  For instance, in L1, the EVM opcode `SSTORE(...)` is used  during an ERC20 transfer to update the sender and recipients' balance.  In the OVM, the ERC20 code contract instead calls
`executionManager.ovmSSTORE(...)` to update the OVM's virtualized state
instead.

![The Execution Manager](https://i.imgur.com/9eMuXwc.png)


The execution manager contract can be found [here](https://github.com/ethereum-optimism/optimism-monorepo/blob/master/packages/ovm/src/contracts/ExecutionManager.sol).

## Safety Checker

To ensure that the execution of an OVM transaction is deterministic for a given sandbox state, we must enforce that **only** the container interface
described above is used. To accomplish this, we have a "purity checker."
The purity checker analyzes the low-level assembly bytecode of an EVM
contract to tell the execution manager whether the code conforms to the
OVM interface. If it does not, then the execution manager does not allow
such a contract to be created or used in a fraud proof.

![The Safety Checker](https://i.imgur.com/JYKNqNC.png)

The safety checker contract can be found [here](https://github.com/ethereum-optimism/optimism-monorepo/blob/master/packages/ovm/src/contracts/SafetyChecker.sol)

## Transpiler

Because smart contracts are not normally compiled to comply with any container interface, we have a transpiler which takes low-level EVM assembly, detects the usage of any stateful opcodes, and converts them into calls to the relevant Execution Manager method. There's a lot more that goes into doing that, which you can read about in the following section.

Code for the transpiler can be found [here](https://github.com/ethereum-optimism/optimism-monorepo/tree/master/packages/rollup-dev-tools/src/tools/transpiler).