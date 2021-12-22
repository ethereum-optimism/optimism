# Deposits

<!-- All glossary references in this file. -->
[transaction-type]: /glossary.md#transaction-type
[derivation]:  /glossary.md#L2-chain-derivation
[execution-engine]: /glossary.md#execution-engine

Deposits are transactions which are initiated on L1, and executed on L2. This document outlines a new
[transaction type][transaction-type] for deposits. It also describes how deposits are initiated on
L1, along with the authorization and validation conditions on L2.

## The Deposit Transaction Type

[deposit-transaction-type]: #the-deposit-transaction-type

Deposit transactions have the following notable distinctions from existing transaction types:

1. They are derived from Layer 1 blocks, and must be included as part of the protocol.
2. They do not include signature validation (see [L1 transaction deposits][l1-transaction-deposits] for the rationale).

We define a new [EIP-2718] compatible transaction type with the prefix `0x7E`, and the following
fields (rlp encoded in the order they appear here):

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

- `address from`
- `address to`
- `uint256 value`
- `bytes data`
- `uint256 gasLimit`

In contrast to [EIP-155] transactions, this transaction type does not include signature information,
and makes the `from` address explicit.

[EIP-155]:https://eips.ethereum.org/EIPS/eip-155

We select `0x7E` because transaction type identifiers are currently allowed to go up to `0x7F`.
Picking a high identifier minimizes the risk that the identifier will be used be claimed by another
transaction type on the L1 chain in the future. We don't pick `0x7F` itself in case it becomes used
for a variable-length encoding scheme.

### Uses of the Deposit Transaction Type

Although in practice we define only one new transaction type we can distinguish between two distinct
situations which occur in the deposit block, based on their positioning.

1. The first transaction MUST be a [L1 attributes deposit][l1-attributes-deposit], followed by
2. an array of zero-or-more [L1 transaction deposits][l1-transaction-deposits] submitted to the
deposit feed contract on L1.

The rationale for creating only one new transaction type is to minimize both
modifications to L1 client software and complexity in general.

> **TODO** Specify and link to deposit blocks

### Validation and Authorization of Deposit Transaction Types

[authorization]: #validation-and-authorization-of-deposit-transaction-types

As noted above, the deposit transaction type does not include a signature for validation. Rather,
authorization is handled by the [L2 chain Derivation][derivation] process, which when
correctly processed will only derive transactions with a `from` address attested to
by the logs of the [L1 deposit feed contract][deposit-feed-contract].

In the event a deposit transaction is included which is not derived by the [execution engine][execution-engine]
using the correct derivation algorithm, the resulting state transition would be invalid.

### Execution

In order to execute a deposit transaction:

First, the balance of the `from` account MUST be increased by the amount of `value`.

Then, the execution environment for a deposit transaction is initialized based on the transaction's
values, in exactly the same manner as it would be for an EIP-155 transaction.

Specifically, a new EVM call frame targeting the `to` address is created with values initialized as
follows:

- `CALLER` and `ORIGIN` set to `from`
  - `from` is unchanged from the deposit feed contract's logs (though the address may have been
  [aliased][address-aliasing] by the deposit feed contract).
- `context.calldata` set to `data`
- `context.gas` set to `gasLimit`
- `context.value` set to `value`

#### Nonce handling

Despite the lack of signature validation, we still increment the nonce of the `from` account when a
deposit transaction is executed. In the context of a deposit-only roll up, this is not necessary
for transaction ordering or replay prevention, however it maintains consistency with the use of
nonces during [contract creation][create-nonce]. It may also simplify integration with downstream
tooling (such as wallets and block explorers).

[create-nonce]: https://github.com/ethereum/execution-specs/blob/617903a8f8d7b50cf71bf1aa733c37897c8d75c1/src/ethereum/frontier/utils/address.py#L40

## L1 Attributes Deposit

[l1-attributes-deposit]: #l1-attributes-deposit

This is a deposit transaction sent to the [L1 attributes predeploy][predeploy] contract.

