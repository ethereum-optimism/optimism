# Fraud Proof Program

The fraud proof program is the specific program which is run on a [fraud proof VM](./fraud_proof_vm.md) during a dispute. This program is designed to evaluate any subset of the Optimistic Ethereum chain.

## Initialization

The OE fraud proof program is used to weed out invalid block hash assertions in the `Block Hash Oracle`. The fraud proof program is initialized using values which are recorded at the time of block hash assertions. This is all handled within the `AssertionManager` contract.

Every time an L2 block hash is asserted using the `AssertionManager`, the following information is hashed together and recorded:

- Latest L1 block hash at the time of assertion.
- Asserted next L2 block hash.
- The previous assertion this assertion builds upon.
- Epoch number of the L2 block hash.

The fraud proof program takes in as input the hash of all these values.

## Execution

The fraud proof program is designed to execute the full state transition between any two epochs. The epochs used for a dispute are the previous assertion epoch & the new assertion epoch number in the `AssertionManager`. The fraud proof program will execute the state transition between these epochs and determine the final epoch block hash (ie. the one valid assertion). If the asserted L2 block hash is deemed incorrect over the course of a dispute game, the `AssertionManager` will delete the invalid assertion and forfeit the asserter's bond.

In practice, the fraud proof program is broken up into two separate processes:

1. Block generation
2. Block processing

We split the program into these two distinct parts to isolate the complexity of block processing. This way if there is an error which causes a single block to fail (see [this discussion](https://github.com/ethereum-optimism/optimistic-specs/discussions/22) for details), it does not break the entire assertion game. While the block processing may fail, it is critically important that the block generation process never fails.

### Block Generation

The fraud proof program starts out running the block generation process. Block generation uses the same algorithm which is laid out in the [block generation document](../rollup_node/block_gen.md). However, it is worth highlighting:

1. In the FPVM, all chain data is queried using the preimage oracle. The program will use the L1 block hash at the time of assertion to search backwards on L1 to find all information required for block generation. This can be done by recursively calling `get_preimage(block.previous_block_hash)` until all required block information is loaded into the FPVM memory.
2. The block generation document outlines the process to generate a single epoch. For the purposes of this fraud proof program we will need to generate **all** epochs between the previous assertion and the next assertion.
3. The final step in the block generation algorithm where blocks are **processed** is **not** handled in this top level program. Instead a sub-process is spawned to execute that single block.

### Block Processing

For every block which needs to be processed in the block generation algorithm, a sub-process is spawned. This process accepts the block inputs and returns the `block hash` computed by executing the block inputs. The specifics of the block processing algorithm is laid out in the [block processing document] [TODO].

## Termination

Once the fraud proof program has generated and processed all blocks between the last asserted L2 block hash and the newly asserted L2 block hash, it will return the **valid** L2 block hash that should have been asserted. In the proposal manager it will compare the value returned by the fraud proof with the value asserted, and if it is incorrect delete the invalid assertion and slash the asserter's bond.