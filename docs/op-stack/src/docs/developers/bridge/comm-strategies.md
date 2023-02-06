---
title: Communication Strategies
lang: en-US
---

Dapps' inter-layer communication strategies are based on trade-offs between several parameters:

- Speed
- Cost
- Trust assumptions

An issue related to both speed and decentralization is the L2 state.
This state is vulnerable to fault challenges until the fault challenge period (currently one week) passes.
If you want to do something that relies on the L2 state prior to that point, you should [run a replica](../build/run-a-node.md) yourself to make sure the state you use is correct.



## Fully centralized

If your dapp has a centralized always on server, the easiest solution is to just have two providers, one connected to Ethereum (L1) and the other to Optimism (L2).

| Parameter         | Evaluation |
| - | - |
| Speed             | Fastest
| Cost              | Cheapest
| Trust assumption  | Centralized


### Using the client (please don't)

The client (typically a browser with a wallet) can also connect to both Ethereum and Optimism, but it isn't a great mechanism for inter-layer communication.
You know what the code in the server is, because you wrote it.
You know what the code in the client is *supposed to be*, but it is possible for users to run a different client that pretends to be the legitimate one. 
The only time that you can trust the client for inter-layer communication is when it is in the best interest of the user running the client not to lie.
And even then, you shouldn't because a hacker can cause a user to run malware.


## Fully decentralized

If you want to keep the same decentralization promises as Optimism and Ethereum, you can [rely on our messaging infrastructure](messaging.md).
You are already trusting Optimism to run the chain, and the messaging infrastructure goes through the same development process.

### Messages from L1 to L2

If you want L1 code to either tell L2 code to do something, or update L2 code with some information, you just need to [issue a single L1 transaction](messaging.md#for-ethereum-l1-to-optimism-l2-transactions).

| Parameter         | Evaluation |
| - | - |
| Speed             | ~15 minutes
| Cost              | Cheapish (requires an L1 transaction)
| Trust assumption  | Same as using Optimism

### Messages from L2 to L1

Sending messages from L2 to L1 is [a lot harder](messaging.md#for-optimism-l2-to-ethereum-l1-transactions). 
It requires two transactions:

1. An initiating transaction on L2, which is pretty cheap.
1. Once the fault challenge period passes, a claiming transaction on L1, which includes [a merkle proof](https://medium.com/crypto-0-nite/merkle-proofs-explained-6dd429623dc5). 
   This transaction is expensive because merkle proof verification is expensive.

| Parameter         | Evaluation |
| - | - |
| Speed             | >7 days 
| Cost              | Expensive
| Trust Assumption  | Almost as good as using Optimism, however someone needs to initiate the claim transaction on L1


## Incentivized communication

You can also use incentives, for example using a mechanism such as [UMA's](../../useful-tools/oracles/#universal-market-access-uma).
This is similar to the way optimistic rollups work - honest relays get paid, dishonest ones get slashed.
However,  this mechanism is only truly decentralized if there are enough relayers to make sure there will always be an honest one.
Otherwise, it's similar to centralized communications, just with a few extra relayers that can take over.