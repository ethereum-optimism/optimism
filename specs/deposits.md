# Deposits

<!-- All glossary references in this file. -->
[g-transaction-type]: glossary.md#transaction-type
[g-derivation]:  glossary.md#L2-chain-derivation
[g-deposited]: glossary.md#deposited
[g-deposits]: glossary.md#deposits
[g-l1-attr-deposit]: glossary.md#l1-attributes-deposited-transaction
[g-user-deposited]: glossary.md#user-deposited-transaction
[g-eoa]: glossary.md#eoa
[g-exec-engine]: glossary.md#execution-engine

[Deposited transactions][g-deposited], also known as [deposits][g-deposits] are transactions which
are initiated on L1, and executed on L2. This document outlines a new [transaction
type][g-transaction-type] for deposits. It also describes how deposits are initiated on L1, along
with the authorization and validation conditions on L2.

**Vocabulary note**: *deposited transaction* refers specifically to an L2 transaction, while
*deposit* can refer to the transaction at various stages (for instance when it is deposited on L1).

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [The Deposited Transaction Type](#the-deposited-transaction-type)
  - [Source hash computation](#source-hash-computation)
  - [Kinds of Deposited Transactions](#kinds-of-deposited-transactions)
  - [Validation and Authorization of Deposited Transactions](#validation-and-authorization-of-deposited-transactions)
  - [Execution](#execution)
    - [Nonce Handling](#nonce-handling)
- [Deposit Receipt](#deposit-receipt)
- [L1 Attributes Deposited Transaction](#l1-attributes-deposited-transaction)
- [Special Accounts on L2](#special-accounts-on-l2)
  - [L1 Attributes Depositor Account](#l1-attributes-depositor-account)
  - [L1 Attributes Predeployed Contract](#l1-attributes-predeployed-contract)
    - [L1 Attributes Predeployed Contract: Reference Implementation](#l1-attributes-predeployed-contract-reference-implementation)
- [User-Deposited Transactions](#user-deposited-transactions)
  - [Deposit Contract](#deposit-contract)
    - [Address Aliasing](#address-aliasing)
    - [Deposit Contract Implementation: Optimism Portal](#deposit-contract-implementation-optimism-portal)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## The Deposited Transaction Type

[deposited-tx-type]: #the-deposited-transaction-type

[Deposited transactions][g-deposited] have the following notable distinctions from existing
transaction types:

1. They are derived from Layer 1 blocks, and must be included as part of the protocol.
2. They do not include signature validation (see [User-Deposited Transactions][user-deposited]
   for the rationale).
3. They buy their L2 gas on L1 and, as such, the L2 gas is not refundable.

We define a new [EIP-2718] compatible transaction type with the prefix `0x7E` to represent a deposit transaction.

A deposit has the following fields
(rlp encoded in the order they appear here):

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

- `bytes32 sourceHash`: the source-hash, uniquely identifies the origin of the deposit.
- `address from`: The address of the sender account.
- `address to`: The address of the recipient account, or the null (zero-length) address if the
  deposited transaction is a contract creation.
- `uint256 mint`: The ETH value to mint on L2.
- `uint256 value`: The ETH value to send to the recipient account.
- `bytes data`: The input data.
- `bool isSystemTx`: If true, the transaction does not interact with the L2 block gas pool.
  - Note: boolean is disabled (enforced to be `false`) starting from the Regolith upgrade.
- `uint64 gasLimit`: The gasLimit for the L2 transaction.

In contrast to [EIP-155] transactions, this transaction type:

- Does not include a `nonce`, since it is identified by the `sourceHash`.
  API responses still include a `nonce` attribute:
  - Before Regolith: the `nonce` is always `0`
  - With Regolith: the `nonce` is set to the `depositNonce` attribute of the corresponding transaction receipt.
- Does not include signature information, and makes the `from` address explicit.
  API responses contain zeroed signature `v`, `r`, `s` values for backwards compatibility.
- Includes new `sourceHash`, `from`, `mint`, and `isSystemTx` attributes.
  API responses contain these as additional fields.

[EIP-155]:https://eips.ethereum.org/EIPS/eip-155

We select `0x7E` because transaction type identifiers are currently allowed to go up to `0x7F`.
Picking a high identifier minimizes the risk that the identifier will be used be claimed by another
transaction type on the L1 chain in the future. We don't pick `0x7F` itself in case it becomes used
for a variable-length encoding scheme.

### Source hash computation

The `sourceHash` of a deposit transaction is computed based on the origin:

- User-deposited:
  `keccak256(bytes32(uint256(0)), keccak256(l1BlockHash, bytes32(uint256(l1LogIndex))))`.
  Where the `l1BlockHash`, and `l1LogIndex` all refer to the inclusion of the deposit log event on L1.
  `l1LogIndex` is the index of the deposit event log in the combined list of log events of the block.
- L1 attributes deposited:
  `keccak256(bytes32(uint256(1)), keccak256(l1BlockHash, bytes32(uint256(seqNumber))))`.
  Where `l1BlockHash` refers to the L1 block hash of which the info attributes are deposited.
  And `seqNumber = l2BlockNum - l2EpochStartBlockNum`,
  where `l2BlockNum` is the L2 block number of the inclusion of the deposit tx in L2,
  and `l2EpochStartBlockNum` is the L2 block number of the first L2 block in the epoch.

Without a `sourceHash` in a deposit, two different deposited transactions could have the same exact hash.

The outer `keccak256` hashes the actual uniquely identifying information with a domain,
to avoid collisions between different types of sources.

We do not use the sender's nonce to ensure uniqueness because this would require an extra L2 EVM state read from the
[execution engine][g-exec-engine] during block-derivation.

### Kinds of Deposited Transactions

Although we define only one new transaction type, we can distinguish between two kinds of deposited
transactions, based on their positioning in the L2 block:

1. The first transaction MUST be a [L1 attributes deposited transaction][l1-attr-deposit], followed by
2. an array of zero-or-more [user-deposited transactions][user-deposited] submitted to the deposit
   feed contract on L1. User-deposited transactions are only present in the first block of a L2 epoch.

We only define a single new transaction type in order to minimize modifications to L1 client
software, and complexity in general.

### Validation and Authorization of Deposited Transactions

[authorization]: #validation-and-authorization-of-deposited-transaction

As noted above, the deposited transaction type does not include a signature for validation. Rather,
authorization is handled by the [L2 chain derivation][g-derivation] process, which when correctly
applied will only derive transactions with a `from` address attested to by the logs of the [L1
deposit contract][deposit-contract].

### Execution

In order to execute a deposited transaction:

First, the balance of the `from` account MUST be increased by the amount of `mint`.
This is unconditional, and does not revert on deposit failure.

Then, the execution environment for a deposited transaction is initialized based on the
transaction's attributes, in exactly the same manner as it would be for an EIP-155 transaction.

The deposit transaction is processed exactly like a type-3 (EIP-1559) transaction, with the exception of:

- No fee fields are verified: the deposit does not have any, as it pays for gas on L1.
- No `nonce` field is verified: the deposit does not have any, it's uniquely identified by its `sourceHash`.
- No access-list is processed: the deposit has no access-list, and it is thus processed as if the access-list is empty.
- No check if `from` is an Externally Owner Account (EOA): the deposit is ensured not to be an EAO through L1 address
  masking, this may change in future L1 contract-deployments to e.g. enable an account-abstraction like mechanism.
- Before the Regolith upgrade:
  - The execution output states a non-standard gas usage:
    - If `isSystemTx` is false: execution output states it uses `gasLimit` gas.
    - If `isSystemTx` is true: execution output states it uses `0` gas.
- No gas is refunded as ETH. (either by not refunding or utilizing the fact the gas-price of the deposit is `0`)
- No transaction priority fee is charged. No payment is made to the block fee-recipient.
- No L1-cost fee is charged, as deposits are derived from L1 and do not have to be submitted as data back to it.
- No base fee is charged. The total base fee accounting does not change.

Note that this includes contract-deployment behavior like with regular transactions,
and gas metering is the same (with the exception of fee related changes above), including metering of intrinsic gas.

Any non-EVM state-transition error emitted by the EVM execution is processed in a special way:

- It is transformed into an EVM-error:
  i.e. the deposit will always be included, but its receipt will indicate a failure
  if it runs into a non-EVM state-transition error, e.g. failure to transfer the specified
  `value` amount of ETH due to insufficient account-balance.
- The world state is rolled back to the start of the EVM processing, after the minting part of the deposit.
- The `nonce` of `from` in the world state is incremented by 1, making the error equivalent to a native EVM failure.
  Note that a previous `nonce` increment may have happened during EVM processing, but this would be rolled back first.

Finally, after the above processing, the execution post-processing runs the same:
i.e. the gas pool and receipt are processed identical to a regular transaction.
Starting with the Regolith upgrade however, the receipt of deposit transactions is extended with an additional
`depositNonce` value, storing the `nonce` value of the `from` sender as registered *before* the EVM processing.

Note that the gas used as stated by the execution output is subtracted from the gas pool,
but this execution output value has special edge cases before the Regolith upgrade.

Note for application developers: because `CALLER` and `ORIGIN` are set to `from`, the
semantics of using the `tx.origin == msg.sender` check will not work to determine whether
or not a caller is an EOA during a deposit transaction. Instead, the check could only be useful for
identifying the first call in the L2 deposit transaction. However this check does still satisfy
the common case in which developers are using this check to ensure that the `CALLER` is unable to
execute code before and after the call.

#### Nonce Handling

Despite the lack of signature validation, we still increment the nonce of the `from` account when a
deposit transaction is executed. In the context of a deposit-only roll up, this is not necessary
for transaction ordering or replay prevention, however it maintains consistency with the use of
nonces during [contract creation][create-nonce]. It may also simplify integration with downstream
tooling (such as wallets and block explorers).

[create-nonce]: https://github.com/ethereum/execution-specs/blob/617903a8f8d7b50cf71bf1aa733c37897c8d75c1/src/ethereum/frontier/utils/address.py#L40

## Deposit Receipt

Transaction receipts use standard typing as per [EIP-2718].
The Deposit transaction receipt type is equal to a regular receipt,
but extended with an optional `depositNonce` field.

The RLP-encoded consensus-enforced fields are:

- `postStateOrStatus` (standard): this contains the transaction status, see [EIP-658].
- `cumulativeGasUsed` (standard): gas used in the block thus far, including this transaction.
  - The actual gas used is derived from the difference in `CumulativeGasUsed` with the previous transaction.
  - Starting with Regolith, this accounts for the actual gas usage by the deposit, like regular transactions.
- `bloom` (standard): bloom filter of the transaction logs.
- `logs` (standard): log events emitted by the EVM processing.
- `depositNonce` (unique extension): Optional field. The deposit transaction persists the nonce used during execution.
  - Before Regolith, this `depositNonce` field must always be omitted.
  - With Regolith, this `depositNonce` field must always be included.

Starting with Regolith, the receipt API responses utilize the receipt changes for more accurate response data:

- The `depositNonce` is included in the receipt JSON data in API responses
- For contract-deployments (when `to == null`), the `depositNonce` helps derive the correct `contractAddress` meta-data,
  instead of assuming the nonce was zero.
- The `cumulativeGasUsed` accounts for the actual gas usage, as metered in the EVM processing.

[EIP-658]: https://eips.ethereum.org/EIPS/eip-658

## L1 Attributes Deposited Transaction

[l1-attr-deposit]: #l1-attributes-deposited-transaction

An [L1 attributes deposited transaction][g-l1-attr-deposit] is a deposit transaction sent to the [L1
attributes predeployed contract][predeploy].

This transaction MUST have the following values:

1. `from` is `0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001` (the address of the
[L1 Attributes depositor account][depositor-account])
2. `to` is `0x4200000000000000000000000000000000000015` (the address of the [L1 attributes predeployed
   contract][predeploy]).
3. `mint` is `0`
4. `value` is `0`
5. `gasLimit` is set to 150,000,000.
6. `isSystemTx` is set to `true`.
7. `data` is an [ABI] encoded call to the [L1 attributes predeployed contract][predeploy]'s
   `setL1BlockValues()` function with correct values associated with the corresponding L1 block (cf.
   [reference implementation][l1-attr-ref-implem]).

If the Regolith upgrade is active, some fields are overridden:

1. `gasLimit` is set to 1,000,000
2. `isSystemTx` is set to `false`

This system-initiated transaction for L1 attributes is not charged any ETH for its allocated `gasLimit`,
as it is effectively part of the state-transition processing.

## Special Accounts on L2

The L1 attributes deposit transaction involves two special purpose accounts:

1. The L1 attributes depositor account
2. The L1 attributes predeployed contract

### L1 Attributes Depositor Account

[depositor-account]: #l1-attributes-depositor-account

The depositor account is an [EOA][g-eoa] with no known private key. It has the address
`0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001`. Its value is returned by the `CALLER` and `ORIGIN`
opcodes during execution of the L1 attributes deposited transaction.

### L1 Attributes Predeployed Contract

[predeploy]: #l1-attributes-predeployed-contract

A predeployed contract on L2 at address `0x4200000000000000000000000000000000000015`, which holds
certain block variables from the corresponding L1 block in storage, so that they may be accessed
during the execution of the subsequent deposited transactions.

The predeploy stores the following values:

- L1 block attributes:
  - `number` (`uint64`)
  - `timestamp` (`uint64`)
  - `basefee` (`uint256`)
  - `hash` (`bytes32`)
- `sequenceNumber` (`uint64`): This equals the L2 block number relative to the start of the epoch,
  i.e. the L2 block distance to the L2 block height that the L1 attributes last changed,
  and reset to 0 at the start of a new epoch.
- System configurables tied to the L1 block, see [System configuration specification](./system_config.md):
  - `batcherHash` (`bytes32`): A versioned commitment to the batch-submitter(s) currently operating.
  - `l1FeeOverhead` (`uint256`): The L1 fee overhead to apply to L1 cost computation of transactions in this L2 block.
  - `l1FeeScalar` (`uint256`): The L1 fee scalar to apply to L1 cost computation of transactions in this L2 block.

The contract implements an authorization scheme, such that it only accepts state-changing calls from
the [depositor account][depositor-account].

The contract has the following solidity interface, and can be interacted with according to the
[contract ABI specification][ABI].

[ABI]: https://docs.soliditylang.org/en/v0.8.10/abi-spec.html

#### L1 Attributes Predeployed Contract: Reference Implementation

[l1-attr-ref-implem]: #l1-attributes-predeployed-contract-reference-implementation

A reference implementation of the L1 Attributes predeploy contract can be found in [L1Block.sol].

[L1Block.sol]: ../packages/contracts-bedrock/contracts/L2/L1Block.sol

After running `yarn build` in the `packages/contracts` directory, the bytecode to add to the genesis
file will be located in the `deployedBytecode` field of the build artifacts file at
`/packages/contracts/artifacts/contracts/L2/L1Block.sol/L1Block.json`.

## User-Deposited Transactions

[user-deposited]: #user-deposited-transactions

[User-deposited transactions][g-user-deposited] are [deposited transactions][deposited-tx-type]
generated by the [L2 Chain Derivation][g-derivation] process. The content of each user-deposited
transaction are determined by the corresponding `TransactionDeposited` event emitted by the
[deposit contract][deposit-contract] on L1.

1. `from` is unchanged from the emitted value (though it may have been transformed to an alias in
   the deposit feed contract).
2. `to` is any 20-byte address (including the zero address)
    - In case of a contract creation (cf. `isCreation`), this address is always zero.
3. `mint` is set to the emitted value.
4. `value` is set to the emitted value.
5. `gaslimit` is unchanged from the emitted value. It must be at least 21000.
6. `isCreation` is set to `true` if the transaction is a contract creation, `false` otherwise.
7. `data` is unchanged from the emitted value. Depending on the value of `isCreation` it is handled
   as either calldata or contract initialization code.
8. `isSystemTx` is set by the rollup node for certain transactions that have unmetered execution.
  It is `false` for user deposited transactions

### Deposit Contract

[deposit-contract]: #deposit-contract

The deposit contract is deployed to L1. Deposited transactions are derived from the values in
the `TransactionDeposited` event(s) emitted by the deposit contract.

The deposit contract is responsible for maintaining the [guaranteed gas market](./guaranteed-gas-market.md),
charging deposits for gas to be used on L2, and ensuring that the total amount of guaranteed
gas in a single L1 block does not exceed the L2 block gas limit.

The deposit contract handles two special cases:

1. A contract creation deposit, which is indicated by setting the `isCreation` flag to `true`.
   In the event that the `to` address is non-zero, the contract will revert.
2. A call from a contract account, in which case the `from` value is transformed to its L2
   [alias][address-aliasing].

#### Address Aliasing

[address-aliasing]: #address-aliasing

If the caller is a contract, the address will be transformed by adding
`0x1111000000000000000000000000000000001111` to it. The math is `unchecked` and done on a
Solidity `uint160` so the value will overflow. This prevents attacks in which a
contract on L1 has the same address as a contract on L2 but doesn't have the same code. We can safely ignore this
for EOAs because they're guaranteed to have the same "code" (i.e. no code at all). This also makes
it possible for users to interact with contracts on L2 even when the Sequencer is down.

#### Deposit Contract Implementation: Optimism Portal

A reference implementation of the deposit contract can be found in [OptimismPortal.sol].

[OptimismPortal.sol]: ../packages/contracts-bedrock/contracts/L1/OptimismPortal.sol
