## Manual Fault Proof Interactions

### Creating a Game

The process of disputing an output root starts by creating a new dispute game. There are conceptually three key inputs
required for a dispute game:

- The output root being disputed
- The agreed output root the derivation process will start from
- The L1 head block that defines the canonical L1 chain containing all required batch data to perform the derivation

The creator of the game selects the output root to dispute. It is identified by its L2 block number which can be used to
look up the full details in the L2 output oracle.

The agreed output root is defined as the output root immediately prior to the disputed output root in the L2 output
oracle. Therefore, a dispute game should only be created for the first invalid output root. If it is successfully
disputed, all output roots after it are considered invalid by inference.

The L1 head block can be any L1 block where the disputed output root is present in the L2 output oracle. Proposers
should therefore ensure that all batch data has been submitted to L1 before submitting a proposal. The L1 head block is
recorded in the `BlockOracle` and then referenced by its block number.

Creating a game requires two separate transactions. First the L1 head block is recorded in the `BlockOracle` by calling
its `checkpoint` function. This records the parent of the block the transaction is included in. The `BlockOracle` emits
a log `Checkpoint(blockNumber, blockHash, childTimestamp)`.

Now, using the L1 head along with output root info available in the L2 output oracle, cannon can be executed to
determine the root claim to use when creating the game. In simple cases, where the claim is expected to be incorrect, an
arbitrary hash can be used for claim values. For more advanced cases [cannon can be used](./cannon.md) to generate a
trace, including the claim values to use at specific steps. Note that it is not valid to create a game that disputes an
output root, using the final hash from a trace that confirms the output root is valid. To dispute an output root
successfully, the trace must resolve that the disputed output root is invalid. This is indicated by the first byte of
the claim value being set to the invalid [VM status](../../specs/cannon-fault-proof-vm.md#state-hash) (`0x01`).

The game can then be created by calling the `create` method on the `DisputeGameFactory` contract. This requires three
parameters:

- `gameType` - a `uint8` representing the type of game to create. For fault dispute games using cannon and op-program
  traces, the game type is 0.
- `rootClaim` - a `bytes32` hash of the final state from the trace.
- `extraData` - arbitrary bytes which are used as the initial inputs for the game. For fault dispute games using cannon
  and op-program traces, this is the abi encoding of `(uint256(l2_block_number), uint256(l1_checkpoint))`.
    - `l2_block_number` is the L2 block number from the output root being disputed
    - `l1_checkpoint` is the L1 block number recorded by the `BlockOracle` checkpoint

This emits a log event `DisputeGameCreated(gameAddress, gameType, rootClaim)` where `gameAddress` is the address of the
newly created dispute game.

The helper script, [create_game.sh](../../op-challenger#create_gamesh) can be used to easily create a new dispute
game and also acts as an example of using `cast` to manually create a game.

### Performing Moves

The dispute game progresses by actors countering existing claims via either the `attack` or `defend` methods in
the `FaultDisputeGame` contract. Note that only `attack` can be used to counter the root claim. In both cases, there are
two inputs required:

- `parentIndex` - the index in the claims array of the parent claim that is being countered.
- `claim` - a `bytes32` hash of the state at the trace index corresponding to the new claim’s position.

The helper script, [move.sh](../../op-challenger#movesh), can be used to easily perform moves and also
acts as an example of using `cast` to manually call `attack` and `defend`.

### Performing Steps

Attacking or defending are the only available actions before the maximum depth of the game is reached. To counter claims
at the maximum depth, a step must be performed instead. Calling the `step` method in the `FaultDisputeGame` contract
counters a claim at the maximum depth by running a single step of the cannon VM on chain. The `step` method will revert
unless the cannon execution confirms the claim being countered is invalid. Note, if an actor's clock runs out at any
point, the game can be [resolved](#resolving-a-game).

The inputs for step are:

- `claimIndex` - the index in the claims array of the claim that is being countered
- `isAttack` - Similar to regular moves, steps can either be attacking or defending
- `stateData` - the full cannon state witness to use as the starting state for execution
- `proof` - the additional proof data for the state witness required by cannon to perform the step

When a step is attacking, the caller is asserting that the claim at `claimIndex` is incorrect, and the claim for
the previous trace index (made at a previous level in the game) was correct. The `stateData` must be the pre-image for
the agreed correct hash at the previous trace index. The call to `step` will revert if the post-state from cannon
matches the claim at `claimIndex` since the on-chain execution has proven the claim correct and it should not be
countered.

When a step is defending, the caller is asserting that the claim at `claimIndex` is correct, and the claim for
the next trace index (made at a previous level in the game) is incorrect. The `stateData` must be the pre-image for the
hash in the claim at `claimIndex`.

The `step` function will revert with `ValidStep()` if the cannon execution proves that the claim attempting to be
countered is correct. As a result, claims at the maximum game depth can only be countered by a valid execution of the
single instruction in cannon running on-chain.

#### Populating the Pre-image Oracle

When the instruction to be executed as part of a `step` call reads from some pre-image data, that data must be loaded
into the pre-image oracle prior to calling `step`.
For [local pre-image keys](../../specs/fault-proof.md#type-1-local-key), the pre-image must be populated via
the `FaultDisputeGame` contract by calling the `addLocalData` function.
For [global keccak256 keys](../../specs/fault-proof.md#type-2-global-keccak256-key), the data should be added directly
to the pre-image oracle contract.

### Resolving a Game

The final action required for a game is to resolve it by calling the `resolve` method in the `FaultDisputeGame`
contract. This can only be done once the clock of the left-most uncontested claim’s parent has expired. A game can only
be resolved once.

There are no inputs required for the `resolve` method. When successful, a log event is emitted with the game’s final
status.

The helper script, [resolve.sh](../../op-challenger#resolvesh), can be used to easily resolve a game and also acts as an
example of using `cast` to manually call `resolve` and understand the result.
