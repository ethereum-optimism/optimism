# Glossary

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [General Terms](#general-terms)
  - [Layer 1 (L1)](#layer-1-l1)
  - [Layer 2 (L2)](#layer-2-l2)
  - [Block](#block)
  - [EOA](#eoa)
  - [Merkle Patricia Trie](#merkle-patricia-trie)
  - [Chain Re-Organization](#chain-re-organization)
  - [Predeployed Contract ("Predeploy")](#predeployed-contract-predeploy)
  - [Receipt](#receipt)
  - [Transaction Type](#transaction-type)
  - [Fork Choice Rule](#fork-choice-rule)
  - [Priority Gas Auction](#priority-gas-auction)
- [Sequencing](#sequencing)
  - [Sequencer](#sequencer)
  - [Sequencing Window](#sequencing-window)
  - [Sequencing Epoch](#sequencing-epoch)
  - [L1 Origin](#l1-origin)
- [Deposits](#deposits)
  - [Deposited Transaction](#deposited-transaction)
  - [L1 Attributes Deposited Transaction](#l1-attributes-deposited-transaction)
  - [User-Deposited Transaction](#user-deposited-transaction)
  - [Depositing Call](#depositing-call)
  - [Depositing Transaction](#depositing-transaction)
  - [Depositor](#depositor)
  - [Deposited Transaction Type](#deposited-transaction-type)
  - [Deposit Contract](#deposit-contract)
- [Withdrawals](#withdrawals)
  - [Relayer](#relayer)
  - [Finalization Period](#finalization-period)
- [Batch Submission](#batch-submission)
  - [Data Availability](#data-availability)
  - [Data Availability Provider](#data-availability-provider)
  - [Sequencer Batch](#sequencer-batch)
  - [Channel](#channel)
  - [Channel Frame](#channel-frame)
  - [Batcher](#batcher)
  - [Batcher Transaction](#batcher-transaction)
  - [Channel Timeout](#channel-timeout)
- [L2 Chain Derivation](#l2-chain-derivation)
  - [L2 Derivation Inputs](#l2-derivation-inputs)
  - [System Configuration](#system-configuration)
  - [Payload Attributes](#payload-attributes)
  - [L2 Genesis Block](#l2-genesis-block)
  - [L2 Chain Inception](#l2-chain-inception)
  - [Safe L2 Block](#safe-l2-block)
  - [Safe L2 Head](#safe-l2-head)
  - [Unsafe L2 Block](#unsafe-l2-block)
  - [Unsafe L2 Head](#unsafe-l2-head)
  - [Unsafe Block Consolidation](#unsafe-block-consolidation)
  - [Finalized L2 Head](#finalized-l2-head)
- [Other L2 Chain Concepts](#other-l2-chain-concepts)
  - [Address Aliasing](#address-aliasing)
  - [Rollup Node](#rollup-node)
  - [Rollup Driver](#rollup-driver)
  - [L1 Attributes Predeployed Contract](#l1-attributes-predeployed-contract)
  - [L2 Output Root](#l2-output-root)
  - [L2 Output Oracle Contract](#l2-output-oracle-contract)
  - [Validator](#validator)
  - [Fault Proof](#fault-proof)
  - [Time Slot](#time-slot)
  - [Block Time](#block-time)
  - [Unsafe Sync](#unsafe-sync)
- [Execution Engine Concepts](#execution-engine-concepts)
  - [Execution Engine](#execution-engine)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

------------------------------------------------------------------------------------------------------------------------

# General Terms

## Layer 1 (L1)

[L1]: glossary.md#layer-1-L1

Refers the Ethereum blockchain, used in contrast to [layer 2][L2], which refers to Optimism.

## Layer 2 (L2)

[L2]: glossary.md#layer-2-L2

Refers to the Optimism blockchain (specified in this repository), used in contrast to [layer 1][L1], which
refers to the Ethereum blockchain.

## Block

[block]: glossary.md#block

Can refer to an [L1] block, or to an [L2] block, which are structured similarly.

A block is a sequential list of transactions, along with a couple of properties stored in the *header* of the block. A
description of these properties can be found in code comments [here][nano-header], or in the [Ethereum yellow paper
(pdf)][yellow], section 4.3.

It is useful to distinguish between input block properties, which are known before executing the transactions in the
block, and output block properties, which are derived after executing the block's transactions. These include various
[Merkle Patricia Trie roots][mpt] that notably commit to the L2 state and to the log events emitted during execution.

## EOA

[EOA]: glossary.md#EOA

"Externally Owned Account", an Ethereum term to designate addresses operated by users, as opposed to contract addresses.

## Merkle Patricia Trie

[mpt]: glossary.md#merkle-patricia-trie

A [Merkle Patricia Trie (MPT)][mpt-details] is a sparse trie, which is a tree-like structure that maps keys to values.
The root hash of a MPT is a commitment to the contents of the tree, which allows a
proof to be constructed for any key-value mapping encoded in the tree. Such a proof is called a Merkle proof, and can be
verified against the Merkle root.

## Chain Re-Organization

[reorg]: glossary.md#chain-re-organization

A re-organization, or re-org for short, is whenever the head of a blockchain (its last block) changes (as dictated by
the [fork choice rule][fork-choice-rule]) to a block that is not a child of the previous head.

L1 re-orgs can happen because of network conditions or attacks. L2 re-orgs are a consequence of L1 re-orgs, mediated via
[L2 chain derivation][derivation].

## Predeployed Contract ("Predeploy")

[predeploy]: glossary.md#predeployed-contract-predeploy

A contract placed in the L2 genesis state (i.e. at the start of the chain).

All predeploy contracts are specified in the [predeploys specification][./predeploys.md].

## Receipt

[receipt]: glossary.md#receipt

A receipt is an output generated by a transaction, comprising a status code, the amount of gas used, a list of log
entries, and a [bloom filter] indexing these entries. Log entries are most notably used to encode [Solidity events].

Receipts are not stored in blocks, but blocks store a [Merkle Patricia Trie root][mpt] for a tree containing the receipt
for every transaction in the block.

Receipts are specified in the [yellow paper (pdf)][yellow] section 4.3.1.

## Transaction Type

[transaction-type]: glossary.md#transaction-type

Ethereum provides a mechanism (as described in [EIP-2718]) for defining different transaction types.
Different transaction types can contain different payloads, and be handled differently by the protocol.

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

## Fork Choice Rule

[fork-choice-rule]: glossary.md#fork-choice-rule

The fork choice rule is the rule used to determined which block is to be considered as the head of a blockchain. On L1,
this is determined by the proof of stake rules.

L2 also has a fork choice rule, although the rules vary depending on whether we want the [safe L2 head][safe-l2-head],
the [unsafe L2 head][unsafe-l2-head] or the [finalized L2 head][finalized-l2-head].

## Priority Gas Auction

Transactions in ethereum are ordered by the price that the transaction pays to the miner. Priority Gas Auctions
(PGAs) occur when multiple parties are competing to be the first transaction in a block. Each party continuously
updates the gas price of their transaction. PGAs occur when there is value in submitting a transaction before other
parties (like being the first deposit or submitting a deposit before there is not more guaranteed gas remaining).
PGAs tend to have negative externalities on the network due to a large amount of transactions being submitted in a
very short amount of time.

------------------------------------------------------------------------------------------------------------------------

# Sequencing

[sequencing]: glossary.md#sequencing

Transactions in the rollup can be included in two ways:

- Through a [deposited transaction](#deposited-transaction), enforced by the system
- Through a regular transaction, embedded in a [sequencer batch](#sequencer-batch)

Submitting transactions for inclusion in a batch saves costs by reducing overhead, and enables the sequencer to
pre-confirm the transactions before the L1 confirms the data.

## Sequencer

[sequencer]: glossary.md#sequencer

A sequencer is either a [rollup node][rollup-node] ran in sequencer mode, or the operator of this rollup node.

The sequencer is a priviledged actor, which receives L2 transactions from L2 users, creates L2 blocks using them, which
it then submits to [data availability provider][avail-provider] (via a [batcher]). It also submits [output
roots][l2-output] to L1.

## Sequencing Window

[sequencing-window]: glossary.md#sequencing-window

A sequencing window is a range of L1 blocks from which a [sequencing epoch][sequencing-epoch] can be derived.

A sequencing window whose first L1 block has number `N` contains [batcher transactions][batcher-transaction] for epoch
`N`. The window contains blocks `[N, N + SWS)` where `SWS` is the sequencer window size.

> **TODO** specify sequencer window size

Additionally, the first block in the window defines the [depositing transactions][depositing-tx] which determine the
[deposits] to be included in the first L2 block of the epoch.

## Sequencing Epoch

[sequencing-epoch]: glossary.md#sequencing-epoch

A sequencing epoch is sequential range of L2 blocks derived from a [sequencing window](#sequencing-window) of L1 blocks.

Each epoch is identified by an epoch number, which is equal to the block number of the first L1 block in the
sequencing window.

Epochs can have variable size, subject to some constraints. See the [L2 chain derivation specification][derivation-spec]
for more details.

## L1 Origin

[l1-origin]: glossary.md#l1-origin

The L1 origin of an L2 block is the L1 block corresponding to its [sequencing epoch][sequencing-epoch].

------------------------------------------------------------------------------------------------------------------------

# Deposits

[deposits]: glossary.md#deposits

In general, a deposit is an L2 transaction derived from an L1 block (by the [rollup driver]).

While transaction deposits are notably (but not only) used to "deposit" (bridge) ETH and tokens to L2, the word
*deposit* should be understood as "a transaction *deposited* to L2 from L1".

This term *deposit* is somewhat ambiguous as these "transactions" exist at multiple levels. This section disambiguates
all deposit-related terms.

Notably, a *deposit* can refer to:

- A [deposited transaction][deposited] (on L2) that is part of a [deposit block][deposit-block].
- A [depositing call][depositing-call] that causes a [deposited transaction][deposited] to be derived.
- The event/log data generated by the [depositing call][depositing-call], which is what the [rollup driver] reads to
  derive the [deposited transaction][deposited].

We sometimes also talk about *user deposit* which is a similar term that explicitly excludes [L1 attributes deposited
transactions][l1-attr-deposit].

Deposits are specified in the [deposits specification][deposits-spec].

[deposits-spec]: deposits.md

## Deposited Transaction

[deposited]: glossary.md#deposited-transaction

A *deposited transaction* is a L2 transaction that was derived from L1 and included in a L2 block.

There are two kinds of deposited transactions:

- [L1 attributes deposited transaction][l1-attr-deposit], which submits the L1 block's attributes to the [L1 Attributes
  Predeployed Contract][l1-attr-predeploy].
- [User-deposited transactions][user-deposited], which are transactions derived from an L1 call to the [deposit
  contract][deposit-contract].

[deposits-spec]: deposits.md

## L1 Attributes Deposited Transaction

[l1-attr-deposit]: glossary.md#l1-attributes-deposited-transaction

An *L1 attributes deposited transaction* is [deposited transaction][deposited] that is used to register the L1 block
attributes (number, timestamp, ...) on L2 via a call to the [L1 Attributes Predeployed Contract][l1-attr-predeploy].
That contract can then be used to read the attributes of the L1 block corresponding to the current L2 block.

L1 attributes deposited transactions are specified in the [L1 Attributes Deposit][l1-attributes-tx-spec] section of the
deposits specification.

[l1-attributes-tx-spec]: deposits.md#l1-attributes-deposited-transaction

## User-Deposited Transaction

[user-deposited]: glossary.md#user-deposited-transaction

A *user-deposited transaction* is a [deposited transaction][deposited] which is derived from an L1 call to the [deposit
  contract][deposit-contract] (a [depositing call][depositing-call]).

User-deposited transactions are specified in the [Transaction Deposits][tx-deposits-spec] section of the deposits
specification.

[tx-deposits-spec]: deposits.md#user-deposited-transactions

## Depositing Call

[depositing-call]: glossary.md#depositing-call

A *depositing call* is an L1 call to the [deposit contract][deposit-contract], which will be derived to a
[user-deposited transaction][user-deposited] by the [rollup driver].

This call specifies all the data (destination, value, calldata, ...) for the deposited transaction.

## Depositing Transaction

[depositing-tx]: glossary.md#depositing-transaction

A *depositing transaction* is an L1 transaction that makes one or more [depositing calls][depositing-call].

## Depositor

[depositor]: glossary.md#depositor

The *depositor* is the L1 account (contract or [EOA]) that makes (is the `msg.sender` of) the [depositing
call][depositing-call]. The *depositor* is **NOT** the originator of the depositing transaction (i.e. `tx.origin`).

## Deposited Transaction Type

[deposit-tx-type]: glossary.md#deposited-transaction-type

The *deposited transaction type* is an [EIP-2718] [transaction type][transaction-type], which specifies the input fields
and correct handling of a [deposited transaction][deposited].

See the [corresponding section][spec-deposit-tx-type] of the deposits spec for more information.

[spec-deposit-tx-type]: deposits.md#the-deposited-transaction-type

## Deposit Contract

[deposit-contract]: glossary.md#deposit-contract

The *deposit contract* is an [L1] contract to which [EOAs][EOA] and contracts may send [deposits]. The deposits are
emitted as log records (in Solidity, these are called *events*) for consumption by [rollup nodes][rollup-node].

Advanced note: the deposits are not stored in calldata because they can be sent by contracts, in which case the calldata
is part of the *internal* execution between contracts, and this intermediate calldata is not captured in one of the
[Merkle Patricia Trie roots][mpt] included in the L1 block.

cf. [Deposits Specification](deposits.md)

------------------------------------------------------------------------------------------------------------------------

# Withdrawals

> **TODO** expand this whole section to be clearer

[withdrawals]: glossary.md#withdrawals

In general, a withdrawal is a transaction sent from L2 to L1 that may transfer data and/or value.

The term *withdrawal* is somewhat ambiguous as these "transactions" exist at multiple levels. In order to differentiate
 between the L1 and L2 components of a withdrawal we introduce the following terms:

- A *withdrawal initiating transaction* refers specifically to a transaction on L2 sent to the Withdrawals predeploy.
- A *withdrawal finalizing transaction* refers specifically to an L1 transaction which finalizes and relays the
  withdrawal.

## Relayer

[relayer]: glossary.md#withdrawals

An EOA on L1 which finalizes a withdrawal by submitting the data necessary to verify its inclusion on L2.

## Finalization Period

[finalization-period]: glossary.md#finalization-period

The finalization period — sometimes also called *withdrawal delay* — is the minimum amount of time (in seconds) that
must elapse before a [withdrawal][withrawals] can be finalized.

The finalization period is necessary to afford sufficient time for [validators][validator] to make a [fault
proof][fault-proof].

> **TODO** specify current value for finalization period

------------------------------------------------------------------------------------------------------------------------

# Batch Submission

[batch-submission]: glossary.md#batch-submission

## Data Availability

 [data-availability]: glossary.md#data-availability

Data availability is the guarantee that some data will be "available" (i.e. *retrievable*) during a reasonably long time
window. In Optimism's case, the data in question are [sequencer batches][sequencer-batch] that [validators][validator]
needs in order to verify the sequencer's work and validate the L2 chain.

The [finalization period][finalization-period] should be taken as the lower bound on the availability window, since
that is when data availability is the most crucial, as it is needed to perform a [fault proof][fault-proof].

"Availability" **does not** mean guaranteed long-term storage of the data.

## Data Availability Provider

[avail-provider]: glossary.md#data-availability-provider

A data availability provider is a service that can be used to make data available. See the [Data
Availability][data-availability] for more information on what this means.

Ideally, a good data availability provider provides strong *verifiable* guarantees of data availability

Currently, the only supported data availability provider is Ethereum call data. [Ethereum data blobs][eip4844] will be
supported when they get deployed on Ethereum.

## Sequencer Batch

[sequencer-batch]: glossary.md#sequencer-batch

A sequencer batch is list of L2 transactions (that were submitted to a sequencer) tagged with an [epoch
number](#sequencing-epoch) and an L2 block timestamp (which can trivially be converted to a block number, given our
block time is constant).

Sequencer batches are part of the [L2 derivation inputs][deriv-inputs]. Each batch represents the inputs needed to build
**one** L2 block (given the existing L2 chain state) — except for the first block of each epoch, which also needs
information about deposits (cf. the section on [L2 derivation inputs][deriv-inputs]).

## Channel

[channel]: glossary.md#channel

A channel is a sequence of [sequencer batches][sequencer-batch] (for sequential blocks) compressed together. The reason
to group multiple batches together is simply to obtain a better compression rate, hence reducing data availability
costs.

A channel can be split in [frames][channel-frame] in order to be transmitted via [batcher
transactions][batcher-transaction]. The reason to split a channel into frames is that a channel might be too large to
include in a single batcher transaction.

A channel is uniquely identified by its timestamp (UNIX time at which the channel was created) and a random value. See
the [Frame Format][frame-format] section of the L2 Chain Derivation specification for more information.

[frame-format]: derivation.md#frame-format

On the side of the [rollup node][rollup-node] (which is the consumer of channels), a channel is considered to be
*opened* if its final frame (explicitly marked as such) has not been read, or closed otherwise.

## Channel Frame

[channel-frame]: glossary.md#channel-frame

A channel frame is a chunk of data belonging to a [channel]. [Batcher transactions][batcher-transaction] carry one or
multiple frames. The reason to split a channel into frames is that a channel might too large to include in a single
batcher transaction.

## Batcher

[batcher]: glossary.md#batcher

A batcher is a software component (independant program) that is responsible to make channels available on a data
availability provider. The batcher communicates with the rollup node in order to retrieve the channels. The channels are
then made available using [batcher transactions][batcher-transaction].

> **TODO** In the future, we might want to make the batcher responsible for constructing the channels, letting it only
> query the rollup node for L2 block inputs.

## Batcher Transaction

[batcher-transaction]: glossary.md#batcher-transaction

A batcher transaction is a transaction submitted by a [batcher] to a data availability provider, in order to make
channels available. These transactions carry one or more full frames, which may belong to different channels. A
channel's frame may be split between multiple batcher transactions.

When submitted to Ethereum calldata, the batcher transaction's receiver must be the sequencer inbox address. The
transaction must also be signed by a recognized batch submitter account.

> **TODO** specify where these recognized batch submitter accounts are stored

## Channel Timeout

[channel-timeout]: glossary.md#channel-timeout

The channel timeout is a duration (in L1 blocks) during which [channel frames][channel-frame] may land on L1 within
[batcher transactions][batcher-transaction].

The acceptable time range for the frames of a [channel][channel] is `[channel_id.timestamp, channel_id.timestamp +
CHANNEL_TIMEOUT]`. The acceptable L1 block range for these frames are any L1 block whose timestamp falls inside this
time range. (Note that `channel_id.timetamp` must be lower than the L1 block timestamp of any L1 block in which frames
of the channel are seen, or else these frames are ignored.)

The purpose of channel timeouts is dual:

- Avoid keeping old unclosed channel data around forever (an unclosed channel is a channel whose final frame was not
  sent).
- Bound the number of L1 blocks we have to look back in order to decode [sequencer batches][sequencer-batch] from
  channels. This is particularly relevant during L1 re-orgs, see the [Resetting Channel Buffering][reset-channel-buffer]
  section of the L2 Chain Derivation specifiction for more information.

[reset-channel-buffer]: derivation.md#resetting-channel-buffering

> **TODO** specify `CHANNEL_TIMEOUT`

------------------------------------------------------------------------------------------------------------------------

# L2 Chain Derivation

[derivation]: glossary.md#L2-chain-derivation

L2 chain derivation is a process that reads [L2 derivation inputs][deriv-inputs] from L1 in order to derive the L2
chain.

See the [L2 chain derivation specification][derivation-spec] for more details.

## L2 Derivation Inputs

[deriv-inputs]: glossary.md#l2-chain-derivation-inputs

This term refers to data that is found in L1 blocks and is read by the [rollup node][rollup-node] to construct [payload
attributes][payload-attr].

L2 derivation inputs include:

- L1 block attributes
  - block number
  - timestamp
  - basefee
- [deposits] (as log data)
- [sequencer batches][sequencer-batch] (as transaction data)
- [System configuration][system-config] updates (as log data)

## System Configuration

[system-config]: glossary.md#system-configuration

This term refers to the collection of dynamically configurable rollup parameters maintained
by the [`SystemConfig`](./system_config.md) contract on L1 and read by the L2 [derivation] process.
These parameters enable keys to be rotated regularly and external cost parameters to be adjusted
without the network upgrade overhead of a hardfork.

## Payload Attributes

[payload-attr]: glossary.md#payload-attributes

This term refers to an object that can be derived from [L2 chain derivation inputs][deriv-inputs] found on L1, which are
then passed to the [execution engine][execution-engine] to construct L2 blocks.

The payload attributes object essentially encodes [a block without output properties][block].

Payload attributes are originally specified in the [Ethereum Engine API specification][engine-api], which we expand in
the [Execution Engine Specification](exec-engine.md).

See also the [Building The Payload Attributes][building-payload-attr] section of the rollup node specification.

[building-payload-attr]: rollup-node.md#building-the-payload-attributes

## L2 Genesis Block

[l2-genesis]: glossary.md#l2-genesis-block

The L2 genesis block is the first block of the L2 chain in its current version.

The state of the L2 genesis block comprises:

- State inherited from the previous version of the L2 chain.
  - This state was possibly modified by "state surgeries". For instance, the migration to Bedrock entailed changes on
    how native ETH balances were stored in the storage trie.
- [Predeployed contracts][predeploy]

The timestap of the L2 genesis block must be a multiple of the [block time][block-time] (i.e. a even number, since the
block time is 2 seconds).

When updating the rollup protocol to a new version, we may perform a *squash fork*, a process that entails the creation
of a new L2 genesis block. This new L2 genesis block will have block number `X + 1`, where `X` is the block number of
the final L2 block before the update.

A squash fork is not to be confused with a *re-genesis*, a similar process that we employed in the past, which also
resets L2 block numbers, such that the new L2 genesis block has number 0. We will not employ re-genesis in the future.

Squash forks are superior to re-geneses because they avoid duplicating L2 block numbers, which breaks a lot of external
tools.

## L2 Chain Inception

[l2-chain-inception]: glossary.md#L2-chain-inception

The L1 block number at which the output roots for the [genesis block][l2-genesis] were proposed on the [output
oracle][output-oracle] contract.

In the current implementation, this is the L1 block number at which the output oracle contract was deployed or upgraded.

## Safe L2 Block

[safe-l2-block]: glossary.md#safe-l2-block

A safe L2 block is an L2 block can be derived entirely from L1 by a [rollup node][rollup-node]. This can vary between
different nodes, based on their view of the L1 chain.

## Safe L2 Head

[safe-l2-head]: glossary.md#safe-l2-head

The safe L2 head is the highest [safe L2 block][safe-l2-block] that a [rollup node][rollup-node] knows about.

## Unsafe L2 Block

[unsafe-l2-block]: glossary.md#unsafe-l2-block

An unsafe L2 block is an L2 block that a [rollup node][rollup-node] knows about, but which was not derived from the L1
chain. In sequencer mode, this will be a block sequenced by the sequencer itself. In validator mode, this will be a
block acquired from the sequencer via [unsafe sync][unsafe-sync].

## Unsafe L2 Head

[unsafe-l2-head]: glossary.md#unsafe-l2-head

The unsafe L2 head is the highest [unsafe L2 block][unsafe-l2-block] that a [rollup node][rollup-node] knows about.

## Unsafe Block Consolidation

[consolidation]: glossary.md#unsafe-block-consolidation

Unsafe block consolidation is the process through which the [rollup node][rollup-node] attempts to move the [safe L2
head] a block forward, so that the oldest [unsafe L2 block][unsafe-l2-block] becomes the new safe L2 head.

In order to perform consolidation, the node verifies that the [payload attributes][payload-attr] derived from the L1
chain match the oldest unsafe L2 block exactly.

See the [Engine Queue section][engine-queue] of the L2 chain derivatiaon spec for more information.

[engine-queue]: derivation.md#engine-queue

## Finalized L2 Head

[finalized-l2-head]: glossary.md#finalized-l2-head

The finalized L2 head is the highest L2 block that can be derived from *[finalized][finality]* L1 blocks — i.e. L1
blocks older than two L1 epochs (64 L1 [time slots][time-slot]).

[finality]: <https://hackmd.io/@prysmaticlabs/finality> *finalized* L1 data.

------------------------------------------------------------------------------------------------------------------------

# Other L2 Chain Concepts

## Address Aliasing

[address-aliasing]: glossary.md#address-aliasing

When a contract submits a [deposit][deposits] from L1 to L2, it's address (as returned by `ORIGIN` and `CALLER`) will be
aliased with a modified representation of the address of a contract.

- cf. [Deposit Specification](deposits.md#address-aliasing)

## Rollup Node

[rollup-node]: glossary.md#rollup-node

The rollup node is responsible for [deriving the L2 chain][derivation] from the L1 chain (L1 [blocks][block] and their
associated [receipts][receipt]).

The rollup node can run either in *validator* or *sequencer* mode.

In sequencer mode, the rollup node receives L2 transactions from users, which it uses to create L2 blocks. These are
then submitted to a [data availability provider][avail-provider] via [batch submission][batch-submission]. The L2 chain
derivation then acts as a sanity check and a way to detect L1 chain [re-orgs][reorg].

In validator mode, the rollup node performs derivation as indicated above, but is also able to "run ahead" of the L1
chain by getting blocks directly from the sequencer, in which case derivation serves to validate the sequencer's
behaviour.

A rollup node running in validator mode is sometimes called *a replica*.

> **TODO** expand this to include output root submission

See the [rollup node specification][rollup-node-spec] for more information.

## Rollup Driver

[rollup driver]: glossary.md#rollup-driver

The rollup driver is the [rollup node][rollup-node] component responsible for [deriving the L2 chain][derivation]
from the L1 chain (L1 [blocks][block] and their associated [receipts][receipt]).

> **TODO** delete this entry, alongside its reference — can be replaced by "derivation process" or "derivation logic"
> where needed

## L1 Attributes Predeployed Contract

[l1-attr-predeploy]: glossary.md#l1-attributes-predeployed-contract

A [predeployed contract][predeploy] on L2 that can be used to retrieve the L1 block attributes of L1 blocks with a given
block number or a given block hash.

cf. [L1 Attributes Predeployed Contract Specification](deposits.md#l1-attributes-predeployed-contract)

## L2 Output Root

[l2-output]: glossary.md#l2-output-root

A 32 byte value which serves as a commitment to the current state of the L2 chain.

cf. [Proposing L2 output commitments](proposals.md#l2-output-root-proposals-specification)

## L2 Output Oracle Contract

[output-oracle]: glossary.md#l2-output-oracle-contract

An L1 contract to which [L2 output roots][l2-output] are posted by the [sequencer].

> **TODO** expand

## Validator

[validator]: glossary.md#validator

A validator is an entity (individual or organization) that runs a [rollup node][rollup-node] in validator mode.

Doing so grants a lot of benefits similar to running an Ethereum node, such as the ability to simulate L2 transactions
locally, without rate limiting.

It also lets the validator verify the work of the [sequencer], by re-deriving [output roots][l2-output] and comparing
them against those submitted by the sequencer. In case of a mismatch, the validator can perform a [fault
proof][fault-proof].

## Fault Proof

[fault-proof]: glossary.md#fault-proof

An on-chain *interactive* proof, performed by [validators][validator], that demonstrates that a [sequencer] provided
erroneous [output roots][l2-output].

Fault proofs are not specified yet. For now, the best place to find information about fault proofs is the [Cannon
repository][cannon].

> **TODO** expand

## Time Slot

[time-slot]: glossary.md#time-slot

On L2, there is a block every 2 second (this duration is known as the [block time][block-time]).

We say that there is a "time slot" every multiple of 2s after the timestamp of the [L2 genesis block][l2-genesis].

On L1, post-[merge], the time slots are every 12s. However, an L1 block may not be produced for every time slot, in case
of even benign consensus issues.

## Block Time

[block-time]: glossary.md#block-time

The L2 block time is 2 second, meaning there is an L2 block at every 2s [time slot][time-slot].

Post-[merge], it could be said the that L1 block time is 12s as that is the L1 [time slot][time-slot]. However, in
reality the block time is variable as some time slots might be skipped.

Pre-merge, the L1 block time is variable, though it is on average 13s.

## Unsafe Sync

[unsafe-sync]: glossary.md#unsafe-sync

Unsafe sync is the process through which a [validator][validator] learns about [unsafe L2 blocks][unsafe-l2-block] from
the [sequencer][sequencer].

These unsafe blocks will later need to be confirmed by the L1 chain (via [unsafe block consolidation][consolidation]).

------------------------------------------------------------------------------------------------------------------------

# Execution Engine Concepts

## Execution Engine

[execution-engine]: glossary.md#execution-engine

The execution engine is responsible for executing transactions in blocks and computing the resulting state roots,
receipts roots and block hash.

Both L1 (post-[merge]) and L2 have an execution engine.

On L1, the executed blocks can come from L1 block synchronization; or from a block freshly minted by the execution
engine (using transactions from the L1 [mempool]), at the request of the L1 consensus layer.

On L2, the executed blocks are freshly minted by the execution engine at the request of the [rollup node][rollup-node],
using transactions [derived from L1 blocks][derivation].

In these specifications, "execution engine" always refer to the L2 execution engine, unless otherwise specified.

- cf. [Execution Engine Specification](exec-engine.md)

<!-- Internal Links -->
[derivation-spec]: derivation.md
[rollup-node-spec]: rollup-node.md

<!-- External Links -->
[mpt-details]: https://github.com/norswap/nanoeth/blob/d4c0c89cc774d4225d16970aa44c74114c1cfa63/src/com/norswap/nanoeth/trees/patricia/README.md
[trie]: https://en.wikipedia.org/wiki/Trie
[bloom filter]: https://en.wikipedia.org/wiki/Bloom_filter
[Solidity events]: https://docs.soliditylang.org/en/latest/contracts.html?highlight=events#events
[nano-header]: https://github.com/norswap/nanoeth/blob/cc5d94a349c90627024f3cd629a2d830008fec72/src/com/norswap/nanoeth/blocks/BlockHeader.java#L22-L156
[yellow]: https://ethereum.github.io/yellowpaper/paper.pdf
[engine-api]: https://github.com/ethereum/execution-apis/blob/main/src/engine/paris.md#PayloadAttributesV1
[merge]: https://ethereum.org/en/eth2/merge/
[mempool]: https://www.quicknode.com/guides/defi/how-to-access-ethereum-mempool
[L1 consensus layer]: https://github.com/ethereum/consensus-specs/#readme
[cannon]: https://github.com/ethereum-optimism/cannon
[eip4844]: https://www.eip4844.com/
