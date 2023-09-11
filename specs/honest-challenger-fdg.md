# Honest Challenger (Fault Dispute Game)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Honest Challenger (Fault Dispute Game)](#honest-challenger-fault-dispute-game)
  - [Overview](#overview)
  - [L2OutputOracle Responses](#l2outputoracle-responses)
  - [FDG Responses](#fdg-responses)
    - [Root Claims](#root-claims)
    - [Counter Claims](#counter-claims)
    - [Steps](#steps)
  - [Resolution](#resolution)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The honest challenger is an agent interacting in the [Fault Dispute Game](./fault-dispute-game.md) (FDG) supporting honest claims and dispute false claims. An honest challenger strives to ensure a correct game resolution. The honest challenger is also _rational_ as its behavior is the only one that results in a positive outcome.
This document specifies the behavior of an honest challenger.

## Overview

The Honest Challenger has two responsibilities:
1. Challenge invalid outputs by creating FDGs that aim to delete them.
2. Challenge the root claims in FDGs that aim to delete valid outputs.

It monitors the `L2OutputOracle` to detect invalid proposals and the `DisputeGameFactory` to locate in-progress FDGs.

The Honest Challenger relies on a trusted rollup node that is synced to the canonical state and a trace provider (ex: [Cannon](./cannon-fault-proof-vm.md)) in order to determine the validity of proposed outputs and claims made to Fault Dispute Games. The trace provider must be configured with the [ABSOLUTE_PRESTATE](./fault-dispute-game.md#execution-trace) of the chosen FDG to generate the traces needed to make truthful claims.

## L2OutputOracle Responses

When a new output is proposed to the `L2OutputOracle`, the honest challenger has a binary decision to make:

1. If the trusted node agrees with the output, take no action. A `FaultDisputeGame` is designed to prove that a proposed output root is incorrect and to delete it. Therefore, an honest challenger will not create a dispute game that challenges an output root that its trusted node agrees with.
2. If the trusted node disagrees, create a new `FaultDisputeGame` via the `DisputeGameFactory`. In contrast to the above, an honest challenger aims to delete any output roots that its trusted node disagrees with in order to claim the bond attached to it. The honest challenger assumes that their rollup node is synced to the canonical state and that the fault proof program is correct, so it is willing to put its money on the line to counter any faults.

## FDG Responses

### Root Claims

When a `FaultDisputeGame` is created, the honest challenger has two possible correct responses to its root claim:

1. [**Attack**](./fault-dispute-game.md#attack) if they disagree with the root claim. When an honest challenger disagrees with a root claim of a game, it is akin to them agreeing with the output proposal that the game is attempting to delete. The root claim commits to the entire execution trace, so the first move by a defender of the output root is to attack with the [ClaimHash](./fault-dispute-game.md#claims) at the midpoint instruction within their execution trace.
2. **Do Nothing** if they agree with the root claim. If an honest challenger agrees with a root claim of a game, it means that they disagree with the output root it is trying to delete. They do nothing because if the root claim is left un-countered, the game will delete the output root they disagree with.

### Counter Claims

When a new claim is made in a dispute game with a [game tree](./fault-dispute-game.md#game-tree) depth in the range of `[1, MAX_DEPTH]`, the honest challenger processes it and performs a response. If multiple claims are observed since the last sync, the honest challenger responds to them in chronological order.

The challenger first needs to determine which [_team_](./fault-dispute-game.md#team-dynamics)
 it belongs to. This determines the set of claims it should respond to in the FDG.
If the agent determines itself to be a Defender, aiming to delete an output root,
 then it must dispute claims positioned at odd depths in the game tree.
Otherwise, the challenger dispute claims positioned at even depths in the game tree.
This means an honest challenger will only respond to claims made by the opposing team.

The next step is to determine if the claim, now at a depth it disagrees with, disputes another claim it _agrees with_. An honest challenger agrees with a claim iff every other claim along its path to the root claim commits to a valid `ClaimHash`. Thus, an honest challenger will avoid supporting invalid claims on the same team.

The last step is to determine whether the claim has a valid commitment (i.e. `ClaimHash`).
If the `ClaimHash` matches ours at the same trace index, then we disagree with the claim's stance
 by moving to [defend](./fault-dispute-game.md#defend).
Otherwise, the claim is [attacked](./fault-dispute-game.md#attack).


The following pseudocode illustrates the response logic at a high level.
```python
class Team(Enum):
    DEFENDER = 0
    CHALLENGER = 1

class Claim:
    parent: Claim
    position: uint64
    claim_hash: ClaimHash

MAX_TRACE = 2**MAX_GAME_DEPTH

def agree_with(claim: Claim, chal_trace: List[ClaimHash, MAX_TRACE]):
    if chal_trace[claim.trace_index] != claim.claim_hash:
        return False
    grand_parent = claim.parent.parent if claim.parent is not None else None
    if grand_parent is not None:
        return agree_with(grand_parent)
    return True

def respond(claim: Claim, chal: Team, chal_trace: List[ClaimHash, MAX_TRACE]):
    if depth(claim.position) % 2 != chal.value:
        if claim.parent is None or agree_with(claim.parent, chal_trace):
            if chal_trace[claim.trace_index] == claim.claim_hash:
                defend()
            else:
                attack()
        else: pass # no response
```

The honest challenger SHOULD respond to claims as soon as possible to avoid the clock of its counter-claim from expiring.

### Steps

At the max depth of the game, claims represent commitments to the state of the fault proof VM at a single instruction step interval. Because the game can no longer bisect further, when the honest challenger has a valid move against these claims (valid defined by the response in [Counter Claims](#counter-claims)), the only option for an honest challenger is to execute a VM step on-chain to disprove the claim at `MAX_GAME_DEPTH`. If the VM step proves this claim correct, the claim will be left uncountered.

The same rules for determining whether a move is an attack, defense, or noop from the above section apply to claims at the bottom level of the tree. Instead of calling `attack` or `defend`, the challenger issues a [step](./fault-dispute-game.md#step). An honest challenger will issue an attack step if it disagrees with the claim, otherwise a defense step is issued.

## Resolution

When one side of a `FaultDisputeGame`'s chess clock runs out, the honest challenger’s responsibility is to resolve the game. This action entails the challenger calling the `resolve` function on the `FaultDisputeGame` contract.

The `FaultDisputeGame` does not put a time cap on resolution - because of the liveness assumption on honest challengers and the bonds attached to the claims they’ve countered, challengers should resolve the game promptly in order to make their funds liquid again and capture their reward(s).
