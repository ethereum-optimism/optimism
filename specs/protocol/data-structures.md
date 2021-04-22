# Optimistic Ethereum Data Structures

For convenience, the data structures are currently organized according to the solidity file in which they are defined. We may wish to relocate them.

- [Optimistic Ethereum Data Structures](#optimistic-ethereum-data-structures)
  - [Chain](#chain)
    - [TransactionChainElement](#transactionchainelement)
    - [QueueElement](#queueelement)
    - [ChainBatchHeader](#chainbatchheader)
    - [Extra Data](#extra-data)
    - [BatchContext](#batchcontext)
    - [ChainInclusionProof](#chaininclusionproof)
  - [Accounts](#accounts)
    - [Address](#address)
    - [EVMAccount](#evmaccount)
    - [Account](#account)
    - [EOASignatureType](#eoasignaturetype)
    - [EIP155Transaction](#eip155transaction)
  - [Execution](#execution)
    - [Transaction](#transaction)
    - [QueueOrigin](#queueorigin)
    - [RevertFlag (enum)](#revertflag-enum)
    - [GasMetadataKey (enum)](#gasmetadatakey-enum)
    - [GasMeterConfig](#gasmeterconfig)
    - [GlobalContext](#globalcontext)
    - [TransactionContext](#transactioncontext)
    - [TransactionRecord](#transactionrecord)
    - [MessageContext](#messagecontext)
    - [MessageRecord](#messagerecord)
  - [Bridge](#bridge)
    - [L2MessageInclusionProof](#l2messageinclusionproof)
    - [L2ToL1Message](#l2tol1message)
  - [Verification](#verification)
    - [ItemState (enum)](#itemstate-enum)
    - [TransitionPhase (enum)](#transitionphase-enum)
  - [Misc](#misc)
    - [RLPItem](#rlpitem)
    - [RLPItemType (enum)](#rlpitemtype-enum)

## Chain

### TransactionChainElement

The elements which are added (in batches) in the Canonical Transaction Chain. One per transaction.

| Name        | Type    | Description                                                           | Validation                                             | Notes                                |
| ----------- | ------- | --------------------------------------------------------------------- | ------------------------------------------------------ | ------------------------------------ |
| isSequenced | bool    | The transaction was included by the Sequencer (rather than enqueued). | \*                                                     | \*️⃣ Redundant with QueueOrigin enum. |
| queueIndex  | uint256 | The index of the transaction in the queue.                            | 0 for Sequenced Transactions.                          |                                      |
| timestamp   | uint256 | L1 timestamp at which the transaction was included in the chain.      | Monotonically increasing. Equivalent within a context. | Description needs confirmation       |
| blockNumber | uint256 | L1 block number at which the transaction was included in the chain.   | Monotonically increasing. Equivalent within a context. | Description needs confirmation       |
| txData      | bytes   | Transaction data                                                      |                                                        |                                      |

Refer to "[Verifying transaction inclusion in the CTC](#verifying-transaction-inclusion-in-the-ctc)" for details on its usage.

**Note:** this structure is tightly coupled with the [Transaction](#transaction) type, and should perhaps be combined or otherwise refactored.

### QueueElement

Elements in the Queue.

| Name            | Type    | Description                                                                                                              | Validation | Notes                                                   |
| --------------- | ------- | ------------------------------------------------------------------------------------------------------------------------ | ---------- | ------------------------------------------------------- |
| transactionHash | bytes32 | keccak256( abi.encode( \_transaction.l1TxOrigin, \_transaction.entrypoint, \_transaction.gasLimit, \_transaction.data )) |            | \*️⃣ As formulated, there will definitely be collisions. |
| timestamp       | uint40  | timestamp at the time of enqueue                                                                                         |            |                                                         |
| blockNumber     | uint40  | block number at the time of enqueue                                                                                      |            |                                                         |

### ChainBatchHeader

The chain itself is compressed into a sequence of batches. `ChainBatchHeader`s contain information related to a batch of transactions.
This structure is used in both the CTC and SCC.

| Name              | Type             | Description                                                          | Validation                        |
| ----------------- | ---------------- | -------------------------------------------------------------------- | --------------------------------- |
| batchIndex        | uint256          | Index of a given batch in the array of batches.                      | MUST be monotonically increasing. |
| batchRoot         | bytes32          | Root node of a Merkle Tree encoding the elements within a batch.     | \*                                |
| batchSize         | uint256          | Number of transactions in the batch.                                 | \*                                |
| prevTotalElements | uint256          | Total number of transactions submitted prior to this batch.          | \*                                |
| extraData         | bytes27 / bytes? | Provides additional context data for a batch (see `ExtraData` below) | \*                                |

### Extra Data

Additional context for use in the execution environment.

- (\*️⃣ This structure is not explicitly defined as a struct in solidity. It's also represented as both bytes and bytes27 in different places.)
- (\*️⃣ Consider renaming `extraData` to something more descriptive such as `CanonicalContext` or `CanonicalBatchContext`.

| Name           | Type   | Description                                                                   | Validation                        |
| -------------- | ------ | ----------------------------------------------------------------------------- | --------------------------------- |
| totalElements  | uint40 | Sum total number of transactions submitted prior to AND including this batch. |                                   |
| nextQueueIndex | uint40 | Index of the queue element to process in the next batch.                      |                                   |
| timestamp      | uint40 | The timestamp of the _last context_ in the `ChainBatchHeader`.                | MUST increase monotonically.      |
| blockNumber    | uint40 | The blockNumber of the _last context_ in the `ChainBatchHeader`.              | MUST increase monotonically by 1. |

### BatchContext

The Sequencer calls `appendSequencerBatch()` with an array of `BatchContext` structs. The CTC contract uses the `BatchContext` to determine the ordering of transactions to append to the chain, and provides the timestamp and blocknumber which will be provided in the transaction context during execution.
The nomenclature is slightly confusing here, because each "Batch" contains multiple `BatchContexts`.

Additional context for use in the execution environment, and for enforcing monotonicity of contexts.
The difference or lack thereof of this and the previous structure needs to be elaborated.

| Name                           | Type    | Description                                                                          | Validation                                                                  |
| ------------------------------ | ------- | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| numSequencedTransactions       | uint256 | Number of transactions to append to the CTC from calldata provided by the Sequencer. | \* (Zero is valid and will result in appending only enqueued transactions)  |
| numSubsequentQueueTransactions | uint256 | Number of transactions to append                                                     | \* (Zero is valid and will result in appending only Sequencer transactions) |
| timestamp                      | uint256 | Timestamp of the transaction.                                                        | Must be strictly increasing.                                                |
| blockNumber                    | uint256 | Block Number of the transaction.                                                     | Must be strictly increasing.                                                |

### ChainInclusionProof

(\*️⃣ I noticed the proof is decomposed before passing to Lib_MerkleTree.verify(). It might be cleaner to pass the struct and decompose in the lib.)
Data required to verify the inclusion of a transaction (leaf) in the Merkle Tree defined by `ChainBatchHeader.batchRoot`.

| Name     | Type      | Description                                                                                | Validation                                                                                                                                |
| -------- | --------- | ------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| index    | uint256   | The index in the tree of the leaf for which inclusion is being proven.                     | Must be less than total number of leaves in the tree.                                                                                     |
| siblings | bytes32[] | Array of sibling nodes in the inclusion proof, starting from depth 0 (bottom of the tree). | Length must be greater than 0. Length must be equal to the integer ceiling of log2(x), where x is the total number of leaves in the tree. |

## Accounts

### Address

Address is a type alias for bytes20.

### EVMAccount

An Ethereum (Layer 1) account.

| Name        | Type    | Description                                                                                                                                                                                                                                                                         | Validation                   |
| ----------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- |
| nonce       | uint256 | A one time use value used to prevent collisions during contract creation. Note that because all accounts are 'abstract', the protocol does not natively prevent nonce reuse. Nonce validation must be handled by the contract account implementation (see OVM_ECDSAContractAccount) | Must increase monotonically. |
| balance     | uint256 | The native ETH balance in wei                                                                                                                                                                                                                                                       | \*                           |
| storageRoot | bytes32 | Root node of a Merkle Patricia Tree encoding the storage contents of the account.                                                                                                                                                                                                   | \*                           |
| storageRoot | bytes32 | Keccak-256 hash of the bytecode of the account                                                                                                                                                                                                                                      | \*                           |

### Account

An Optimistic Ethereum account (L2). Similar to an EVM account with some additional fields.

| Name        | Type           | Description                                                                            | Validation                                                  |
| ----------- | -------------- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| nonce       | uint256        | See EVMAccount.nonce                                                                   | MUST be initialized at 1. MUST increase monotonically by 1. |
| balance     | uint256        | See EVMAccount.balance                                                                 | MUST be equal to 0.                                         |
| storageRoot | bytes32        | See EVMAccount.storageRoot                                                             | \*                                                          |
| codeHash    | bytes32        | See EVMAccount.codeHash                                                                | \*                                                          |
| ethAddress  | address (link) | The layer-1 Ethereum address associated with an Optimistic Ethereum Account            | \*                                                          |
| isFresh     | bool           | Indicates that an account will or has been created during the course of Fraud Proving. | \*                                                          |

### EOASignatureType

An enum indicating the type of a ECDSA signature.

| Name               | Description                                                                    |
| ------------------ | ------------------------------------------------------------------------------ |
| EIP155_TRANSACTION | Indicates that the signed message includes replay protection based on EIP-155. |
| ETH_SIGNED_MESSAGE | Indicates that the signed message does not include replay protection.          |

### EIP155Transaction

<!-- todo update to EIP155Tx -->

Encoding used in the `OVM_ECDSAContractAccount` and `OVM_SequencerEntrypoint` (though seems to have been removed on the `OZ` branch).

| Name     | Type    | Description | Validation |
| -------- | ------- | ----------- | ---------- |
| nonce    | uint256 |             |            |
| gasPrice | uint256 |             |            |
| gasLimit | uint256 |             |            |
| to       | address |             |            |
| value    | uint256 |             |            |
| data     | bytes   |             |            |
| chainId  | uint256 |             |            |

## Execution

### Transaction

Optimistic Ethereum transaction data.

| Name          | Type        | Description                                                                                                        | Validation                                                                                          | Notes |
| ------------- | ----------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------- | ----- |
| timestamp     | uint256     | Timestamp of the transaction.                                                                                      | MUST be identical across transactions with a context, and monotonically increasing across contexts. |       |
| blockNumber   | uint256     | Block Number of the transaction.                                                                                   | MUST be identical across transactions with a context, and monotonically increasing across contexts. |       |
| l1QueueOrigin | QueueOrigin | See QueueOrigin below.                                                                                             | \*                                                                                                  |       |
| l1TxOrigin    | address     | The EOA or contract address on L1 which called the CTC to enqueue the transaction.                                 | \*                                                                                                  |       |
| entrypoint    | address     | The address to call within the ExecutionManager's run() function.                                                  | \*                                                                                                  |       |
| gasLimit      | uint256     | The gas limit (minus the minTransactionGasLimit) to pass to the call within the ExecutionManager's run() function. | MUST be > gasMeterConfig.minTransactionGasLimit MUST be < gasMeterConfig.maxTransactionGasLimit     |       |
| data          | bytes       | Transaction input data                                                                                             | \*                                                                                                  |       |

### QueueOrigin

Enum indicating whether a transaction on L2 was provided by the Sequencer or the Queue.

| Name            | Description                                         |
| --------------- | --------------------------------------------------- |
| SEQUENCER_QUEUE | The transaction was included by the Sequencer       |
| L1TOL2_QUEUE    | The transaction was enqueued via an L1 transaction. |

### RevertFlag (enum)

Enum encoding the reason for reverting during execution in the OVM.

| Name                 | Description |
| -------------------- | ----------- |
| OUT_OF_GAS           |             |
| INTENTIONAL_REVERT   |             |
| EXCEEDS_NUISANCE_GAS |             |
| INVALID_STATE_ACCESS |             |
| UNSAFE_BYTECODE      |             |
| CREATE_COLLISION     |             |
| STATIC_VIOLATION     |             |
| CREATOR_NOT_ALLOWED  |             |
| CALLER_NOT_ALLOWED   |             |

### GasMetadataKey (enum)

Predefined keys for values to look up in storage of the contract act `GAS_METADATA_ADDRESS`.

| Name                           | Description |
| ------------------------------ | ----------- |
| CURRENT_EPOCH_START_TIMESTAMP  |             |
| CUMULATIVE_SEQUENCER_QUEUE_GAS |             |
| CUMULATIVE_L1TOL2_QUEUE_GAS    |             |
| PREV_EPOCH_SEQUENCER_QUEUE_GAS |             |
| PREV_EPOCH_L1TOL2_QUEUE_GAS    |             |

### GasMeterConfig

Constraints for gas metering during execution.

| Name                   | Type    | Description | Validation                               |
| ---------------------- | ------- | ----------- | ---------------------------------------- |
| minTransactionGasLimit | uint256 |             | Must be less than maxTransactionGasLimit |
| maxTransactionGasLimit | uint256 |             | \*                                       |
| maxGasPerQueuePerEpoch | uint256 |             | \*                                       |
| secondsPerEpoch        | uint256 |             |                                          |

### GlobalContext

Context which remains consistent across all transactions and messages.

| Name       | Type    | Description | Validation |
| ---------- | ------- | ----------- | ---------- |
| ovmCHAINID | uint256 | 10          | MUST be 10 |

### TransactionContext

Context which remains consistent within a transaction.

| Name             | Type        | Description | Validation |
| ---------------- | ----------- | ----------- | ---------- |
| ovmL1QUEUEORIGIN | QueueOrigin |             |            |
| ovmTIMESTAMP     | uint256     |             |            |
| ovmNUMBER        | uint256     |             |            |
| ovmGASLIMIT      | uint256     |             |            |
| ovmTXGASLIMIT    | uint256     |             |            |
| ovmL1TXORIGIN    | address     |             |            |

### TransactionRecord

A record of the result of transaction execution.

| Name         | Type    | Description | Validation |
| ------------ | ------- | ----------- | ---------- |
| ovmGasRefund | uint256 |             |            |

### MessageContext

Context pertaining to a given message within the OVM.

| Name       | Type    | Description | Validation |
| ---------- | ------- | ----------- | ---------- |
| ovmCALLER  | address |             |            |
| ovmADDRESS | address |             |            |
| isStatic   | bool    |             |            |

### MessageRecord

A record of the result of message execution.

| Name            | Type    | Description                | Validation |
| --------------- | ------- | -------------------------- | ---------- |
| nuisanceGasLeft | uint256 | The nuisance gas remaining |            |

## Bridge

### L2MessageInclusionProof

Data required to verify the inclusion of an element in the SecureMerkleTrie which encodes the L2 state.

| Name                 | Type                | Description | Validation | Notes |
| -------------------- | ------------------- | ----------- | ---------- | ----- |
| stateRoot            | bytes32             |             |            |       |
| stateRootBatchHeader | ChainBatchHeader    |             |            |       |
| stateRootProof       | ChainInclusionProof |             |            |       |
| stateTrieWitness     | bytes               |             |            |       |
| storageTrieWitness   | bytes               |             |            |       |

### L2ToL1Message

Data used to send a message from L2 to and relay it on L1.

| Name         | Type                    | Description           | Validation |
| ------------ | ----------------------- | --------------------- | ---------- |
| target       | address                 | Address to call on L1 |            |
| sender       | address                 | Sender address        |            |
| message      | bytes                   |                       |            |
| messageNonce | uint256                 |                       |            |
| proof        | L2MessageInclusionProof |                       |            |

## Verification

### ItemState (enum)

Possible states of both accounts and storage slots. Used for metering nuisance gas during execution for a fraud proof.

| Name            | Description |
| --------------- | ----------- |
| ITEM_UNTOUCHED, |             |
| ITEM_LOADED,    |             |
| ITEM_CHANGED,   |             |
| ITEM_COMMITTED  |             |

### TransitionPhase (enum)

Distinct phases in the lifecycle of a fraud proof.

| Name           | Description |
| -------------- | ----------- |
| PRE_EXECUTION  |             |
| POST_EXECUTION |             |
| COMPLETE       |             |

## RLP Structures

### RLPItem

| Name   | Type | Description | Validation |
| ------ | ---- | ----------- | ---------- |
| length | uint |             |            |
| ptr    | uint |             |            |

### RLPItemType (enum)

| Name       | Description |
| ---------- | ----------- |
| DATA_ITEM, |             |
| LIST_ITEM  |             |