This transaction MUST have the following values:

1. `from` is `0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001` (the address of the
[L1 Attributes depositor account][depositor-account])
2. `to` is `0x4200000000000000000000000000000000000014` (the address of the L1 attributes predeploy
   contract).
3. `value` is `0`
4. `gasLimit` is set to the maximum available.
5. `data` is an [ABI] encoded call to the [L1 attributes predeploy][predeploy] contract's `setL1BlockValues()`
   function with correct values associated with the corresponding L1 block.

## Special Accounts on L2

The L1 attributes deposit transaction involves two special purpose accounts:

1. The L1 attributes depositor account
2. The L1 attributes predeploy

### L1 Attributes Depositor Account

[depositor-account]: #l1-attributes-depositor-account

The depositor account is an EOA with no known private key. It has the address
`0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001`. Its value is returned by the `CALLER` and `ORIGIN`
opcodes during execution of the L1 attributes deposit transaction.

### L1 Attributes Predeploy

[predeploy]: #l1-attributes-predeploy

A predeployed contract on L2 at address `0x4200000000000000000000000000000000000014`, which holds
certain block variables from the corresponding L1 block in storage, so that they may be accessed
during the execution of the subsequent deposited transactions.

The contract implements an authorization scheme, such that it only accepts state-changing calls from
the [depositor account].

The contract has the following solidity interface, and can be interacted with according to the
[contract ABI specification][ABI].

[ABI]: https://docs.soliditylang.org/en/v0.8.10/abi-spec.html

#### Reference Implementation

A reference implementation of the L1 Attributes predeploy contract can be found in [L1Block.sol].

[L1Block.sol]: /packages/contracts/contracts/L1Block.sol

The bytecode to add to the genesis file will be located in the `deployedBytecode` of the
[JSON artifact file][l1-block-artifacts] corresponding to L1Block.sol.

[l1-block-artifacts]: /packages/contracts/artifacts/contracts/L2/L1Block.sol/L1Block.json


## L1 Transaction Deposits

[l1-transaction-deposits]: #l1-transaction-deposits

L1 transaction deposits are [deposit transactions][deposit-transaction-type] generated by the
[L2 Chain Derivation][derivation] process. The values of each transaction are determined by the
corresponding `TransactionDeposited` event emitted by the [deposit feed
contract][deposit-feed-contract] on L1.

1. `from` is unchanged from the emitted value (though it may have been transformed to an alias in
   the deposit feed contract).
2. `to` may be either:
    1. any 20-byte address (including the zero-address)
    2. `null` in which case a contract is created.
3. `value` is unchanged from the emitted value.
4. `gaslimit` is unchanged from the emitted value.
5. `data` is unchanged from the emitted value. Depending on the value of `to` it is handled as
   either calldata or initialization code depending on the value of `to`.

### Deposit Feed Contract

[deposit-feed-contract]: #deposit-feed-contract

The deposit feed contract is deployed to L1. Deposited transactions are derived from the values in
the `TransactionDeposited` event(s) emitted by the deposit feed contract.

The deposit feed handles two special cases:

1. A contract creation deposit, which is indicated by setting the `isCreation` flag to `true`.
   In the event that the `to` address is non-zero, the contract will revert.
2. A call from a contract account, in which case the `from` value is transformed to its L2 [alias][address-aliasing].

> **TODO** Define if/how ETH withdrawals occur.

#### Address aliasing

[address-aliasing]: #address-aliasing

If the caller is not a contract, the address will be ed by adding
`0x1111000000000000000000000000000000001111`. This prevents attacks in which a contract on L1
has the same address as a contract on L2 but doesn't have the same code. We can safely ignore
this for EOAs because they're guaranteed to have the same "code" (i.e. no code at all). This also
makes it possible for users to interact with contracts on L2 even when the Sequencer is down.

#### Reference Implementation

A reference implementation of the Deposit Feed contract can be found in [DepositFeed.sol].

[DepositFeed.sol]: /packages/contracts/contracts/DepositFeed.sol
