# Optimistic VM \(OVM\)

## What is the OVM?

The Optimistic Virtual Machine \(OVM\) is a scalable form of the EVM. It lives at the core of [Optimistic Rollup](https://medium.com/plasma-group/ethereum-smart-contracts-in-l2-optimistic-rollup-2c1cef2ec537) \(ORU\) fullnodes and is able to execute Ethereum smart contracts at scale. It makes use of **optimistic execution**, allowing the blockchain to only evaluate smart contracts when there is fraud. This enables OVM computation to scale in the number of disputes as opposed to the number of transactions.

### For Developers...

The OVM is an EVM-based VM which supports optimistically executing EVM smart contracts on a layer 1 blockchain like Ethereum. It is structured in such a way that it is possible to verify individual steps of it's computation on Ethereum mainnet. This allows the mainnet to enforce validity of state roots with fraud proofs in the layer 2 Optimistic Rollup chain.

Each computational step is called a transition. These transitions can be evaluated off-chain as well as on-chain in an OVM [sandbox](https://en.wikipedia.org/wiki/Sandbox_%28computer_security%29) to ensure their validity. Through techniques similar to a technique called [stateless clients](https://ethresear.ch/t/the-stateless-client-concept/172) originally developed for Eth2, each transition's validity may be evaluated efficiently & in isolation.

### Why not just use the EVM?

Unfortunately the EVM is not structured in a way that allows you to spawn sandboxed subprocesses. Without sandboxing, we are unable to verify the validity of ORU transitions, and therefore are unable to build an ORU compatible with the EVM.

Thankfully, the EVM is turing complete & therefore flexible enough for us to embed this sandbox functionality directly inside of it. By embedding the OVM inside of the EVM, we're able to take advantage of all of the great work on the EVM while adding this critical feature that we need for ORU.

