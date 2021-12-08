# Deposits

<!-- All glossary references in this file. -->
[transaction-type]: /glossary.md#transaction-type

Deposits are transactions initiated on L1, and executed on L2. This document outlines a new
[Transaction Type][transaction-type] for deposits. It also describes how deposits are initiated on
L1, along with the authorization and validation conditions on L2.

## The Deposit Transaction Type

Deposit transactions have the following notable distinctions from existing transaction types:

1. They are initiated by the system as part of the protocol.
2. They do not require signature validation.
We define a new [EIP-2718] compatible transaction type with the prefix `0x7E`, and the following
fields:

[EIP-2718]: <https://eips.ethereum.org/EIPS/eip-2718>

<!-- ToDo: set to more GoLang like type defs? -->
- `address to`
- `address from`
- `uint256 value`
- `bytes data`

This transaction type contains a subset of the fields used in [EIP-155], but does not include
signature information.

[EIP-155]: https://eips.ethereum.org/EIPS/eip-155
We select `0x7E` because transaction type identifiers are currently allowed to go up to `0x7F`.
Picking a high identifier minimizes the risk that the identifier will be used by Ethereum in the
future. We don't pick `0x7F` itself in case it becomes used for a variable-length encoding scheme.

Although in practice we define only one new Transaction Type we can distinguish between two distinct
transactions which occur in the deposit block, based on their positioning. The first transaction
MUST be the [L1 Attributes Deposit Transaction][l1-attributes-deposit-transaction], followed by a
dynamic array of [Deposited Transactions][deposited-transactions] submitted to the Deposit Feed
contract by accounts on L1.

## L1 Attributes Deposit Transaction

[l1-attributes-deposit-transaction]: #l1-attributes-deposit-transaction

This transaction is a call to the [Layer 1 Attributes Predeploy][l1-attributes-predeploy] contract.

This transaction MUST have the following values:

1. `from` is the Depositor Account `0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001`.
1. `to` is `0x4200000000000000000000000000000000000014` (the address of the L1 attributes predeploy
   contract).
1. `value` is `0`
1. `data` is an abi encoded call to the [L1 Attributes Predeploy] contract's `setL1BlockValues()`
   function with correct values associated with the corresponding L1 block.

## Special Accounts on L2

The L1 Attributes Deposit Transaction involves two special purpose accounts:

1. The L1 Attributes Depositor Account
2. The L1 Attributes Predeploy

### L1 Attributes Depositor Account

[l1-attributes-depositor-account]: #l1-attributes-depositor-account

The Depositor Account is an EOA with no known private key. It has the address
`0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001`. Its value returned by the `CALLER` and `ORIGIN`
opcodes during execution of the L1 Attributes Deposit Transaction.

### L1 Attributes Predeploy

[l1-attributes-predeploy]: #l1-attributes-predeploy

A predeployed contract on L2 at address `0x4200000000000000000000000000000000000014`, which holds
certain block variables from the corresponding L1 block in storage, so that they may be accessed
during the execution of the subsequent deposited transactions.

The contract implements an authorization scheme, such that it only accepts state-changing calls from
the [Depositor Account].

The contract has the following solidity interface, and can be interacted with according to the
[contract ABI specification][ABI].

[ABI]: https://docs.soliditylang.org/en/v0.8.10/abi-spec.html

```solidity
interface L1BlockValues {

  function setL1BlockValues(
    uint256 number,
    uint256 timestamp,
    uint256 baseFee,
    bytes32 hash
  ) external;

  function l1Number() view;
  function l1Timestamp() view;
  function l1BaseFee() view;
  function l1Hash() view;
}
```

## Deposited Transactions

[deposited-transactions]: #deposited-transactions

"Deposited Transactions" are derived from logs emitted by the [Deposit Feed
contract][deposit-feed-contract] on L1.

### Deposit Feed Contract

[deposit-feed-contract]: #deposit-feed-contract

The Deposit Feed contract is deployed to L1. Deposited Transactions are derived from the values in
the `TransactionDeposited` event(s) emitted by the Deposit Feed contract.

The Deposit Feed handles two special cases:

1. A `to` value of `0`, which results in a contract creation on L2.
2. A call from a contract account, in which case the `from` value is transformed to its L2 alias.
   This prevents attacks in which a contract on L1 has the same address as a contract on L2 but
   doesn't have the same code. We can safely ignore this for EOAs because they're guaranteed to have
   the same "code" (i.e. no code at all). This also makes it possible for users to interact with
   contracts on L2 even when the Sequencer is down.

A solidity like pseudocode implementation demonstrates the functionality:

```solidity
contract DepositFeed {
  event TransactionDeposited(
    address indexed from,
    address indexed to,
    uint256 value,
    bytes _data
  );

  event TransactionDeposited(
    address indexed from,
    uint256 value,
    bytes _data
  );

  function depositTransaction(
    address to,
    uint256 value,
    bytes memory _data
  ) external {
    address from;
    if (msg.sender == tx.origin) {
        from = msg.sender;
    } else {
        from = msg.sender + 0x1111000000000000000000000000000000001111;
    }

    if(_to == address(0)) {
        emit TransactionDeposited(
          msg.sender,
          value,
          initCode
        );
    } else {
      emit TransactionDeposited(
        msg.sender,
        to,
        value,
        initCode
      );
    }
  }
}
```
