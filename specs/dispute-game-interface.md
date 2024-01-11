# Dispute Game Interface

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Types](#types)
- [`DisputeGameFactory` Interface](#disputegamefactory-interface)
- [`DisputeGame` Interface](#disputegame-interface)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

A dispute game is played between multiple parties when contesting the truthiness
of a claim. In the context of an optimistic rollup, claims are made about the
state of the layer two network to enable withdrawals to the layer one. A proposer
makes a claim about the layer two state such that they can withdraw and a
challenger can dispute the validity of the claim. The security of the layer two
comes from the ability of fraudulent withdrawals being able to be disputed.

A dispute game interface is defined to allow for multiple implementations of
dispute games to exist. If multiple dispute games run in production, it gives
a similar security model as having multiple protocol clients, as a bug in a
single dispute game will not result in the bug becoming consensus.

## Types

For added context, we define a few types that are used in the following snippets.

```solidity
/// @notice A `Claim` type represents a 32 byte hash or other unique identifier for a claim about
///         a certain piece of information.
type Claim is bytes32;

/// @notice A custom type for a generic hash.
type Hash is bytes32;

/// @notice A dedicated timestamp type.
type Timestamp is uint64;

/// @notice A `GameType` represents the type of game being played.
type GameType is uint8;

/// @title GameTypes
/// @notice A library that defines the IDs of games that can be played.
library GameTypes {
    /// @dev A dispute game type the uses the cannon vm.
    GameType internal constant CANNON = GameType.wrap(0);

    /// @dev A dispute game type that performs output bisection and then uses the cannon vm.
    GameType internal constant OUTPUT_CANNON = GameType.wrap(1);

    /// @notice A dispute game type that performs output bisection and then uses an alphabet vm.
    ///         Not intended for production use.
    GameType internal constant OUTPUT_ALPHABET = GameType.wrap(254);

    /// @notice A dispute game type that uses an alphabet vm.
    ///         Not intended for production use.
    GameType internal constant ALPHABET = GameType.wrap(255);
}

/// @notice The current status of the dispute game.
enum GameStatus {
    /// @dev The game is currently in progress, and has not been resolved.
    IN_PROGRESS,
    /// @dev The game has concluded, and the `rootClaim` was challenged successfully.
    CHALLENGER_WINS,
    /// @dev The game has concluded, and the `rootClaim` could not be contested.
    DEFENDER_WINS
}
```

## `DisputeGameFactory` Interface

The dispute game factory is responsible for creating new `DisputeGame` contracts
given a `GameType` and a root `Claim`. Challenger agents listen to the `DisputeGameCreated` events in order to
keep up with on-going disputes in the protocol and participate accordingly.

A [`clones-with-immutable-args`](https://github.com/Saw-mon-and-Natalie/clones-with-immutable-args) factory
(originally by @wighawag, but forked by @Saw-mon-and-Natalie) is used to create Clones. Each `GameType` has
a corresponding implementation within the factory, and when a new game is created, the factory creates a
clone of the `GameType`'s pre-deployed implementation contract.

The `rootClaim` of created dispute games can either be a claim that the creator agrees or disagrees with.
This is an implementation detail that is left up to the `IDisputeGame` to handle within its `resolve` function.

When the `DisputeGameFactory` creates a new `DisputeGame` contract, it calls `initialize()` on the clone to
set up the game.

```solidity
/// @title IDisputeGameFactory
/// @notice The interface for a DisputeGameFactory contract.
interface IDisputeGameFactory {
    /// @notice Emitted when a new dispute game is created
    /// @param disputeProxy The address of the dispute game proxy
    /// @param gameType The type of the dispute game proxy's implementation
    /// @param rootClaim The root claim of the dispute game
    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    /// @notice Emitted when a new game implementation added to the factory
    /// @param impl The implementation contract for the given `GameType`.
    /// @param gameType The type of the DisputeGame.
    event ImplementationSet(address indexed impl, GameType indexed gameType);

    /// @notice Emitted when a game type's initialization bond is updated
    /// @param gameType The type of the DisputeGame.
    /// @param newBond The new bond (in wei) for initializing the game type.
    event InitBondUpdated(GameType indexed gameType, uint256 indexed newBond);

    /// @notice The total number of dispute games created by this factory.
    /// @return gameCount_ The total number of dispute games created by this factory.
    function gameCount() external view returns (uint256 gameCount_);

    /// @notice `games` queries an internal mapping that maps the hash of
    ///         `gameType ++ rootClaim ++ extraData` to the deployed `DisputeGame` clone.
    /// @dev `++` equates to concatenation.
    /// @param _gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param _rootClaim The root claim of the DisputeGame.
    /// @param _extraData Any extra data that should be provided to the created dispute game.
    /// @return proxy_ The clone of the `DisputeGame` created with the given parameters.
    ///         Returns `address(0)` if nonexistent.
    /// @return timestamp_ The timestamp of the creation of the dispute game.
    function games(
        GameType _gameType,
        Claim _rootClaim,
        bytes calldata _extraData
    )
        external
        view
        returns (IDisputeGame proxy_, Timestamp timestamp_);

    /// @notice `gameAtIndex` returns the dispute game contract address and its creation timestamp
    ///          at the given index. Each created dispute game increments the underlying index.
    /// @param _index The index of the dispute game.
    /// @return gameType_ The type of the DisputeGame - used to decide the proxy implementation.
    /// @return timestamp_ The timestamp of the creation of the dispute game.
    /// @return proxy_ The clone of the `DisputeGame` created with the given parameters.
    ///         Returns `address(0)` if nonexistent.
    function gameAtIndex(uint256 _index)
        external
        view
        returns (GameType gameType_, Timestamp timestamp_, IDisputeGame proxy_);

    /// @notice `gameImpls` is a mapping that maps `GameType`s to their respective
    ///         `IDisputeGame` implementations.
    /// @param _gameType The type of the dispute game.
    /// @return impl_ The address of the implementation of the game type.
    ///         Will be cloned on creation of a new dispute game with the given `gameType`.
    function gameImpls(GameType _gameType) external view returns (IDisputeGame impl_);

    /// @notice Returns the required bonds for initializing a dispute game of the given type.
    /// @param _gameType The type of the dispute game.
    /// @return bond_ The required bond for initializing a dispute game of the given type.
    function initBonds(GameType _gameType) external view returns (uint256 bond_);

    /// @notice Creates a new DisputeGame proxy contract.
    /// @param _gameType The type of the DisputeGame - used to decide the proxy implementation.
    /// @param _rootClaim The root claim of the DisputeGame.
    /// @param _extraData Any extra data that should be provided to the created dispute game.
    /// @return proxy_ The address of the created DisputeGame proxy.
    function create(
        GameType _gameType,
        Claim _rootClaim,
        bytes calldata _extraData
    )
        external
        payable
        returns (IDisputeGame proxy_);

    /// @notice Sets the implementation contract for a specific `GameType`.
    /// @dev May only be called by the `owner`.
    /// @param _gameType The type of the DisputeGame.
    /// @param _impl The implementation contract for the given `GameType`.
    function setImplementation(GameType _gameType, IDisputeGame _impl) external;

    /// @notice Sets the bond (in wei) for initializing a game type.
    /// @dev May only be called by the `owner`.
    /// @param _gameType The type of the DisputeGame.
    /// @param _initBond The bond (in wei) for initializing a game type.
    function setInitBond(GameType _gameType, uint256 _initBond) external;

    /// @notice Returns a unique identifier for the given dispute game parameters.
    /// @dev Hashes the concatenation of `gameType . rootClaim . extraData`
    ///      without expanding memory.
    /// @param _gameType The type of the DisputeGame.
    /// @param _rootClaim The root claim of the DisputeGame.
    /// @param _extraData Any extra data that should be provided to the created dispute game.
    /// @return uuid_ The unique identifier for the given dispute game parameters.
    function getGameUUID(
        GameType _gameType,
        Claim _rootClaim,
        bytes memory _extraData
    )
        external
        pure
        returns (Hash uuid_);
}
```

## `DisputeGame` Interface

The dispute game interface defines a generic, black-box dispute. It exposes stateful information such as the status of
the dispute, when it was created, as well as the bootstrap data and dispute type. This interface exposes one state
mutating function, `resolve`, which when implemented should deterministically yield an opinion about the `rootClaim`
and reflect the opinion by updating the `status` to `CHALLENGER_WINS` or `DEFENDER_WINS`.

Clones of the `IDisputeGame`'s `initialize` functions will be called by the `DisputeGameFactory` atomically upon
creation.

```solidity
/// @title IDisputeGame
/// @notice The generic interface for a DisputeGame contract.
interface IDisputeGame is IInitializable {
    /// @notice Emitted when the game is resolved.
    /// @param status The status of the game after resolution.
    event Resolved(GameStatus indexed status);

    /// @notice Returns the timestamp that the DisputeGame contract was created at.
    /// @return createdAt_ The timestamp that the DisputeGame contract was created at.
    function createdAt() external view returns (Timestamp createdAt_);

    /// @notice Returns the current status of the game.
    /// @return status_ The current status of the game.
    function status() external view returns (GameStatus status_);

    /// @notice Getter for the game type.
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return gameType_ The type of proof system being used.
    function gameType() external view returns (GameType gameType_);

    /// @notice Getter for the root claim.
    /// @dev `clones-with-immutable-args` argument #1
    /// @return rootClaim_ The root claim of the DisputeGame.
    function rootClaim() external pure returns (Claim rootClaim_);

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #2
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function extraData() external pure returns (bytes memory extraData_);

    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    /// @return status_ The status of the game after resolution.
    function resolve() external returns (GameStatus status_);

    /// @notice A compliant implementation of this interface should return the components of the
    ///         game UUID's preimage provided in the cwia payload. The preimage of the UUID is
    ///         constructed as `keccak256(gameType . rootClaim . extraData)` where `.` denotes
    ///         concatenation.
    /// @return gameType_ The type of proof system being used.
    /// @return rootClaim_ The root claim of the DisputeGame.
    /// @return extraData_ Any extra data supplied to the dispute game contract by the creator.
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_);
}
```
