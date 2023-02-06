---
title: Sending data between L1 and L2
lang: en-US
---

Apps on Optimism can be made to interact with apps on Ethereum via a process called "bridging".
In a nutshell, **contracts on Optimism can trigger contract functions on Ethereum, and vice versa**.
With just a little bit of elbow grease, you too can create contracts that bridge the gap between Layer 1 and Layer 2!


::: tip 
[See here for a step by step tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/cross-dom-comm)
:::

<details>

<summary><b>Understanding contract calls</b></summary>

To understand the process of creating bridges between contracts on Layer 1 and Layer 2, you should first have a basic understanding of the way contracts on *Ethereum* communicate with one another.
If you're a smart contract developer, you might be familiar with stuff like this:

```solidity
contract MyContract {
    function doSomething(uint256 myFunctionParam) public {
        // ... some sort of code goes here
    }
}

contract MyOtherContract {
    function doTheThing(address myContractAddress, uint256 myFunctionParam) public {
        MyContract(myContractAddress).doSomething(myFunctionParam);
    }
}
```

Here, `MyOtherContract.doTheThing` triggers a "call" to `MyContract.doSomething`.
A "call" is defined by a few key input parameters, mainly a `target address` and some `calldata`.
In this specific example, the `target address` is going to be the address of our instance of `MyContract`.
The `calldata`, on the other hand, depends on the function we're trying to call.
Solidity uses an encoding scheme called [Contract ABI](https://docs.soliditylang.org/en/v0.8.4/abi-spec.html) to both [select which function to call](https://docs.soliditylang.org/en/v0.8.4/abi-spec.html#function-selector) and to [encode function input arguments](https://docs.soliditylang.org/en/v0.8.4/abi-spec.html#argument-encoding).

Solidity gives us some useful tools to perform this same encoding manually.
For the sake of learning, let's take a look at how we can duplicate the same code with a manual encoding:

```solidity
contract MyContract {
    function doSomething(uint256 myFunctionParam) public {
        // ... some sort of code goes here
    }
}

contract MyOtherContract {
    function doTheThing(address myContractAddress, uint256 myFunctionParam) public {
        myContractAddress.call(
            abi.encodeWithSignature(
                "doSomething(uint256)",
                myFunctionParam
            )
        );
    }
}
```

Here we're using the [low-level "call" function](https://docs.soliditylang.org/en/v0.8.4/units-and-global-variables.html#members-of-address-types) and one of the [ABI encoding functions built into Solidity](https://docs.soliditylang.org/en/v0.8.4/units-and-global-variables.html#abi-encoding-and-decoding-functions).
Although these two code snippets look a bit different, they're actually functionally identical.

</details>

## Communication basics between layers

At a high level, this process is pretty similar to the same process for two contracts on Ethereum (with a few caveats).
**Communication between L1 and L2 is enabled by two special smart contracts called the "messengers"**.
Each layer has its own messenger contract which serves to abstract away some lower-level communication details, a lot like how HTTP libraries abstract away physical network connections.

We won't get into *too* much detail about these contracts here — the only thing you really need to know about is the `sendMessage` function attached to each messenger:

```solidity
function sendMessage(
    address _target,
    bytes memory _message,
    uint32 _gasLimit
) public;
```

It's the same as that `call` function used for contract messaging within L1 Ethereums.
We have an extra `_gasLimit` field here, but `call` has that too.
This is basically equivalent to:

```solidity
address(_target).call{gas: _gasLimit}(_message);
```

Except, of course, that we're calling a contract on a completely different network.

We're glossing over a lot of the technical details that make this whole thing work under the hood.
Point is, it works.
Want to call a contract on Optimism from a contract on Ethereum?
It's dead simple:

```solidity
// Pretend this is on L2
contract MyOptimisticContract {
    function doSomething(uint256 myFunctionParam) public {
        // ... some sort of code goes here
    }
}

// And pretend this is on L1
contract MyOtherContract {
    function doTheThing(address myOptimisticContractAddress, uint256 myFunctionParam) public {
        ovmL1CrossDomainMessenger.sendMessage(
            myOptimisticContractAddress,
            abi.encodeWithSignature(
                "doSomething(uint256)",
                myFunctionParam
            ),
            1000000 // use whatever gas limit you want
        )
    }
}
```

::: tip Using the messenger contracts
Our messenger contracts, the [`L1CrossDomainMessenger`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L1/messaging/L1CrossDomainMessenger.sol) and [`L2CrossDomainMessenger`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/messaging/L2CrossDomainMessenger.sol), always come pre-deployed to each of our networks.
You can find the exact addresses of these contracts on our various deployments [inside of the Optimism monorepo](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/deployments).
:::

## Communication speed

Unlike calls between contracts on the same blockchain, calls between Ethereum and Optimism are *not* instantaneous.
The exact speed of a cross-chain transaction depends on the direction in which the transaction is sent.

### For Ethereum (L1) to Optimism (L2) transactions

Transactions sent from L1 to L2 take up to approximately 15 minutes on mainnet and 5 minutes on the Optimism Goerli testnet to reach the target L2 contract.
This is because L2 nodes will wait for a certain number of block confirmations on Ethereum before executing an L1 to L2 transaction.

### For Optimism (L2) to Ethereum (L1) transactions

L2 to L1 transactions have to wait two periods:

1. The time until the state root is written to L1.
   You can estimate this time by looking at how often transactions happen to the State Commitment Chain (on both [mainnet](https://etherscan.io/address/0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19) and [goerli](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)).

   As of the Bedrock update, it is necessary to provide a Merkle proof of the message on L1 after the state root is written.
   The fault challenge period starts *after* that proof transaction becomes part of the L1 chain.

1. The [fault challenge period](#understanding-the-challenge-period), which is a few seconds on goerli and seven days on mainnet.
   This waiting period is a core part of the security mechanism designed to keep funds on Optimism secure and cannot be circumvented.
   After this waiting period, any user can "finalize" the transaction by triggering a second transaction on Ethereum that sends the message to the target L1 contract.

## Accessing `msg.sender`

Contracts frequently make use of `msg.sender` to make decisions based on the calling account.
For example, many contracts will use the [Ownable](https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/access/Ownable.sol) pattern to selectively restrict access to certain functions.
Because messages are essentially shuttled between L1 and L2 by the messenger contracts, **the `msg.sender` you'll see when receiving one of these messages will be the messenger contract** corresponding to the layer you're on.

In order to get around this, we added a `xDomainMessageSender` function to each messenger:

```solidity
function xDomainMessageSender() public returns (address);
```

If your contract has been called by one of the messenger contracts, you can use this function to see who's *actually* sending this message.
Here's how you might implement an `onlyOwner` modifier on L2:

```solidity
modifier onlyOwner() {
    require(
        msg.sender == address(ovmL2CrossDomainMessenger)
        && ovmL2CrossDomainMessenger.xDomainMessageSender() == owner
    );
    _;
}
```

## Fees for sending data between L1 and L2

### For L1 ⇒ L2 transactions

The majority of the cost of an L1 to L2 transaction comes from sending a transaction on Ethereum.
You send a transaction to the [`L1CrossDomainMessenger`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L1/messaging/L1CrossDomainMessenger.sol)
contract, which then sends a call to the [`CanonicalTransactionChain`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L1/rollup/CanonicalTransactionChain.sol).
This cost is ultimately determined by gas prices on Ethereum when you're sending the cross-chain transaction.

An L1 to L2 message is expected to trigger contract execution on L2, and that contract execution costs gas.
The first 1.92 million gas on L2 is free.
The vast majority of L1 to L2 calls spend less than the 1.92 million, so nothing further is required.

If you think that your call might spend more than that on L2, you can specify a higher gas limit.
However, to prevent denial of service attacks, we have to impose a cost on gas limits higher than 1.92 million.
This cost is one unit of L1 gas for every 32 units of L2 gas requested beyond the free amount.

For example, if you specify a 2.0 million gas limit in the call to `L1CrossDomainMessenger`, it will be processed this way:

| Amount | Action  |
| ------ | ------- |
| free gas: 1.92 million   | Nothing, this gas is provided on L2 for free |
| excess gas required: 80,000 | 2,500 gas is spent on the L1 portion of the gas fee and in return 80,000 extra gas is provided to the L2 transaction. This is inline with the 1:32 ratio of gas. |

This gas burn happens on L1 when the L1 contract calls `L1CrossDomainMessenger`.
This is before the message has been sent to the L2, and as such there is no way to know how much L2 gas will actually be used.
Therefore, the amount burned is based *only* on the gas limit specified in the L1 call.

For example, if the call above with a gas limit of two million only takes ten thousand gas on L2, the 2,500 gas on L1 is still burned.
There is no refund.

The parameters in the explanation above were 1.92 million and 32 at the time of writing, but they may change in the future.
To see the present values, [go to Etherscan](https://etherscan.io/address/0x5E4e65926BA27467555EB562121fac00D24E9dD2#readContract) and expand `enqueueL2GasPrepaid` for the free L2 gas amount and `l2GasDiscountDivisor` for the exchange rate at which L1 gas is burned for additional L2 gas.


### Fees for L2 ⇒ L1 transactions

Each message from L2 to L1 requires three transactions:

1. An L2 transaction that *initiates* the transaction, which is priced the same as any other transaction made on Optimism.

1. An L1 transaction that *proves* the transaction.
   This transaction can only be submitted after the state root is submitted to L1.
   This transaction is expensive because it includes verifying a [Merkle trie](https://eth.wiki/fundamentals/patricia-tree) inclusion proof.

1. An L1 transaction that *finalizes* the transaction. 
   This transaction can only be submitted after the transaction challenge period (7 days on mainnet) has passed. 

The total cost of an L2 to L1 transaction is therefore the combined cost of the L2 initialization transaction and the two L1 transactions.
The L1 proof and finalization transactions are typically significantly more expensive than the L2 initialization transaction.

## Understanding the challenge period

One of the most important things to understand about L1 ⇔ L2 interaction is that **messages sent from Layer 2 to Layer 1 cannot be relayed for at least one week**.
This means that any messages you send from Layer 2 will only be received on Layer 1 after this one week period has elapsed.
We call this period of time the "challenge period" because it is the time during which a transaction can be challenged with a [fault proof](../../protocol/2-rollup-protocol.md#fault-proofs).

Optimistic Rollups are "optimistic" because they're based around the idea of publishing the *result* of a transaction to Ethereum without actually executing the transaction on Ethereum.
In the "optimistic" case, this transaction result is correct and we can completely avoid the need to perform complicated (and expensive) logic on Ethereum.
Cheap transactions, yay!

However, we still need some way to prevent incorrect transaction results from being published in place of correct ones.
Here's where the "fault proof" comes into play.
Whenever a transaction result is published, it's considered "pending" for a period of time known as the challenge period.
During this period of time, anyone may re-execute the transaction *on Ethereum* in an attempt to demonstrate that the published result was incorrect.

If someone is able prove that a transaction result is faulty, then the result is scrubbed from existence and anyone can publish another result in its place (hopefully the correct one this time, financial punishments make faulty results *very* costly for their publishers).
Once the window for a given transaction result has fully passed without a challenge the result can be considered fully valid (or else someone would've challenged it).

Anyway, the point here is that **you don't want to be making decisions about Layer 2 transaction results from inside a smart contract on Layer 1 until this challenge period has elapsed**.
Otherwise you might be making decisions based on an invalid transaction result.
As a result, L2 ⇒ L1 messages sent using the standard messenger contracts cannot be relayed until they've waited out the full challenge period.

::: tip On the length of the challenge period
We've set the challenge period to be exactly seven days on the Optimism mainnet.
We believe this is a reasonable balance between security and usability, with an emphasis on increased security to start.
We're open to changing the length of the window as long as we feel this can be done without significantly reducing the security of the system.
If you're strongly opinionated about this, we recommend [opening an issue on GitHub](https://github.com/ethereum-optimism/optimism/issues) explaining your position.
We *will* hear you out!
:::
