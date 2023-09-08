# Fault Dispute Game

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [Definitions](#definitions)
  - [Virtual Machine (VM)](#virtual-machine-vm)
  - [Execution Trace](#execution-trace)
  - [Claims](#claims)
  - [DAG](#dag)
  - [Game Tree](#game-tree)
  - [Position](#position)
  - [GAME_DURATION](#game_duration)
- [Game Mechanics](#game-mechanics)
  - [Actors](#actors)
  - [Moves](#moves)
    - [Attack](#attack)
    - [Defend](#defend)
  - [Step](#step)
  - [Step Types](#step-types)
  - [Team Dynamics](#team-dynamics)
  - [Game Clock](#game-clock)
  - [Resolution](#resolution)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The Fault Dispute Game (FDG) is a specific type of
[dispute game](./dispute-game-interface.md) that verifies the validity of a
root claim by iteratively bisecting an execution trace down to a single instruction step.
It relies on a Virtual Machine (VM) to falsify invalid claims made
at a single instruction step.

Actors, i.e. Players, interact with the game by making claims that dispute other claims in the FDG.
Each claim made narrows the execution trace until the source of dispute is a single state transition.
Once a time limit is reached, the dispute game is _resolved_, based on
claims made that are disputed and which aren't, to determine the winners of the game.

## Definitions

### Virtual Machine (VM)

This is a state transition function (STF) that takes a _pre-state_ and computes the post-state.
The VM may reference external data during the STF and as such, it also accepts a _proof_ of this data.
Typically, the pre-state contains a commitment to the _proof_ to verify the integrity of the data referenced.

Mathemtically, we define the STF as $VM(S_i,P_i)$ where

- $S_i$ is the pre-state
- $P_i$ is an optional proof needed for the transition from $S_i$ to $S_{i+1}$.

### Execution Trace

An execution trace $T$ is a sequence $(S_0,S_1,S_2,...,S_n)$ where each $S_i$ is a VM state and
for each $i$, $0 \le i \lt n$, $S_{n+1} = VM(S_i, P_i)$.
Every execution trace has a unique starting state, $S_0$, that's preset to a FDG implementation.
We refer to this state as the **ABSOLUTE\_PRESTATE**.

### Claims

Claims assert an execution trace. This is represented as `ClaimHash`, a `bytes32` commitment to
the last VM state in the trace. A FDG is initialized with a root claim, which commits to the entire
execution trace. As we'll see later, there can be multiple claims, committing to different states in the FDG.

### DAG

A Directed Acyclic Graph $G = (V,E)$ representing the relationship between claims, where:

- $V$ is the set of nodes, each representing a claim. Formally, $V = \{C_1,C_2,...,C_n\}$,
where $C_i$ is a claim.
- $E$ is the set of _directed_ edges. An edge $(C_i,C_j)$ exists if $C_j$ is a direct dispute
against $C_i$ through either an "Attack" or "Defend" [move](#moves).

### Game Tree

The Game Tree is a binary tree of positions. Every claim in the DAG references a position in the Game Tree.
 The Game Tree has a maximum depth, `MAX_GAME_DEPTH`, that's preset to an FDG implementation.
Thus, the Game Tree contains $2^{d-1}$ positions, where $d$ is the `MAX_GAME_DEPTH`
(unless $d=0$, in which case there's only 1 position).

### Position

A position represents the location of a claim in the Game Tree. This is represented by a
"generalized index" (or **gindex**) where the high-order bit is the level in the tree and the remaining
bits is a unique bit pattern, allowing a unique identifier for each node in the tree.

The **gindex** of a position $n$ can be calculated as $2^{d(n)} + idx(n)$, where:

- $d(n)$ is a function returning the depth of the position in the Game Tree
- $idx(n)$ is a function returning the index of the position at its depth (starting from the left).

Positions at the deepest level of the game tree correspond to indices in the execution trace.
Positions higher up the game tree also cover the deepest, right-most positions relative to the current position.
We refer to this coverage as the **trace index** of a Position.

> This means claims commit to an execution trace that terminates at the same index as their Position's trace index.

Note that there can be multiple positions covering the same _trace index_.

### GAME_DURATION

This is an immutable, preset to a FDG implementation, representing the duration of the game.

## Game Mechanics

### Actors

The game involves two types of participants (or Players): **Challengers** and **Defenders**.
These players are grouped into separate teams, each employing distinct strategies to interact with the game.
Team members share a common goal regarding the game's outcome. Players interact with the game primarily through _moves_.

### Moves

A Move is a challenge against a claim's execution trace and must include an alternate claim asserting a different trace.
Moves can either be attacks or defenses and serve to update to DAG by adding nodes and edges targeting the disputed
claim.

Initially, claims added to the DAG are _uncontesteed_ (i.e. not **countered**). Once a move targets a claim, that claim
 is considered countered.
The status of a claim &mdash; whether it's countered or not &mdash; helps determine its validity and, ultimately, the
 game's winner.

#### Attack

A logical move made when a claim is disagreed with.
A claim at the relative attack position to a node, `n`, in the Game Tree commits to half
of the trace of the `n`’s claim.
The attack position relative to a node can be calculated by multiplying its gindex by 2.

To illustrate this, here's a Game Tree highlighting an attack on a Claim positioned at 6.

![Attacking node 6](assets/attack.png)

Attacking the node at 6 moves creates a new claim positioned at 12.

#### Defend

The logical move against a claim when you agree with both it and its parent.
A defense at the relative position to a node, `n``, in the Game Tree commits to the first half of n + 1’s trace range.

![Defend at 4](assets/defend.png)

Note that because of this, some nodes may never exist within the Game Tree.
However, they're not necessary as these nodes have complimentary, valid positions
with the same trace index within the tree. For example, a Position with gindex 5 has the same
trace index as another Position with gindex 2. We can verify that all trace indices have valid moves within the game:

![Game Tree Showing All Valid Move Positions](assets/valid-moves.png)

There may be multiple claims at the same position, so long as their `ClaimHash` are unique.

Each move adds new claims to the Game Tree at strictly increasing depth.
Once a claim is at `MAX_GAME_DEPTH`, the only way to dispute such claims is to **step**.

### Step

At `MAX_GAME_DEPTH`, the position of claims correspond to indices of an execution trace.
It's at this point that the FDG is able to query the VM to determine the validity of claims,
by checking the states they're committing to.
This is done by applying the VM's STF to the state a claim commits to.
If the STF post-state does not match the claimed state, the challenge succeeds.

```solidity
/// @notice Perform an instruction step via an on-chain fault proof processor.
/// @dev This function should point to a fault proof processor in order to execute
///      a step in the fault proof program on-chain. The interface of the fault proof
///      processor contract should adhere to the `IBigStepper` interface.
/// @param _claimIndex The index of the challenged claim within `claimData`.
/// @param _isAttack Whether or not the step is an attack or a defense.
/// @param _stateData The stateData of the step is the preimage of the claim at the given
///        prestate, which is at `_stateIndex` if the move is an attack and `_claimIndex` if
///        the move is a defense. If the step is an attack on the first instruction, it is
///        the absolute prestate of the fault proof VM.
/// @param _proof Proof to access memory nodes in the VM's merkle state tree.
function step(uint256 _claimIndex, bool isAttack, bytes calldata _stateData, bytes calldata _proof) external;
```

### Step Types

Similar to moves, there are two ways to step on a claim; attack or defend.
These determine the pre-state input to the VM STF and the expected output.

- **Attack Step** - Challenges a claim by providing a pre-state, proving an invalid state transition.
It uses the previous state in the execution trace as input and expects the disputed claim's state as output.
There must exist a claim in the DAG that commits to the input.
- **Defense Step** - Challenges a claim by proving it was an invalid attack,
thereby defending the disputed ancestor's claim. It uses the disputed claim's state as input and expects
the next state in the execution trace as output. There must exist a claim in the DAG that commits to the
 expected output.

The FDG step handles the inputs to the VM and asserts the expected output.
A step that successfully proves an invalid post-state (when attacking) or pre-state (when defending) is a
successful counter against the disputed claim.
Players interface with `step` by providing an indicator of attack and state data (including any proofs)
that corresponds to the expected pre/post state (depending on whether it's an attack or defend).
The FDG will assert that an existing claim commits to the state data provided by players.

### Team Dynamics

Challengers seek to dispute the root claim, while Defenders aim to support it.
Both types of actors will move accordingly to support their team. For Challengers, this means
attacking the root claim and disputing claims positioned at even depths in the Game Tree.
Defenders do the opposite by disputing claims positioned at odd depths.

Players on either team are motivated to support the actions of their teammates.
This involves countering disputes against claims made by their team (assuming these claims are honest).
Uncontested claims are likely to result in a loss, as explained later under [Resolution](#resolution).

### Game Clock

Every claim in the game has a Clock. A claim's inherits the clock of its grandparent claim in the
 DAG (and so on). Akin to a chess clock, it keeps track of the total time each team takes to make
  moves, preventing delays.
Making a move resumes the clock for the disputed claim and puases it for the newly added one.

A move against a particular claim is no longer possible once the parent of the disputed claim's Clock
 has exceeded half of the `GAME_DURATION`. By which point, the claim's clock has _expired_.

### Resolution

Resolving the FDG determines which team won.
This is done by examining the left-most, uncontested (or undisputed) claim.
If the depth of this claim is odd then Challengers win, otherwise Defenders win.
Resolution is only possible once the Clock of the left-most, uncontested claim has expired.

Given these rules, players are motivated to move quickly to challenge dishonest claims.
Each move bisects the execution trace and eventually, `MAX_GAME_DEPTH` is reached where disputes
can be settled conclusively. Dishonest players are disincentivized to participate, via backwards induction,
as an invalid claim won't remain uncontested. Further incentives can be added to the game by requiring
claims to be bonded, while rewarding game winners using the bonds of dishonest claims.
