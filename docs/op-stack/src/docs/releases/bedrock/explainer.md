---
title: Bedrock Explainer 
lang: en-US
meta:
    - name: og:image
      content: https://dev.optimism.io/content/images/size/w2000/2022/12/bedrock-BLUE.jpg
---

![Bedrock](https://dev.optimism.io/content/images/size/w2000/2022/12/bedrock-BLUE.jpg)

::: tip Staying up to date

[Stay up to date on the Superchain and the OP Stack by subscribing to the Optimism newsletter](https://optimism.us6.list-manage.com/subscribe/post?u=9727fa8bec4011400e57cafcb&id=ca91042234&f_id=002a19e3f0).

:::

Bedrock is the name of the first ever official release of the OP Stack codebase, which is a set of free and open-source modular components that work together to power Optimism.

- To understand what is in the Bedrock release, keep reading.
- To develop on Optimism Mainnet, which will upgrade its infrastructure to the Bedrock release, read the docs.
- To contribute to the OP Stack, see the contribution guidelines on the ethereum-optimism monorepo.

## Summary of Improvements

Bedrock improves on its predecessor by reducing transaction fees using optimized batch [compression](#optimized-data-compression) and Ethereum as a data availability layer; shortening delays of including L1 transactions in rollups by handling L1 re-orgs more gracefully; enabling modular proof systems through code re-use; and improving node performance by removing technical debt.

### Lower fees

In addition, Bedrock implements an optimized data [compression](#optimized-data-compression) strategy to minimize data costs. We are currently benchmarking the impact of this change, but we expect it to reduce fees significantly.

Bedrock also removes all gas costs associated with EVM execution when submitting data to L1. This reduces fees by an additional 10% over the previous version of the protocol.

### Shorter deposit times

Bedrock introduces support for L1 re-orgs in the node software, which significantly reduces the amount of time users need to wait for deposits. Earlier versions of the protocol could take up to 10 minutes to confirm deposits. With Bedrock, we expect deposits to confirm within 3 minutes.

### Improved proof modularity

Bedrock abstracts the proof system from the OP Stack so that a rollup may use either a fault proof or validity proof (e.g., a zk-SNARK) to prove correct execution of inputs on the rollup. This abstraction enables systems like [Cannon](https://github.com/ethereum-optimism/cannon) to be used to prove faults in the system.

### Improved node performance

The node software performance has been significantly improved by enabling execution of several transactions in a single rollup "block" as opposed to the prior "one transaction per block" model in the previous version. This allows the cost of merkle trie updates to be amortized across multiple transactions. At current transaction volume, this reduces state growth by approximately 15GB/year.

Node performance is further improved by removing technical debt from the previous version of the protocol. This includes removing the need for a separate "data transport layer" node to index L1, and updating the node software to efficiently query for transaction data from L1.

### Improved Ethereum equivalence

Bedrock was designed from the ground up to be as close to Ethereum as possible. Multiple deviations from Ethereum in the previous version of the protocol have been removed, including:

1. The one-transaction-per-block model.
2. Custom opcodes to get L1 block information.
3. Separate L1/L2 fee fields in the JSON-RPC API.
4. A custom ERC20 representation of ETH balances.

Bedrock also adds support for EIP-1559, chain re-orgs, and other Ethereum features present on L1.

## Design Principles

Bedrock was built to be modular & upgradeable, to reuse existing code from Ethereum, and to be as close to 100% Ethereum-equivalent as possible.

### Modularity

Bedrock makes it easy to swap out different components in the OP Stack codebase and add new capabilities by using well-defined interfaces and versioning schemes. This allows for a flexible architecture that can adapt to future developments in the Ethereum ecosystem.

Examples:
- Separation of [rollup node](#rollup-node) and execution client
- Modular fault proof design

### Code re-use

Bedrock uses existing Ethereum architecture and infrastructure as much as possible. This approach enables the OP Stack to inherit security and "lindy" benefits from the battle-tested codebases used in production on Ethereum Mainnet. You'll find examples of this throughout the design including:

Examples:
- [Minimally modified execution clients](https://op-geth.optimism.io/)
- EVM contracts instead of precompiled client code

### Ethereum equivalence

Bedrock is designed to have maximum compatibility with the existing Ethereum developer experience. A few exceptions exist due to fundamental differences between an L1 and a rollup: an altered fee model, faster block time (2s vs 12s), and a special transaction type for including L1 [deposit](#deposits) transactions.

Examples:
- Fault proof designed to prove faults of minimally modified Ethereum [execution client](#execution-client)
- Code re-use of Ethereum [execution client](#execution-client) for use by nodes in the L2 network and sequencers

## Protocol

Rollups are derived from a data availability source (generally an L1 blockchain like Ethereum). In their most common configuration, rollup protocols derive a  **"canonical L2 chain"** from two primary sources of information:

1. Transaction data posted by Sequencers to the L1 and;
2. [Deposit](#deposits) transactions posted by accounts and contracts to a bridge contract on L1.

The following are the fundamental components of the protocol:

* Deposits are _writes_ to the canonical L2 chain by directly interacting with smart contracts on the L1.
* Withdrawals are _writes_ to the canonical L2 chain that implicitly trigger interactions with contracts and accounts on the L1.
* Batches are _writes_ of data corresponding to batches on the rollup.
* Block derivation is how _reads_ of data on the L1 are interpreted to understand the canonical L2 chain.
* Proof systems define _finality_ of posted output roots on the L1 such that they may be _executed_ upon (e.g., to execute withdrawals).

### Deposits

A **deposit** is a transaction on L1 that is to be included in the rollup. [Deposits](#deposits) are _guaranteed_ by definition to be included in the [canonical L2 chain](#protocol) as a means of preventing censorship or control of the L2.

#### Arbitrary message passing from L1

A **deposited transaction** is the transaction on the rollup that is made as a part of a [deposit](#deposits). With Bedrock, [deposits](#deposits) are fully generalized Ethereum transactions. For example, an account or contract on Ethereum can “deposit” a contract creation.

Bedrock defines a **deposit contract** that is available on the L1: it is a smart contract that L1 accounts and contracts can interact with to write to the L2. [Deposited transactions](#arbitrary-message-passing-from-l1) on the L2 are derived from the values in the event(s) emitted by this [deposit](#deposits) contract, which include expected parameters such as from, to, and data.

For full details, see the [deposit contract](https://github.com/ethereum-optimism/optimism/blob/develop/specs/deposits.md#deposit-contract) section of the protocol specifications.

#### Purchasing guaranteed L2 gas on L1

Bedrock also specifies a gas burn mechanism and a fee market for [deposits](#deposits). The gas that [deposited transactions](#arbitrary-message-passing-from-l1) spend on an L2 is bought on L1 via a gas burn. This gas is purchased on a fee market and there is a hard cap on the amount of gas provided to all [deposits](#deposits) in a single L1 block. This mechanism is used to prevent denial of service attacks that could occur by writing transactions to L2 from L1 that are extremely gas-intensive on L2, but cheap on L1.

The gas provided to [deposited transactions](#arbitrary-message-passing-from-l1) is sometimes called "guaranteed gas." Guaranteed gas is unique in that it is paid for by burning gas on L1 and is therefore not refundable.The total amount of L1 gas that must be burned per unit of guaranteed L2 gas requested depends on the price of L2 gas reported by a EIP-1559 style fee mechanism. Furthermore, users receive a dynamic gas stipend based on the amount of L1 gas spent to compute updates to the fee mechanism.

For a deeper explanation, read the [deposits section](https://github.com/ethereum-optimism/optimism/blob/develop/specs/deposits.md#deposits) of the protocol specifications.

### Withdrawals

**Withdrawals** are cross-domain transactions that are initiated on L2 and finalized by a transaction executed on L1. Notably, withdrawals may be used by an L2 account to call an L1 contract, or to transfer ETH from an L2 account to an L1 account.

Withdrawals are initiated on L2 via a call to the **Message Passer** predeploy contract, which records the important properties of the message in its storage. Withdrawals are finalized on L1 via a call to the [OptimismPortal](https://github.com/ethereum-optimism/optimism/blob/develop/specs/withdrawals.md#the-optimism-portal-contract) contract, which proves the inclusion of this withdrawal message. In this way, withdrawals are different from [deposits](#deposits). Instead of relying on [block derivation](#block-derivation), withdrawal transactions must use smart contracts on L1 for finalization.

#### Two-step withdrawals

Withdrawal proof validation bugs have been the root cause of many of the biggest bridge hacks of the last few years. The Bedrock release introduces an additional step in the withdrawals’ process of prior versions meant to provide an extra layer of defense against these types of bugs. In the two-step withdrawal process, a Merkle proof corresponding to the withdrawal must be submitted 7 days before the withdrawal can be finalized..  This new safety mechanism gives monitoring tools a full  7 days to find and detect invalid withdrawal proofs . If the [withdrawal](#withdrawals) proof is found to be invalid, a contract fix can be deployed before funds are lost. This dramatically reduces the risk of a bridge compromise.

For full details, see the [withdrawals](https://github.com/ethereum-optimism/optimism/blob/develop/specs/withdrawals.md) section of the protocol specification.

### Batches

In Bedrock, a wire format is defined for messaging between the L1 and L2 (i.e., for L2 deriving blocks from L1 and for L2 to write transactions to the L1). This wire format is designed to minimize costs and software complexity for writing to the L1.

#### Optimized data compression

To optimize data compression, lists of L2 transactions called **sequencer batches** are organized into groups of objects called **channels**, each of which have a maximum size that is defined in a [configurable parameter](https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md#channel-format) that will initially be set to ~9.5Mb. These [channels](#optimized-data-compression) are expected to be compressed using a compression function and submitted to the L1.

#### Parallelized batch submission

To parallelize messages from the sequencers that are submitting [compressed](#optimized-data-compression) [channel](#optimized-data-compression) data to the L1, [channels](#optimized-data-compression) are further broken down into **channel frames**, which are chunks of [compressed](#optimized-data-compression) [channel](#optimized-data-compression) data that can fit inside of a single L1 transaction. Given [channel frames](#parallelized-batch-submission) are mutually independent and the ordering is known, the Ethereum transactions sent by the sequencer to the L1 can be sent in parallel which minimizes sequencer software complexity and allows for filling up all available space for data on the L1.

#### Minimized usage of Ethereum gas

Bedrock removes all execution gas used by the L1 system from submitting [channel](#optimized-data-compression) data to the L1 in transactions called **batcher transactions**. All validation logic that was previously happening on smart contracts on the L1 is moved into the [block derivation](#block-derivation) logic.  Instead, [batcher transactions](#minimized-usage-of-ethereum-gas) are sent to a single EOA on Ethereum referred to as the **batch inbox address**.

Batches are still subject to validity checks (i.e. they have to be encoded correctly), and so are individual transactions within the batch (e.g. signatures have to be valid). Invalid [batches](#optimized-data-compression) and invalid individual transactions within an otherwise valid batch are considered to be discarded and irrelevant to the system.

> Note: Ethereum will soon upgrade to include [EIP-4844](https://eip4844.com/), which introduces a separate fee market for writing data and an increased cap of the amount of data the Ethereum protocol is willing to store. This change is expected to further decrease the costs associated with posting data to an L1.

For a deeper explanation, read [the wire format specifications](https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md#overview).

### Block Derivation

In Bedrock, the protocol is designed to guarantee that the timing of [deposits](#deposits) on the L1 is respected with regards to the [block derivation](#block-derivation) of the [canonical L2 chain](#protocol). Doing so is a _pure function_ of data written to the L1 by sequencers, [deposits](#deposits), and L1 block attributes. To accomplish this, the protocol defines strategies for guaranteeing inclusion of deposits, handling L1 and L2 timestamps, and processing sequencing windows in a pipeline to ensure correct ordering.

#### Guaranteed inclusion of deposits

A goal of the [block derivation](#block-derivation) protocol is to define it such that there must be an L2 block every "L2 block time" number of seconds, and that the timestamp of L2 blocks stays in sync with the timestamps of L1 (i.e., to ensure [deposits](#deposits) are included in a logical temporal order).

In Bedrock, the concept of a **sequencing epoch** is introduced: it is a range of L2 blocks derived from a range of L1 blocks. Each [epoch](#guaranteed-inclusion-of-deposits) is identified by an **epoch number**, which is equal to the block number of the first L1 block in the sequencing window. Epochs can vary in size, subject to some constraints.

The batch derivation pipeline treats the timestamps of the L1 blocks associated with [epoch number](#guaranteed-inclusion-of-deposits) as the anchor point for determining the order of transactions on the L2. The protocol guarantees that the first L2 block of an [epoch](#guaranteed-inclusion-of-deposits) never falls behind the timestamp of the L1 block matching the [epoch](#guaranteed-inclusion-of-deposits). The first blocks of an epoch _must_ contain deposits on L1 in order to guarantee that deposits will be processed.

Note that the target configuration for the block time on L2 in the Bedrock release is 2 seconds.

#### Handling L1 and L2 timestamps

Bedrock attempts to address the problem of reconciling the timestamps on L2 with timestamps on L1 present in [deposited transactions](#arbitrary-message-passing-from-l1). It does this by allowing a short window of time for sequencing to liberally apply timestamps on L2 transactions between [epochs](#guaranteed-inclusion-of-deposits).

A **sequencing window** is a sequence of L1 blocks from which an [epoch](#guaranteed-inclusion-of-deposits) can be derived. A [sequencing window](#handling-l1-and-l2-timestamps) whose first L1 block has the number `N` contains [batcher transactions](#minimized-usage-of-ethereum-gas) for [epoch](#guaranteed-inclusion-of-deposits) `N`.

The [sequencing window](#handling-l1-and-l2-timestamps) contains blocks `[N, N + SWS)` where `SWS` is the **sequencer window size**: a fixed rollup-level configuration parameter. This parameter must be at least 2. Increasing it provides more opportunity for sequencers to order L2 transactions with respect to [deposits](#deposits), and lowering it introduces stricter windows of time for sequencers to submit batcher transactions. It is a tradeoff between creating MEV opportunity and increasing software complexity.

A protocol constant called **max sequencer drift** governs the maximum timestamp a block can have within its epoch. Having this drift allows the sequencer to maintain liveness in case of temporary problems connecting to L1. Each L2 block’s timestamp fits within the following range:

```
l1_timestamp <= l2_block.timestamp <= max(l1_timestamp + max_sequencer_drift, l1_timestamp + l2_block_time)
```

#### Block derivation pipeline

The [canonical L2 chain](#protocol) can be processed from scratch by starting with the L2 genesis state, setting the L2 chain inception as the first epoch, and then processing all sequencing windows in order to determine the correct ordering of [sequencer batches](#optimized-data-compression) and [deposits](#deposits) according to the following simplified pipeline:

| **Stage** | **Notes** |
| --- | --- |
| Read from L1 | Epochs are defined by L1 blocks. Contained within an L2 block is data pertaining to [batcher transactions](#minimized-usage-of-ethereum-gas) or [deposits](#deposits) which must be included in the [canonical L2 chain](#protocol) |
| Buffer and decode into [channels](#optimized-data-compression) | The data from L1 blocks contains unordered [channel frames](#parallelized-batch-submission), which must all be collected before reconstructing them into channels. |
| Decompress [channels](#optimized-data-compression) into [batches](#optimized-data-compression) | Since [channels](#optimized-data-compression) are [compressed](#optimized-data-compression) to minimize data fee costs on the L1, they must be decompressed. |
| Queue [batches](#optimized-data-compression) into sequential order | With the latest information from L1, [batches](#optimized-data-compression) can be validated and processed sequentially. There are some nuances to what the correct ordering is in relation to [epochs](#guaranteed-inclusion-of-deposits) and timestamps from L2, see the full specification [here](https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md#batch-queue). |
| Interpret as L2 blocks | At this point, the correct ordering of [batches](#optimized-data-compression) can be determined.<br><br>Following this, the [execution client](#execution-client) can interpret them into L2 blocks. For implementation details pertaining to [execution clients](#execution-client), see the [engine queue](https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md#engine-queue) section of the protocol specifications. |

### Fault Proofs

After a sequencer processes one or more L2 blocks, the outputs computed from executing transactions in those blocks will need to be written with L1 for trustless execution of L2-to-L1 messaging, such as [withdrawals](#withdrawals).

In Bedrock, outputs are hashed in a tree-structured form which minimizes the cost of proving any piece of data captured by the outputs. Proposers periodically submit **output roots** that are Merkle roots of the entire [canonical L2 chain](#protocol) to the L1.

Future upgrades of the OP Stack codebase should include a specification for a variation of a fault proof with bonding included to create incentives for proposers to propose correct output roots.

For full details, read the [L2 Output Root Proposals section](https://github.com/ethereum-optimism/optimism/blob/develop/specs/proposals.md#l2-output-root-proposals-specification) of the protocol specifications.

## Implementation

With Bedrock, the OP Stack codebase leans heavily into the technical separation of concerns specified by Ethereum by mirroring the separation between the Ethereum execution layer and consensus layer. Bedrock introduces separation of execution client and rollup node in this same way.

### Execution Client

An **execution client** is the system that sequencers and other kinds of node operators run to determine the state of the [canonical L2 chain](#protocol). It also performs other functions such as processing inbound transactions and communicating them peer-to-peer, and handling the state of the system to process queries against it.

With Bedrock, the OP Stack is designed to reuse [Ethereum’s own execution client specifications](https://github.com/ethereum/execution-specs) and its many implementations. In this release, Bedrock has demonstrated an extremely limited modification of go-ethereum, the most popular Ethereum client written in Go, to a [diff of less than 2000 lines of code](https://op-geth.optimism.io/).

There are two fundamental reasons for having any diff at all: handling deposited transactions, and charging transaction fees.

#### Handling deposited transactions

To represent [deposited transactions](#arbitrary-message-passing-from-l1) in the rollup, there is an additional transaction type introduced. The [execution client](#execution-client) implements this [new transaction type](https://github.com/ethereum-optimism/optimism/blob/develop/specs/%5Bdeposits%5D(#deposits).md%23the-deposited-transaction-type) according to the [EIP-2718 typed transactions](https://eips.ethereum.org/EIPS/eip-2718) standard.

#### Charging transaction fees

Rollups also fundamentally have two kinds of fees associated with transactions:

**Sequencer fees**

The cost of operating a sequencer is computed using the same gas table as Ethereum and with the same [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559) algorithm. These fees go to the protocol for operating sequencers and fluctuate based on the congestion of the network.

**Data availability fees**

Data availability costs are associated with writing [batcher transactions](#minimized-usage-of-ethereum-gas) to the L1. These fees are intended to cover the cost that sequencers need to pay to submit [batcher transactions](#minimized-usage-of-ethereum-gas) to the L1.

In Bedrock, the data availability portion of the fee is determined based on information in a system contract on the rollup called a [GasPriceOracle](https://github.com/ethereum-optimism/optimism/blob/develop/specs/predeploys.md#gaspriceoracle). This contract is updated during [block derivation](#block-derivation) from the gas pricing information retrieved from the L1 block attributes that get inserted at the beginning of every [epoch](#guaranteed-inclusion-of-deposits).

Bedrock specifies that both of these fees are added up into a single `gasPrice` field when using the JSON-RPC.

### Rollup Node

Unlike Ethereum, Bedrock does not have proof-of-stake consensus. Instead, the consensus of the [canonical L2 chain](#protocol) is defined by [block derivation](#block-derivation). An [execution client](#execution-client) of the OP Stack communicates to a new component that implements [block derivation](#block-derivation) called a **rollup node**. This node communicates to the [execution client](#execution-client) using the exact same [Engine API](https://github.com/ethereum/execution-apis/tree/main/src/engine) that Ethereum uses.

The [rollup node](#rollup-node) is a stateless component responsible for deriving the state of the system by reading data and [deposits](#deposits) on the L1. In Bedrock, a [rollup node](#rollup-node) can either be used to sequence incoming transactions from users or other [rollup nodes](#rollup-node) or to verify confirmed transactions posted on the L1 by singularly relying on the L1.

The multiple uses of a rollup node are outlined below.

#### Verifying the canonical L2 chain

The simplest mode of running a [rollup node](#rollup-node) is to only follow the [canonical L2 chain](#protocol). In this mode, the [rollup node](#rollup-node) has no peers and is strictly used to read data from the L1 and to interpret it according to [block derivation](#block-derivation) protocol rules.

One purpose of this kind of node is to verify that any output roots shared by other nodes or posted on the L1 are correct according to protocol definition. Additionally, proposers intending to submit output roots to the L1 themselves can generate the output roots they need using the [optimism_outputAtBlock](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md#l2-output-rpc-method) of the node which returns a 32-byte hash corresponding to the L2 output root.

For this purpose, nodes should only need to follow the finalized head. The term ["finalized"](https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/#finality) refers to the Ethereum proof-of-stake consensus (i.e. canonical and practically irreversible) — the finalized L2 head is the head of the [canonical L2 chain](#protocol) that is derived only from finalized L1 blocks.

#### Participating in the L2 network

The most common way to use a [rollup node](#rollup-node) is to participate in a network of other [rollup nodes](#rollup-node) tracking the progression and state of an L2. In this mode, a [rollup node](#rollup-node) is both reading the data and [deposits](#deposits) it observes from the L1 and interpreting it as blocks and accepting inbound transactions from users and peers in a network of other [rollup nodes](#rollup-node).

Nodes participating in the network may make use of the safe and unsafe heads of the L2 they're syncing.

- The **safe L2 head** represents the rollup that can be constructed where every block up to and including the head can be fully derived from the reference L1 chain, before L1 has necessarily finalized (i.e., a re-org may occur on L1 still).
- The **unsafe L2 head** includes [unsafe blocks](https://github.com/ethereum-optimism/optimism/blob/develop/specs/glossary.md#unsafe-l2-block) that have not yet been derived from L1. These blocks either come from operating the [rollup node](#rollup-node) as a sequencer or from [unsafe sync](https://github.com/ethereum-optimism/optimism/blob/develop/specs/glossary.md#unsafe-sync) with the sequencer. This is also known as the "latest" head. The safe L2 head is always chosen over the unsafe L2 head in cases of disagreements. When disagreements occur, the unsafe portion of the chain will reorg.

For most purposes, nodes in the L2 network will refer to the unsafe L2 head for end-user applications.

#### Sequencing transactions

The third way to use a [rollup node](#rollup-node) is to sequence transactions. In this mode, a [rollup node](#rollup-node) will _create_ new blocks on top of the unsafe L2 head. Currently, there is only one sequencer per OP Stack network.

The sequencer is also responsible for posting batches to L1 for other nodes in the network to sync from.

### Batcher

The role of a sequencer is to produce [batches](#batches). To do this, a sequencer can run [rollup nodes](#rollup-node) and have separate processes which perform [batching](#batches) by reading from a trusted [rollup node](#rollup-node) they run. This warrants an additional component of the OP Stack called a [batcher](https://github.com/ethereum-optimism/optimism/blob/develop/specs/glossary.md#batcher) that reads transaction data from a [rollup node](#rollup-node) and interprets it into [batcher transactions](#minimized-usage-of-ethereum-gas) to be written to the L1. The batcher component is responsible for reading the unsafe L2 head of a [rollup node](#rollup-node) run by a sequencer, creating batcher transactions, and writing them to the L1.

### Standard Bridge Contracts

Bedrock also includes a pair of bridge contracts used for the most common kinds of [deposits](#deposits) called the [standard bridges](https://github.com/ethereum-optimism/optimism/blob/develop/specs/bridges.md#standard-bridges). These contracts wrap the [deposit](#deposits) and [withdrawal](#withdrawals) contracts to provide simple interfaces for [depositing](#deposits) and [withdrawing](#withdrawals) ETH and ERC-20 tokens.

These bridges are designed to involve a native token on one side of the bridge, and a wrapped token on the other side that can manage minting and burning. Bridging a native token involves locking the native token in a contract and then minting an equivalent amount of mintable token on the other side of the bridge.

For full details, see the [standard bridge](https://github.com/ethereum-optimism/optimism/blob/develop/specs/bridges.md#standard-bridges) section of the protocol specifications.

### Cannon

Although fault proof construction and verification is implemented in the [Cannon](https://github.com/ethereum-optimism/cannon) project, the fault proof game specification and integration of an output root challenger into the rollup node are part of later specification milestones.

## Further Reading

### Protocol Specification

The protocol specification defines the technical details of the OP Stack codebase. It is the most up-to-date source of truth for the inner workings of the protocol. The protocol specification is located in the ethereum-optimism [monorepo](https://github.com/ethereum-optimism/optimism/blob/develop/specs/README.md).

### Bedrock Differences

For a deep dive into the differences between Bedrock and previous versions of the protocol, see the [How is Bedrock Different?](https://community.optimism.io/docs/developers/bedrock/differences/) page.
