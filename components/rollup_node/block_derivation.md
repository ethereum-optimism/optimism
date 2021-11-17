# Block derivation

The logic which is used to derive the rollup chain from an L1 availability layer.

## Summary

The rollup chain can be deterministically derived given an L1 Ethereum chain. The fact that the entire rollup chain can be derived based on L1 blocks is _what makes OE a rollup_. This process can be represented as:

```
derive_rollup_chain(l1_blockchain) -> rollup_blockchain
```

In this document we define a block derivation function which is designed to:

1. Require no state other than what is easily accessible using L1 and L2 execution engine APIs.
2. Support sequencers and sequencer consensus.
3. Resilient sequencer censorship.

## Rollup Epochs

The rollup chain is subdivided into epochs. There is a 1:1 correspondence between L1 block numbers and epoch numbers. For L1 block number `n`, there is a corresponding rollup epoch `n` which can only be derived after L1 block number `n` is added to the L1 chain. An epoch contains one to many rollup blocks.

### Types of Blocks within Epochs

Within these epochs, there are two block types:

1. Deposit block
2. Sequencer block

Deposit blocks contain contextual information about L1 such as the block hash. They also contain transactions initiated on L1 by any user or contract that execute on L2.

Sequencer blocks are submitted by the sequencer and target _future_ epochs which satisfy the following two conditions:

1. Target epoch is larger than the current L1 block number.
2. Target epoch is less than the current L1 block number PLUS `sequencing_window_size`.

The ability for the sequencer to append sequencer blocks to future epochs allows the sequencer to predict and influence the state of the epoch before the L1 chain has mined it. This property is what enables _fast transaction confirmations_ via the sequencer replicator.

### Epoch Structure

Each epoch contains **1** deposit block and zero to many sequencer blocks. For epoch `n` the deposit block is derived using L1 block number `n - (sequencing_window_size + 1)`. The epoch's sequencer blocks are contained in any of the L1 blocks ranging from `n - sequencing_window_size` to `n`. This a range of blocks is called the "sequencing window".

The following diagram demonstrates the correspondence between L1 blocks and rollup blocks (ie L2 blocks):

![Sequencer block derivation diagram](../../assets/sequencer-block-derivation.svg)

## Deposit Blocks in Depth

For every L1 block (after the rollup's genesis) an L2 deposit block is created. These deposit blocks contain both a `ContextDeposit` and any number of `UserDeposit`s. Context deposits set contextual information about the latest L1 block (eg. `blockHash` and `timestamp`), and user deposits are L1 user initiated L2 transactions which guarantee liveness of the rollup chain even with a censoring sequencer.

The deposit block types are as follows:

```python
class DepositBlock(Block):
    deposits: List[Deposit]

class Deposit:
    feedIndex: uint64
    GasLimit:  uint64

class UserDeposit(Deposit):
    isEOA:       bool
    l1TxOrigin:  Address
    target:      Address
    data:        bytes

class ContextDeposit(Deposit):
    blockHash:   bytes32
    blockNumber: uint64
    timestamp:   uint64
    baseFee:     uint64
```

### Deposit Block derivation

The function `derive_deposit_block(l1_block_number)` is defined as:

- Derive the `ContextDeposit` using the L1 block body. Specifically the L1 block's `blockHash`, `blockNumber`, `timestamp`, and `baseFee`.
- Get all events emitted by the `DepositFeed` contract at block `l1_block_number`.
    - For each event, derive a new `UserDeposit` based on the emitted calldata. All fields should be emitted by DepositFeed contract.

#### **Footnote**: The DepositFeed ensures that no matter the size of the event it is possible to prove the deposit data to the fraud proof. See the DepositFeed spec for details.

## Sequencer Blocks in Depth

Sequencer blocks contain a majority of the transactions submitted to the rollup chain. Sequencer blocks can also influence the current timestamp of L2 which is exposed to contracts & is used in the EIP1559 fee pricing calculations.

Sequencer blocks are of the following form:

```python
class SequencerBlock(Block):
    target_epoch: uint64  # epoch number that this block is intended for
    sequencer_suggested_timestamp: uint64  # a timestamp suggested by the sequencer
    transactions: List[Transaction]  # A list of transactions in the block
```

These sequencer blocks must be validated as properly formatted. These validity checks are:

1. `target_epoch` must be within the sequencing window (introduced in the previous sections)
2. `sequencer_suggested_timestamp` must be monotonic & within an acceptable range.
3. `transactions` must all be correctly formatted (eg. contain valid signatures).

If any of these checks fail the block must be skipped or ignored. Otherwise, all transactions in the sequencer block will be executed.

## Timestamps in Depth

Time can be set by the sequencer; however, it must stay within reasonable bounds and must be monotonic (ie. can never go backwards). The specific conditions that must be met are:

```python
# last_l1_timestamp is the L1 timestamp set during the last ContextDeposit
# last_timestamp is the timestamp from the previous rollup block
# average_l1_block_time is the average time between two L1 blocks
# sequencing_window_size is the number of L1 blocks the sequencer has to submit their blocks
min_timestamp = max(last_l1_timestamp, last_timestamp)
max_timestamp = last_l1_timestamp + sequencing_window_size * average_l1_block_time

assert sequencer_suggested_timestamp >= min_timestamp
assert sequencer_suggested_timestamp <= max_timestamp
```

The timestamp exposed on the rollup chain is:

```python
# last_sequencer_suggested_timestamp is the most recent valid sequencer_suggested_timestamp
def timestamp():
    max(last_l1_timestamp, last_sequencer_suggested_timestamp)
```


## Rollup Epoch Generation

- To derive the blocks in epoch `n`, query L1 blocks `n - (sequencing_window_size + 1)` through `n`. Store the result in an array `l1_blocks`.
- The first block in the epoch is always the deposit block.
    - Derive the deposit block with `derive_deposit_block(l1_blocks[0])` (function defined above).
    - Append the deposit block the rollup chain.
- The rest of the blocks in the epoch are sequencer blocks. To derive these sequencer blocks, for each block in `l1_blocks`:
    - Extract the sequencer feed elements.
    - For each sequencer feed element in the L1 block:
        - Assert the target epoch is correct (`target_epoch == n`). If not skip over this feed element.
        - If the feed element is targetting the correct epoch, check that the sequencer feed element produces a valid block:
            - Assert the sequencer feed element was appended by the current sequencer public key (the current public key may exist in L2 state).
            - Assert the `sequencer_suggested_timestamp` is valid (see timestamp section for details)
            - Assert all transactions included in the sequencer feed element are valid.
        - If all checks pass, append the sequencer block to the rollup chain. Otherwise throw out the invalid feed element.
- At this point we will have appended all rollup blocks in epoch `n` to the rollup chain.

In order to derive the full rollup chain, repeat this process until `n==latest_l1_block_number` indicating that we have reached the tip of L1.