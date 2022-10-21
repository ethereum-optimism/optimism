# Withdrawals

<!-- All glossary references in this file. -->
[g-deposits]: glossary.md#deposits
[g-deposited]: glossary.md#deposited-transaction
[deposit-tx-type]: glossary.md#deposited-transaction-type

[g-withdrawal]: glossary.md#withdrawal
[g-mpt]: glossary.md#merkle-patricia-trie
[g-relayer]: glossary.md#withdrawals
[g-execution-engine]: glossary.md#execution-engine

[Withdrawals][g-withdrawal] are cross domain transactions which are initiated on L2, and finalized by a transaction
executed on L1. Notably, withdrawals may be used by and L2 account to call an L1 contract, or to transfer ETH from
an L2 account to an L1 account.

**Vocabulary note**: *withdrawal* can refer to the transaction at various stages of the process, but we introduce
more specific terms to differentiate:

- A *withdrawal initiating transaction* refers specifically to a transaction on L2 sent to the Withdrawals predeploy.
- A *withdrawal finalizing transaction* refers specifically to an L1 transaction which finalizes and relays the
  withdrawal.

Withdrawals are initiated on L2 via a call to the Message Passer predeploy contract, which records the important
properties of the message in its storage. Withdrawals are finalized on L1 via a call to the `L2WithdrawalVerifier`
contract, which proves the inclusion of this withdrawal message.

