# Execution Manager Overview

The Execution Manager is technically just a smart contract running in a local EVM \(layer 2\) and available on Ethereum to evaluate fraud claims \(layer 1\), but in principle, it is much more. It _is_ the layer 2 EVM, and it allows our Optimistic Rollup implementation to generically support layer 1 smart contracts.

## Motivation

The [Unipig Demo](https://unipig.exchange/) showed that Optimistic Rollup is possible with custom contract code in both layer 1 and layer 2. Layer 1 contracts each need a custom state transition function that can be given a snapshot of the layer 2 state and a state transition to execute in order to evaluate if the layer 2 state transition was properly executed. A simple state transition function example would be transferring an ERC-20 token. The layer 1 token contract would need a function that takes in pre-state \(i.e. address balances, approvals, etc.\), evaluates a particular transition \(e.g. a transfer\), and computes the resulting state \(i.e. updated balances\). Needless to say, the logic to execute this state transition in layer 2 needed to be created as well.

### To support generic smart contracts in layer 1...

We need all state transitions for all possible contracts deployed to layer 2 to be generically calculable by layer 1. The EVM provides this functionality, but layer 1 runs on the EVM -- we need this to run on layer 1 \(_on the EVM_\). If we can create an EVM that can run _inside_ of the EVM, all standard EVM operations can be executed efficiently in this layer 2 EVM while also being generically verifiable in the case of fraud in layer 1 \(by calling the EVM within the layer 1 EVM\).

### To support generic smart contracts in layer 2...

The layer 2 EVM needs to be able to run all layer 1 smart contracts with no additional code created per contract. If layer 2 runs a layer-1-compatible EVM \(that, itself, can be run in layer 1\), this is achieved.

We call this layer-1-compatible EVM that can run within the layer 1 EVM the OVM ExecutionManager Contract.

## Design

### Necessary Features

Just like the EVM, the OVM ExecutionManager Contract must: 

* Handle all opcodes other than those deeply embedded in the layer 1 protocol (like COINBASE, DIFFICULTY, block NUMBER, BLOCKHASH\) 
* Generically support smart contracts, including those that depend on and even create other smart contracts 
* Serve as the entrypoint to all calls, transactions, and state modification 
* Store all state created and modified by transaction and smart contract execution

### Implementation

As stated above, the Execution Manager is a smart contract that runs in a \[slightly modified\] local EVM\*. Below are its implementation details and reasoning behind its functionality.

### Transaction Context & Re-implementing Opcodes

Transactions that are run in layer 2 will necessarily have a different context than fraud proofs in layer 1. For instance, a fraud proof can only be submitted to layer 1 to dispute a layer 2 transaction some time after it has been executed. As such, opcodes like TIMESTAMP will _function_ the same \(it'll be the timestamp when the transaction was actually executed\), but the actual current time will not _be_ the same when executed in layer 1 vs challenged as fraudulent layer 2.

![The Execution Manager](https://i.imgur.com/cOhmFRo.png)

To handle this, the OVM ExecutionManager Contract implements these opcodes as functions. When a contract executing in layer 2 or a fraud proof executing in layer 1 needs to know the timestamp, it will call the OVM ExecutionManager Contract instead of accessing these layer-1-protocol-level opcodes directly. We have [transpilation tools](https://github.com/ethereum-optimism/optimism-monorepo/tree/37044e22125ed779c51d83d7491dc19fcd7bd1cf/packages/docs/protocol-specifications/ovm/protocol-specifications/ovm/transpiler.md) that take compiled layer 1 bytecode and swap out certain opcodes, like TIMESTAMP, for calls to our OVM ExecutionManager Contract. All contracts deployed to layer 2 must be transpiled accordingly.

In our example, the sequencer that commits to a layer 2 transaction passes the timestamp at the time of execution to the OVM ExecutionManager Contract with the transaction to evaluate. They also specify the same timestamp in their rollup block that includes the transaction. This way, when the fraud proof is executed, the same timestamp from the rollup block will be set in the OVM ExecutionManager Contract prior to evaluating fraud so that the context that was committed to can be accessed correctly.

### CALL

There are many contextual differences between layer 1 and layer 2, so we won't go through all of them, but another important one to consider is that the addresses of the corresponding contracts will be different. All contracts in layer 2 are actually deployed by the OVM ExecutionManager Contract, but their addresses are created as if the caller deployed them. As such, all contract deployments go through the OVM ExecutionManager Contract, which maintains a map from OVM contract addresses \(as if the caller created them\) to EVM contract addresses \(the address of the contract that was actually deployed by the OVM ExecutionManager Contract\). This means that all CALL type opcodes must be transpiled to instead call the OVM ExecutionManager Contract so it may fill in the proper address for the call, as well as set other relevant context for the call's execution, like CALLER and ADDRESS.

### SSTORE & SLOAD

The last example to highlight is that SSTORE and SLOAD also need to be transpiled into calls to the OVM ExecutionManager Contract. Recall that one of the requirements is that the OVM ExecutionManager Contract needs to store all layer 2 state. This is so rollup blocks can commit to single pre-state and post-state roots and the fraud proof's pre- and post-state can be verified and executed through the OVM ExecutionManager Contract on layer 1 during fraud proofs.

A list of transpiled opcodes and other transpilation details are available [here](./transpiler.md).

## Example: A user trading ETH for BAT on Uniswap

There are two main parts to this example:

* User transaction calling the layer 2 Uniswap Contract
* The layer 2 Uniswap Contract calling the corresponding ERC-20 Contracts to update balances

### Steps

### Sequencer Handles Request

1. It receives a signed transaction from the User calling the Uniswap BAT Exchange address's `ethToTokenTransferInput(...)` function.
2. It wraps this transaction's calldata in a call to the OVM ExecutionManager Contract's `executeCall(...)` function and sends the wrapped transaction.

### OVM ExecutionManager Contract handles the transaction in executeCall\(...\)

1. It receives the wrapped transaction, sets the transaction context (including timestamp, etc.), and calls the `ovmCALL(...)` opcode replacement function to execute the transaction.
2. Its `ovmCALL(...)` function sets the call-specific context (including the CALLER, the ADDRESS of the uniswap contract, etc.)
3. It looks up the EVM address of the Uniswap contract from the OVM address and CALLs the contract with the original transaction data.

### Uniswap / BAT Contract interaction

1. Uniswap determines the exchange rate based on how much BAT it has by calling the OVM ExecutionManager Contract's `ovmCALL(...)` function to call the layer 2 BAT ERC-20 contract's `balanceOf(...)` function.
2. The OVM ExecutionManager Contract temporarily updates all of the call context variables in `ovmCALL(...)` to properly reflect that the CALLER is the Uniswap contract, ADDRESS is the BAT address, etc.
3. The OVM ExecutionManager Contract calls the BAT contract and it properly returns the balance
4. The OVM ExecutionManager Contract restores the call context such that the CALLER is the original caller, the ADDRESS is the Uniswap contract, etc.
5. The OVM ExecutionManager Contract returns the result to the Uniswap contract.
6. The Uniswap contract then calls the BAT contract, through the OVM ExecutionManager Contract again, to actually execute the transfer of the calculated amount of BAT
7. The Uniswap contract makes a final call to the BAT contract, through the OVM ExecutionManager Contract, to transfer the WETH \(all ETH in the OVM is WETH\)
8. The Uniswap returns the number of tokens bought.
9. The OVM ExecutionManager Contract restores the original call context before the original call to the Uniswap contract and returns the result.

### OVM ExecutionManager Contract handles the transaction in `executeCall(...)` \(continued\)

1. It restores the original transaction context from before the transaction and returns the result

### Sequencer Handles Request \(continued\)

1. It gets the internal transaction hash as a result.
2. It stores a mapping from the original transaction hash to the internal transaction hash for future transaction lookup.
3. It returns the original transaction hash, in compliance with Web3, to the caller.

### Not mentioned above:

* Access of TIMESTAMP, ADDRESS, CALLER, etc. which are actually CALLs to the associated OVM ExecutionManager Contract function. 
* Access of all storage, which is actually a CALL to the `ovmSLOAD(...)` OVM ExecutionManager Contract function.
* Storage modification, which is actually a CALL to the `ovmSSTORE(...)` OVM ExecutionManager Contract function. 
* All other opcodes handled through the OVM ExecutionManager Contract.
* The layer 2 EVM will be run by the Sequencer that submits new layer 2 "blocks" to layer 1, validators who validate these blocks once submitted to layer 1, and any other interested party. Validation entails executing each individual state transition that is claimed to be valid by the Sequencer and ensuring that it is, in fact, valid (i.e. the resulting state from executing the state transition match the post-state claimed by the Sequencer).

