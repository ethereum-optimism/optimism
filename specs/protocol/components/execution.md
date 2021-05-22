# Execution

The Optimistic Virtual Machine (OVM) provides a sandboxed execution environment built on the EVM, with the goal of guaranteeing deterministic execution which maintains agreement between L1 and L2.

## Security properties and invariants

- Deterministic execution; maintaining consensus of rollup state between L1 and L2.

  - Corollary: L1 execution context MUST not be accessible (except in cases where the context can be guaranteed to agree with L2, ie. `GAS` is allowed, but must agree at all time during execution.
  - Unsafe opcodes must not be deployable.
  - It must be possible to complete the execution of a fraud proof within the L1 block gas limit.

- The execution context should be ephemeral, and not persist between calls to `run()`
  - More precisely: although the Execution Manager does hold some permanent values in storage, those values should remain constant before and after each execution. The state root of the contract should be constant.

## Safety Checking

In order to maintain the property of Deterministic Execution, we consider the following opcodes unsafe, and MUST prevent them from being deployed on L2.

All currently unassigned opcodes which are not yet assigned in the EVM are also disallowed.

**Unsafe Opcodes**

- `ADDRESS`
- `BALANCE`
- `ORIGIN`
- `EXTCODESIZE`
- `EXTCODECOPY`
- `EXTCODEHASH`
- `BLOCKHASH`
- `COINBASE`
- `TIMESTAMP`
- `NUMBER`
- `DIFFICULTY`
- `GASLIMIT`
- `GASPRICE`
- `CREATE`
- `CREATE2`
- `CALLCODE`
- `DELEGATECALL`
- `STATICCALL`
- `SELFDESTRUCT`
- `SELFBALANCE`
- `SSTORE`
- `SLOAD`
- `CHAINID`
- `CALLER`\*
- `CALL`\*
- `REVERT`\*

\* The `CALLER`, `CALL`, and `REVERT` opcodes are also banned, except in the special case that they appear as part of one of the following "magic strings" of bytecode:

1. `CALLER PUSH1 0x00 SWAP1 GAS CALL PC PUSH1 0x0E ADD JUMPI RETURNDATASIZE PUSH1 0x00 DUP1 RETURNDATACOPY RETURNDATASIZE PUSH1 0x00 REVERT JUMPDEST RETURNDATASIZE PUSH1 0x01 EQ ISZERO PC PUSH1 0x0a ADD JUMPI PUSH1 0x01 PUSH1 0x00 RETURN JUMPDEST`
2. `CALLER POP PUSH1 0x00 PUSH1 0x04 GAS CALL`

The first magic string MUST:

**TODO**

The second magic string does this:

## Defining the OVM Sandbox

The OVM Sandbox consists of:

- The Execution Manager
- The State Manager
- The Safety Cache and Safety Checker
- The

## Transaction lifecycle

[`TRANSACTION`s](./../data-structures.md#transaction) submitted to L2 (via Transaction Queue, Sequencer or other source) have a data-structure similar to the format of L1 Transactions.

### 1. Modify to ensure the Execution Manager is called first

All transactions begin as calls to the Execution Manager contract.

Thus clients (the Sequencer or Verifiers) MUST modify the `Transaction` with the following modifications:

1. Replace the `to` field with the Execution Manager’s address.
2. Encode the `data` field as arguments to `run()`.

```jsx
function run(
  Transaction _transaction,
  address _ovmStateManager
)
```

Where the parameters are:

- `Transaction _transaction`:
- `address _ovmStateManager`:

Importantly, the `Transaction` parameter includes all the contextual information (ie. `ovmTIMESTAMP`, `ovmBLOCKNUMBER`, `ovmL1QUEUEORIGIN`, `ovmL1TXORIGIN`, `ovmGASLIMIT`) which will be made available to the execution environment.

### 2. OVM Messages via the Sequencer Entrypoint

For **Sequencer transactions only**, the `Transaction.entrypoint` SHOULD be the Sequencer Entrypoint (or simply Entrypoint) contract. In order to achieve this, the Sequencer MUST modify the transaction's `to` field to the address of the Entrypoint.

The Entrypoint contract accepts a more efficient compressed calldata format. This is done at a low level, using the contract's fallback function, which expects an RLP-encoded EIP155 transaction as input.

The Entrypoint then:

- decodes the input to extract the hash, and signature of [`EIP155Transaction`](#eip155transaction).
- calculates an address using `ecrecover()`
- checks for the existence of a contract at that address
  - calls `ovmCREATEEOA` to ensure the necessary 'Account' contract exists.
- initiates an `ovmCALL` to the 'Account' contract's `execute()` function.

### 3. Execution Proceeds within the Sandbox

Given the guarantees provided by the SafetyChecker contract, henceforth all calls to overridden opcodes will be routed through the Execution Manager.

## Exception handling within the OVM

It is critical to handle different exceptions properly during execution.

### Invalid transactions

If a transaction (or more generally a call to `run()`) is 'invalid', the Execution Manager's run function should `RETURN` prior to initiating the first `ovmCALL`.

Invalid calls to to run include calls which:

- don't change the context from its default values
- have a `_gasLimit` outside the minimum and maximum transaction gas limits

### Revert

Refer to Data Structures spec for a description of [`RevertFlag`](./../data-structures.md#revertflag-enum) enum fields.

## Gas Considerations

### Epoch limitations

The OVM does not have blocks, it just maintains an ordered list of transactions. Because of this, there is no notion of a block gas limit; instead, the overall gas consumption is rate limited based on time segments, called epochs8. Before a transaction is executed, there’s a check to see if a new epoch needs to be started, and after execution its gas consumption is added on the cumulative gas used for that epoch. There is a separate gas limit per epoch for sequencer submitted transactions and “L1 to L2” transactions. Any transactions exceeding the gas limit for an epoch return early. This implies that an operator can post several transactions with varying timestamps in one on-chain batch (timestamps are defined by the sequencer, with some restrictions which we explain in the “Data Availability Batches” section).

### GAS Metering

Notably, the `GAS` opcode is not disallowed or overridden. This enables us to use the EVM's built in gas metering, but also creates an attack vector in the case that fees diverge on L1 and L2.

An important property to maintain is that the amount of gas passed to `run()`'s first `ovmCALL` is deterministic, and that within that call-frame, the `GAS` value remains deterministic. Assuming L2 geth is in consensus with L1 geth, and no _other_ L1 context is exposed to the OVM, this property should follow.

### Nuisance Gas

Nuisance-gas is used to enforce an upper-bound on the net gas cost of fraud proofs, by charging a fee for any operation that increases the number of accounts and storage slots which require proving in either the pre or post-state.

Nuisance-gas is initialized in `run()` to be equal to the transaction's `gasLimit`, but from then on the two values are treated independently.

A nuisance gas fee is charged on the following OVM operations the first time they occur:

- a new account is loaded
  - the base fee is `MIN_NUISANCE_GAS_PER_CONTRACT = 30000`
  - the variable fee is `NUISANCE_GAS_PER_CONTRACT_BYTE = 100`
- a new storage slot is read from

  - the fee is `NUISANCE_GAS_SLOAD = 20000`

- a new storage slot is written to
  - the fee is `NUISANCE_GAS_SSTORE = 20000`

If a message tries to use more nuisance gas than allowed in the message’s context, execution reverts.
