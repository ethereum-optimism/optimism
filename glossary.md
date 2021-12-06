# Glossary

--------------------------------------------------------------------------------

# General Terms

## Layer 1 (L1)
[L1]: /glossary.md#layer-1-L1

Refers the Ethereum blockchain, used in contrast to [layer 2][L2], which
refers to Optimistic Ethereum.

## Layer 2 (L2)
[L2]: /glossary.md#layer-2-L2

Refers to the Optimistic Ethereum blockchain (specified in this repository),
used in contrast to [layer 1][L1], which refers to the Ethereum blockchain.

## Block
[block]: /glossary.md#block

Can refer to an [L1] block, or to an [L2] block, which are structured similarly.

A block is a sequential list of transactions, along with a couple of properties
stored in the *header* of the block. A description of these properties can be
found in code comments [here][nano-header], or in the [Ethereum yellow paper
(pdf)][yellow], section 4.3.

It is useful to distinguish between input block properties, which are known
before executing the transactions in the block, and output block properties,
which are derived after executing the block's transactions. These include
various [Merkle roots][Merkle root] that notably commit to the L2 state and to
the log events emitted during the execution.

## EOA
[EOA]: /glossary.md#EOA

"Externally Owned Account", an Ethereum term to designate addresses operated by
users, as opposed to contract addresses.

## Merkle Root
[Merkle root]: /glossary.md#merkle-roots

The Merkle root is the root hash of a [Merkle Patricia tree] (MPT). A MPT is a
sparse [trie], which is a tree-like structure that maps keys to values. The root
hash of a MPT is a commitment to the contents of the tree, which allows a proof
to be constructed for any key-value mapping encoded in the tree. Such a proof is
called a Merkle proof, and can be verified against the Merkle root.

## Chain Re-Organization
[reorg]: /glossary.md#chain-re-organization

A re-organization, or re-org for short, is whenever the head of a blockchain
(its last block) changes (as dictated by the [fork choice rule]) to a block that
is not a child of the previous head.

L1 re-orgs can happen because of network conditions or attacks. L2 re-orgs are a
consequence of L1 re-orgs, mediated via [L2 chain derivation][derivation].

## Predeployed Contract ("Predeploy")
[predeploy]: /glossary.md#predeployed-contract-predeploy

A contract placed in the L2 genesis state (i.e. at the start of the chain).

Optimistic Ethereum has the following predeploys:

- [L1 Attributes Predeployed Contract][l1-attr-predeploy]

--------------------------------------------------------------------------------

# L2 Chain Concepts

## L2 Chain Inception
[L2 chain inception]: /glossary.md#L2-chain-inception

The L1 block number for which the first block of the L2 chain was generated.

## Rollup Node
[rollup node]: /glossary.md#rollup-node

The rollup node is responsible for [deriving the L2 chain][derivation] from the
[L2 derivation inputs][deriv-inputs] available on L1. This is done by its
[rollup driver] component.

- cf. [Rollup Node Specification](/rollup-node.md)

## Rollup Driver
[rollup driver]: /glossary.md#rollup-driver

The rollup driver is the [rollup node] component responsible for [deriving the
L2 chain][derivation] from the [L2 derivation inputs][deriv-inputs] available on
L1.

## L2 Chain Derivation
[derivation]:  /glossary.md#L2-chain-derivation

A process that reads [L2 derivation inputs][deriv-inputs] from L1 in order to
derive the L2 chain.

