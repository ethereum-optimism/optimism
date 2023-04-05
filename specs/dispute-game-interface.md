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
/// @notice The type of proof system being used.
enum GameType {
    /// @dev The game will use a `IDisputeGame` implementation that utilizes fault proofs.
    FAULT,
    /// @dev The game will use a `IDisputeGame` implementation that utilizes validity proofs.
    VALIDITY,
    /// @dev The game will use a `IDisputeGame` implementation that utilizes attestation proofs.
    ATTESTATION
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

/// @notice A dedicated timestamp type.
type Timestamp is uint64;

/// @notice A `Claim` type represents a 32 byte hash or other unique identifier for a claim about
///         a certain piece of information.
/// @dev For the `FAULT` `GameType`, this will be a root of the merklized state of the fault proof
///      program at the end of the state transition.
///      For the `ATTESTATION` `GameType`, this will be an output root.
type Claim is bytes32;
```

## `DisputeGameFactory` Interface

The dispute game factory is responsible for creating new `DisputeGame` contracts
given a `GameType` and a root `Claim`. Challenger agents will listen to the
`DisputeGameCreated` events that are emitted by the factory as well as other events
that pertain to detecting fault (i.e. `OutputProposed(bytes32,uint256,uint256,uint256)`) in order to keep up
with on-going disputes in the protocol.

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

    /// @notice `games` queries an internal a mapping that maps the hash of `gameType ++ rootClaim ++ extraData`
    ///          to the deployed `DisputeGame` clone.
    /// @dev `++` equates to concatenation.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return _proxy The clone of the `DisputeGame` created with the given parameters. Returns `address(0)` if nonexistent.
    function games(GameType gameType, Claim rootClaim, bytes calldata extraData) external view returns (IDisputeGame _proxy);

    /// @notice `gameImpls` is a mapping that maps `GameType`s to their respective `IDisputeGame` implementations.
    /// @param gameType The type of the dispute game.
    /// @return _impl The address of the implementation of the game type. Will be cloned on creation of a new dispute game
    ///               with the given `gameType`.
    function gameImpls(GameType gameType) public view returns (IDisputeGame _impl);

    /// @notice The owner of the contract.
    /// @dev Owner Permissions:
    ///      - Update the implementation contracts for a given game type.
    /// @return _owner The owner of the contract.
    function owner() public view returns (address _owner);

    /// @notice Creates a new DisputeGame proxy contract.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    function create(GameType gameType, Claim rootClaim, bytes calldata extraData) external returns (IDisputeGame proxy);

    /// @notice Sets the implementation contract for a specific `GameType`
    /// @dev May only be called by the `owner`.
    /// @param gameType The type of the DisputeGame
    /// @param impl The implementation contract for the given `GameType`
    function setImplementation(GameType gameType, IDisputeGame impl) external;
}
```

## `DisputeGame` Interface

The dispute game interface should be generic enough to allow it to work with any
proof system. This means that it should work fault proofs, validity proofs,
an attestation based proof system, or any other source of truth that adheres to
the interface.

Clones of the `IDisputeGame`'s `initialize` functions will be called by the `DisputeGameFactory` upon creation.

```solidity
////////////////////////////////////////////////////////////////
//                    GENERIC DISPUTE GAME                    //
////////////////////////////////////////////////////////////////

/// @title IDisputeGame
/// @notice The generic interface for a DisputeGame contract.
interface IDisputeGame {
    /// @notice Initializes the DisputeGame contract.
    /// @custom:invariant The `initialize` function may only be called once.
    function initialize() external;

    /// @notice Returns the semantic version of the DisputeGame contract
    function version() external pure returns (string memory _version);

    /// @notice Returns the timestamp that the DisputeGame contract was created at.
    function createdAt() external pure returns (Timestamp _createdAt);

    /// @notice Returns the current status of the game.
    function status() external view returns (GameStatus _status);

    /// @notice Getter for the game type.
    /// @dev `clones-with-immutable-args` argument #1
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return _gameType The type of proof system being used.
    function gameType() external view returns (GameType _gameType);

    /// @notice Getter for the root claim.
    /// @return _rootClaim The root claim of the DisputeGame.
    /// @dev `clones-with-immutable-args` argument #2
    function rootClaim() external view returns (Claim _rootClaim);

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return _extraData Any extra data supplied to the dispute game contract by the creator.
    function extraData() external view returns (bytes memory _extraData);

    /// @notice Returns the address of the `BondManager` used 
    function bondManager() public view returns (IBondManager _bondManager);

    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    function resolve() public returns (GameStatus _status);
}

////////////////////////////////////////////////////////////////
//              OUTPUT ATTESTATION DISPUTE GAME               //
////////////////////////////////////////////////////////////////

/// @title IDisputeGame_OutputAttestation
/// @notice The interface for an attestation-based DisputeGame meant to contest output
///         proposals in Optimism's `L2OutputOracle` contract.
interface IDisputeGame_OutputAttestation is IDisputeGame {
    /// @notice A mapping of addresses from the `signerSet` to booleans signifying whether
    ///         or not they have authorized the `rootClaim` to be invalidated.
    function challenges(address challenger) external view returns (bool _challenged);

    /// @notice The signer set consists of authorized public keys that may challenge the `rootClaim`.
    /// @return An array of authorized signers.
    function signerSet() external view returns (address[] memory _signers);

    /// @notice The amount of signatures required to successfully challenge the `rootClaim`
    ///         output proposal. Once this threshold is met by members of the `signerSet`
    ///         calling `challenge`, the game will be resolved to `CHALLENGER_WINS`.
    /// @custom:invariant The `signatureThreshold` may never be greater than the length of the `signerSet`.
    function signatureThreshold() public view returns (uint16 _signatureThreshold);

    /// @notice Returns the L2 Block Number that the `rootClaim` commits to. Exists within the `extraData`.
    function l2BlockNumber() public view returns (uint256 _l2BlockNumber);

    /// @notice Challenge the `rootClaim`.
    /// @dev - If the `ecrecover`ed address that created the signature is not a part of the
    ///      signer set returned by `signerSet`, this function should revert.
    ///      - If the `ecrecover`ed address that created the signature is not the msg.sender,
    ///      this function should revert.
    ///      - If the signature provided is the signature that breaches the signature threshold,
    ///      the function should call the `resolve` function to resolve the game as `CHALLENGER_WINS`.
    ///      - When the game resolves, the bond attached to the root claim should be distributed among
    ///      the signers who participated in challenging the invalid claim.
    /// @param signature An EIP-712 signature committing to the `rootClaim` and `l2BlockNumber` (within the `extraData`)
    ///                  from a key that exists within the `signerSet`.
    function challenge(bytes calldata signature) external;
}
```