In this way, withdrawals are different from [deposits][g-deposits] which make use of a special transaction type in the
[execution engine][g-execution-engine] client. Rather, withdrawals transaction must use smart contracts on L1 for
finalization.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Withdrawal Flow](#withdrawal-flow)
  - [On L2](#on-l2)
  - [On L1](#on-l1)
- [The L2ToL1MessagePasser Contract](#the-l2tol1messagepasser-contract)
  - [Addresses are not Aliased on Withdrawals](#addresses-are-not-aliased-on-withdrawals)
- [The Optimism Portal Contract](#the-optimism-portal-contract)
- [Withdrawal Verification and Finalization](#withdrawal-verification-and-finalization)
- [Security Considerations](#security-considerations)
  - [Key Properties of Withdrawal Verification](#key-properties-of-withdrawal-verification)
  - [Handling Successfully Verified Messages That Fail When Relayed](#handling-successfully-verified-messages-that-fail-when-relayed)
- [Summary of Definitions](#summary-of-definitions)
  - [Constants](#constants)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Withdrawal Flow

We first describe the end to end flow of initiating and finalizing a withdrawal:

### On L2

An L2 account sends a withdrawal message (and possibly also ETH) to the `L2ToL1MessagePasser` predeploy contract.
   This is a very simple contract that stores the a hash of the withdrawal data.

### On L1

1. A [relayer][g-relayer] submits the required inputs to the `OptimismPortal` contract. The relayer need
   not be the same entity which initiated the withdrawal on L2.
   These inputs include the withdrawal transaction data, inclusion proofs, and a timestamp. The timestamp
   must be one for which an L2 output root exists, which commits to the withdrawal as registered on L2.
2. The `OptimismPortal` contract retrieves the output root for the given timestamp from the `OutputOracle`'s
   `l2Outputs()` function, and performs the remainder of the verification process internally.
3. If verification fails, the call reverts. Otherwise the call is forwarded, and the hash is recorded to prevent it from
   from being replayed.

## The L2ToL1MessagePasser Contract

[message-passer-contract]: #the-l2tol1messagepasser-contract

A withdrawal is initiated by calling the L2ToL1MessagePasser contract's `initiateWithdrawal` function.
The L2ToL1MessagePasser is a simple predeploy contract at `0x4200000000000000000000000000000000000016`
which stores messages to be withdrawn.

```js
interface L2ToL1MessagePasser {
    event MessagePassed(
        uint256 indexed nonce, // this is a global nonce value for all withdrawal messages
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data,
        bytes32 withdrawalHash
    );

    event WithdrawerBalanceBurnt(uint256 indexed amount);

    function burn() external;

    function initiateWithdrawal(address _target, uint256 _gasLimit, bytes memory _data) payable external;

    function nonce() view external returns (uint256);

    function sentMessages(bytes32) view external returns (bool);
}

```

The `MessagePassed` event includes all of the data that is hashed and
stored in the `sentMessages` mapping, as well as the hash itself.

### Addresses are not Aliased on Withdrawals

[address-aliasing]: #no-address-aliasing

When a contract makes a deposit, the sender's address is [aliased](./deposits.md#address-aliasing). The same is not true
of withdrawals, which do not modify the sender's address. The difference is that:

- on L2, the deposit sender's address is returned by the `CALLER` opcode, meaning a contract cannot easily tell if the
  call originated on L1 or L2, whereas
- on L1, the withdrawal sender's address is accessed by calling the `l2Sender`() function on the `OptimismPortal`
  contract.

Calling `l2Sender()` removes any ambiguity about which domain the call originated from. Still, developers will need to
recognize that having the same address does not imply that a contract on L2 will behave the same as a contract on L1.

## The Optimism Portal Contract

The Optimism Portal serves as both the entry and exit point to the Optimism L2. It is a contract which inherits from
the [DepositFeed](./deposits.md#deposit-contract) contract, and in addition provides the following interface for
withdrawals:

```js
interface OptimismPortal {

    event WithdrawalFinalized(bytes32 indexed);

    function l2Sender() returns(address) external;

    function finalizeWithdrawalTransaction(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data,
        uint256 _timestamp,
        WithdrawalVerifier.OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    )
}
```

## Withdrawal Verification and Finalization

The following inputs are required to verify and finalize a withdrawal:

- Withdrawal transaction data:
  - `nonce`: Nonce for the provided message.
  - `sender`: Message sender address on L2.
  - `target`: Target address on L1.
  - `value`: ETH to send to the target.
  - `data`: Data to send to the target.
  - `gasLimit`: Gas to be forwarded to the target.
- Proof and verification data:
  - `timestamp`: The L2 timestamp corresponding with the output root.
  - `outputRootProof`: Four `bytes32` values which are used to derive the output root.
  - `withdrawalProof`: An inclusion proof for the given withdrawal in the L2ToL1MessagePasser contract.

These inputs must satisfy the following conditions:

1. The `timestamp` is at least `FINALIZATION_PERIOD` seconds old.
1. `OutputOracle.l2Outputs(timestamp)` returns a non-zero value `l2Output`.
1. The keccak256 hash of the `outputRootProof` values is equal to the `l2Output`.
1. The `withdrawalProof` is a valid inclusion proof demonstrating that a hash of the Withdrawal transaction data
   is contained in the storage of the L2ToL1MessagePasser contract on L2.

## Security Considerations

### Key Properties of Withdrawal Verification

1. It should not be possible 'double spend' a withdrawal, ie. to relay a withdrawal on L1 which does not
    correspond to a message initiated on L2. For reference, see [this writeup][polygon-dbl-spend] of a vulnerability
    of this type found on Polygon.

    [polygon-dbl-spend]: https://gerhard-wagner.medium.com/double-spending-bug-in-polygons-plasma-bridge-2e0954ccadf1

1. For each withdrawal initiated on L2 (ie. with a unique `nonce`), the following properties must hold:
    1. It should only be possible to finalize the withdrawal once.
    1. It should not be possible to relay the message with any of its fields modified, ie.
        1. Modifying the `sender` field would enable a 'spoofing' attack.
        1. Modifying the `target`, `message`, or `value` fields would enable an attacker to dangerously change the
           intended outcome of the withdrawal.
        1. Modifying the `gasLimit` could make the cost of relaying too high, or allow the relayer to cause execution
           to fail (out of gas) in the `target`.

### Handling Successfully Verified Messages That Fail When Relayed

If the execution of the relayed call fails in the `target` contract, it is unfortunately not possible to determine
whether or not it was 'supposed' to fail, and whether or not it should be 'replayable'. For this reason, and to
minimize complexity, we have not provided any replay functionality, this may be implemented in external utility
contracts if desired.

## Summary of Definitions

### Constants

| Name                  | Value     | Unit    |
| --------------------- | --------- | ------- |
| `FINALIZATION_PERIOD` | `604_800` | seconds |

This `FINALIZATION_PERIOD` value is equivalent to 7 days.