cf. [L2 Chain Derivation (in Rollup Node
Specification)](/rollup-node.md#l2-chain-derivation)

## L2 Derivation Inputs
[deriv-inputs]: /glossary.md#l2-chain-derivation-inputs

This term refers to data that is found in L1 blocks and is read by the [rollup
node] to construct [payload attributes].

Chain derivation attributes include:
- L1 block attributes
   - block number
   - timestamp
   - basefee
- [deposits]

## Payload Attributes
[payload attributes]: /glossary.md#payload-attributes

This term refers to data that can be derived from [L2 chain derivation
inputs][deriv-inputs] found on L1, which are then passed to the [execution
engine] to construct L2 blocks.

"Payload attributes" is a term that originates and is specified in the [Ethereum
Engine API specification][engine-api], which we extend in this specification.

cf. [Execution Engine Specification](TODO)

> **TODO LINK** execution engine specification

Payload attributes were historically called "L2 block inputs" in the L2 spec and
you might still hear some people using this term.

## L1 Attributes Transaction
[l1-attributes-tx]: /glossary.md#l1-attributes-transaction

A transaction with an Optimistic-Ethereum-specific transaction type, that is
used to register the L1 block attributes (number, timestamp, ...) on L2.

The L1 attributes for a given L1 block can be read on L2 from the [L1 Attributes
Predeployed Contract][l1-attr-predeploy].

cf. [L1 attributes transaction format](/rollup-node.md#payload-transaction-format)
(in the section on [payload attributes])

> **TODO** We might want to move this the format spec to the execution engine.
> **TODO** We might wish to make this a "normal transaction" if deposits end up
> not carrying a signature.

## L1 Attributes Predeployed Contract
[l1-attr-predeploy]: /glossary.md#l1-attributes-predeployed-contract

A [predeployed contract][predeploy] on L2 that can be used to retrieve the L1
block attributes of L1 blocks with a given block number or a given block hash.

cf. [L1 Attributes Predeployed Contract Specification](TODO)

> **TODO LINK** L1 attributes predeployed contract spec

## Deposits
[deposits]: /glossary.md#deposits

A deposit is an L2 transaction that has been submitted on L1, via a transaction
sent to the [deposit feed contract][deposit-feed].

While deposits are notably (but not only) used to "deposit" (bridge) ETH and
tokens to L2, the word *deposit* should be understood as "a transaction
*deposited* to L2".

Deposits are one kind of [L2 derivation input][deriv-input].

## Deposit Feed Contract
[deposit-feed]: /glossary.md#deposit-feed-contract

An [L1] contract to which [EOAs][EOA] and contracts may send [deposits]. The
deposits are emitted as log records (in Solidity, these are called *events*) for
consumption by [rollup nodes][rollup node].

Advanced note: the deposits are not stored in calldata because they can be send
by contracts, in which case the calldata is part of the execution, but its value
is not captured in one of the [Merkle roots][Merkle root] included in the L1
block.

cf. [Deposit Feed Contract Specification](TODO)

> **TODO LINK** deposit feed contract specification

--------------------------------------------------------------------------------

# Execution Engine Concepts

## Execution Engine
[execution engine]: /glossary.md#execution-engine

The execution engine is responsible for executing transactions in blocks and
computing the resulting state roots, receipts roots and block hash.

Both L1 (post-[merge]) and L2 have an execution engine.

On L1, the executed blocks can come from L1 block synchronization; or from a block
freshly minted by the execution engine (using transactions from the L1
[mempool]), at the request of the L1 consensus layer.

On L2, the executed blocks are freshly minted by the execution engine at the
request of the [rollup node], using transactions [derived from L1
blocks][derivations].

In these specifications, "execution engine" always refer to the L2 execution
engine, unless otherwise specified.



<!-- External Links -->
[Merkle Patricia tree]: https://github.com/norswap/nanoeth/blob/d4c0c89cc774d4225d16970aa44c74114c1cfa63/src/com/norswap/nanoeth/trees/patricia/README.md
[trie]: https://en.wikipedia.org/wiki/Trie
[nano-header]: https://github.com/norswap/nanoeth/blob/cc5d94a349c90627024f3cd629a2d830008fec72/src/com/norswap/nanoeth/blocks/BlockHeader.java#L22-L156
[yellow]: https://ethereum.github.io/yellowpaper/paper.pdf
[engine-api]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#PayloadAttributesV1
[merge]: https://ethereum.org/en/eth2/merge/
[mempool]: https://www.quicknode.com/guides/defi/how-to-access-ethereum-mempool
[L1 consensus layer]: https://github.com/ethereum/consensus-specs/#readme
