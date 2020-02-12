==========================
Notes on Design Philosophy
==========================


# Overview

This diagram depicts a simple developer enviornment interacting with a Rollup node using the MVOVM.

![high-level-ovm (2) (1)](https://user-images.githubusercontent.com/706123/70545643-e78cb480-1b3b-11ea-8562-59e7d3e23b0b.png)
( Editable Version -- https://drive.google.com/open?id=1iF2gvJut3LU1NCfcJLn7Jh_Cm0PZIwTZ )

### Components
- Local Solidity test suite
  - Imports transpiler
- Rollup Fullnode
  - Contains the OVM
  - MVOVM uses an Ethereum node on the backend
    - (Stateful) [Execution Manager](https://github.com/op-optimism/optimistic-rollup/wiki/The-Execution-Manager)
- Ethereum fraud contracts
  - Stateless Execution Manager

# Stateful (off-chain) vs Stateless (on-chain) State Manager
There are two settings in which we will be executing transactions against our VM:

1. Off-chain to calculate the current state of the rollup chain; and
2. On-chain to prove a fraudulent [`state root`](https://github.com/plasma-group/optimistic-rollup/wiki/Glossary).

Both cases are identical except for one key detail: _off-chain we have access to the full state, while on-chain we only have access to the state we need to compute the result of the transaction_. This difference means that state access must be handled slightly differently between the two implementations; however, we should keep the two implementations as similar as possible to reduce the risk of bugs.

### Stateless Clients
This design comes from the work on stateless clients introduced by Vitalik: https://ethresear.ch/t/the-stateless-client-concept/172

Stateless clients evalute state transitions with only a subset of the full state. Every storage slot & contract code which is touched during the execution of the smart contract must be stored locally to evaluate the transition. If all touched state is stored, a stateless client can evaluate the validity of a transition as well as calculate the resulting state root.

The stateless client allows us to verify a single state transition in isolation--exactly what is required for a fraud proof. Fraud proofs in ORU cannot hold all state because because then we lose the ORU scalability in the case of fraud. Instead of holding all the state, we can use a stateless client!

### Stateful (off-chain) State Manager
Off-chain there is no problem running our OVM with all of the ORU state. This behaves exactly like an Ethereum fullnode.

... TODO ...

### Stateless (on-chain) State Manager

... TODO ...

# L2_CONTEXT (aka global variables to transpile)
Can we add fields to "msg"? e.g. have msg.queueOrigin?

A: Don't think so. looks like `msg.value` has its own assembly code of `callvalue` for example.

Block and Transaction Properties ([Source](https://solidity.readthedocs.io/en/v0.4.24/units-and-global-variables.html))
--------------------------------

- ``blockhash(uint blockNumber) returns (bytes32)``: hash of the given block - only works for 256 most recent, excluding current, blocks
- ``block.coinbase`` (``address payable``): current block miner's address
- ``block.difficulty`` (``uint``): current block difficulty
- ``block.gaslimit`` (``uint``): current block gaslimit
- ``block.number`` (``uint``): current block number
- ``block.timestamp`` (``uint``): current block timestamp as seconds since unix epoch
- ``gasleft() returns (uint256)``: remaining gas
- ``msg.data`` (``bytes calldata``): complete calldata
- ``msg.sender`` (``address payable``): sender of the message (current call)
- ``msg.sig`` (``bytes4``): first four bytes of the calldata (i.e. function identifier)
- ``msg.value`` (``uint``): number of wei sent with the message
   - turns into assembly: `callvalue`([source](https://ethereum.stackexchange.com/a/47476))
- ``now`` (``uint``): current block timestamp (alias for ``block.timestamp``)
- ``tx.gasprice`` (``uint``): gas price of the transaction
- ``tx.origin`` (``address payable``): sender of the transaction (full call chain)

>  note:
>     The values of all members of ``msg``, including ``msg.sender`` and
>     ``msg.value`` can change for every **external** function call.
>     This includes calls to library functions.

> note:
>     Do not rely on ``block.timestamp``, ``now`` and ``blockhash`` as a source of randomness,
>     unless you know what you are doing.

> note:
>     The block hashes are not available for all blocks for scalability reasons.
>     You can only access the hashes of the most recent 256 blocks, all other
>     values will be zero.

>  note:
>     The function ``blockhash`` was previously known as ``block.blockhash``, which was deprecated in
>     version 0.4.22 and removed in version 0.5.0.

>  note::
>     The function ``gasleft`` was previously known as ``msg.gas``, which was deprecated in
>     version 0.4.21 and removed in version 0.5.0.
 
> index: abi, encoding, packed




# Other things to be transpiled:

Members of Address Types ([Source](https://solidity.readthedocs.io/en/v0.4.24/units-and-global-variables.html))
------------------------

- ``<address>.balance`` (``uint256``):
    balance of the :ref:`address` in Wei
- ``<address payable>.transfer(uint256 amount)``:
    send given amount of Wei to :ref:`address`, reverts on failure, forwards 2300 gas stipend, not adjustable
- ``<address payable>.send(uint256 amount) returns (bool)``:
    send given amount of Wei to :ref:`address`, returns ``false`` on failure, forwards 2300 gas stipend, not adjustable
- ``<address>.call(bytes memory) returns (bool, bytes memory)``:
    issue low-level ``CALL`` with the given payload, returns success condition and return data, forwards all available gas, adjustable
- ``<address>.delegatecall(bytes memory) returns (bool, bytes memory)``:
    issue low-level ``DELEGATECALL`` with the given payload, returns success condition and return data, forwards all available gas, adjustable
- ``<address>.staticcall(bytes memory) returns (bool, bytes memory)``:
    issue low-level ``STATICCALL`` with the given payload, returns success condition and return data, forwards all available gas, adjustable

> warning:
>     There are some dangers in using ``send``: The transfer fails if the call stack depth is at 1024
>     (this can always be forced by the caller) and it also fails if the recipient runs out of gas. So in order
>     to make safe Ether transfers, always check the return value of ``send``, use ``transfer`` or even better:
>     Use a pattern where the recipient withdraws the money.

>  note:
>    Prior to version 0.5.0, Solidity allowed address members to be accessed by a contract instance, for example ``this.balance``.
>    This is now forbidden and an explicit conversion to address must be done: ``address(this).balance``.
NOTE: we will need address(this) to return the L2 address and not the L1 address.

Contract Related
----------------

- ``this`` (current contract's type):
    the current contract, explicitly convertible to :ref:`address`

- ``selfdestruct(address payable recipient)``:
    Destroy the current contract, sending its funds to the given :ref:`address`
    and end execution.
    Note that ``selfdestruct`` has some peculiarities inherited from the EVM:
    - the receiving contract's receive function is not executed.
    - the contract is only really destroyed at the end of the transaction and ``revert`` s might "undo" the destruction.