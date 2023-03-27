# Dispute Game Interface

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Dispute Game Interface](#dispute-game-interface)
  - [Disputable Interface](#disputable-interface)

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

## Dispute Game Interface

The dispute game interface should be generic enough to allow it to work with any
proof system. This means that it should work fault proofs, validity proofs or
an attestation based proof system.

```solidity
/// @notice The type of proof system being used.
enum GameType {
    Fault,
    Validity,
    Attestation
}

/// @title IDisputeGame
/// @notice The generic interface for a DisputeGame contract.
interface IDisputeGame {
    /// @notice Initializes the DisputeGame contract.
    /// @dev It is recommended that the implementations of this interface only allow this function to be called once.
    function initialize() external;

    /// @notice Returns the semantic version of the DisputeGame contract
    function version() external pure returns (string memory _version);

    /// @return _gameType The type of proof system being used.
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    /// i.e. The game type should indicate the security model.
    function gameType() external view returns (GameType _gameType);
}
```

The generic dispute game interface is extended to work specifically with a fault
proof or a validity proof since their modes of operation are quite different.

### Disputable Interface

Any contract that can be disputed by a dispute game implementation should
implement the disputable interface. This allows for any dispute game to
successfully interact with a variety of contracts.

```solidity
/// @title IDisputable
/// @notice The generic interface for a disputable contract
interface IDisputable {
    /// @notice The implementation of the DisputeGame calls this after a
    /// successful dispute. This should be guarded such that it is only
    /// callable by a DisputeGame.
    function challenge(bytes memory _data) external;
}
```
