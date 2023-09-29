# Honest Challenger (Fault Dispute Game)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [FDG Responses](#fdg-responses)
  - [Root Claims](#root-claims)
  - [Counter Claims](#counter-claims)
  - [Steps](#steps)
- [Resolution](#resolution)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The honest challenger is an agent interacting in the [Fault Dispute Game](./fault-dispute-game.md)
(FDG) that supports honest claims and disputes false claims.
An honest challenger strives to ensure a correct, truthful, game resolution.
The honest challenger is also _rational_ as any deviation from its behavior will result in
negative outcomes.
This document specifies the expected behavior of an honest challenger.

## Overview

The Honest Challenger has two primary duties:

1. Support valid root claims in Fault Dispute Games.
2. Dispute invalid root claims in Fault Dispute Games.

The honest challenger polls the `DisputeGameFactory` contract for new and on-going Fault
Dispute Games.
For verifying the legitimacy of claims, it relies on a synced, trusted rollup node
as well as a trace provider (ex: [Cannon](./cannon-fault-proof-vm.md)).
The trace provider must be configured with the [ABSOLUTE_PRESTATE](./fault-dispute-game.md#execution-trace)
of the FDG being interacted with to generate the traces needed to make truthful claims.

## FDG Responses

### Root Claims

When a `FaultDisputeGame` is created, the honest challenger has two possible correct responses
to its root claim:

1. [**Attack**](./fault-dispute-game.md#attack) if they disagree with the root claim.
The root claim commits to the entire execution trace, so the first move here is to
attack with the [ClaimHash](./fault-dispute-game.md#claims) at the midpoint
instruction within their execution trace.
2. **Do Nothing** if they agree with the root claim. They do nothing because if the root
claim is left un-countered, the game resolves to their agreement.
NOTE: The honest challenger will still track this game in order to defend any subsequent
claims made against the root claim - in effect, "playing the game".

### Counter Claims

For every claim made in a dispute game with a [game tree](./fault-dispute-game.md#game-tree)
depth in the range of `[1, MAX_DEPTH]`, the honest challenger processes them and performs
a response.

To determine the appropriate response, the challenger first needs to know which
[_team_](./fault-dispute-game.md#team-dynamics) it belongs to.
This determines the set of claims it should respond to in the FDG.
If the agent determines itself to be a Defender, which aims to support the root claim,
then it must dispute claims positioned at odd depths in the game tree.
Otherwise, it disputes claims positioned at even depths in the game tree.
This means an honest challenger only responds to claims made by the opposing team.

The next step is to determine if the claim, now known to be for the opposing team,
disputes another claim the honest challenger _agrees_ with.
An honest challenger agrees with a claim iff every other claim along its path to the
root claim commits to a valid `ClaimHash`. Put differently, an honest challenger will
avoid countering a claim if it disagrees with the path of claims leading to that
specific claim. But if the honest challenger agrees with the path leading to the claim,
then the claim is countered.

The last step is to determine whether the claim has a valid commitment (i.e. `ClaimHash`).
If the `ClaimHash` matches the honest challenger's at the same trace index, then we
disagree with the claim's stance by moving to [defend](./fault-dispute-game.md#defend).
Otherwise, the claim is [attacked](./fault-dispute-game.md#attack).

The following pseudocode illustrates the response logic.

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
            if chal_trace[trace_index(claim.position)] == claim.claim_hash:
                defend()
            else:
                attack()
        else: pass # avoid supporting invalid claims on the same team
```

In attack or defense, the honest challenger submit a `ClaimHash` corresponding to the
state identified by the trace index of their response position.

The honest challenger responds to claims as soon as possible to avoid the clock of its
counter-claim from expiring.

### Steps

At the max depth of the game, claims represent commitments to the state of the fault proof VM
at a single instruction step interval.
Because the game can no longer bisect further, when the honest challenger has a valid move
against these claims (valid defined by the response in [Counter Claims](#counter-claims)),
the only option for an honest challenger is to execute a VM step on-chain to disprove the claim at `MAX_GAME_DEPTH`.

Similar to the above section, the honest challenger will issue an
[attack step](./fault-dispute-game.md#step-types) when in response to such claims with
invalid `ClaimHash` commitments. Otherwise, it issues a _defense step_.

## Resolution

When the [chess clock](./fault-dispute-game.md#game-clock) of a
[subgame root](./fault-dispute-game.md#resolution) has run out, the subgame can be resolved.
The honest challenger should resolve all subgames in bottom-up order, until the subgame
rooted at the FDG root is resolved.

The honest challenger accomplishes this by calling the `resolveClaim` function on the
`FaultDisputeGame` contract. Once the root claim's subgame is resolved,
the challenger then finally calls the `resolve` function to resolve the entire game.

The `FaultDisputeGame` does not put a time cap on resolution - because of the liveness
assumption on honest challengers and the bonds attached to the claims they’ve countered,
challengers are economically incentivized to resolve the game promptly to capture the bonds.
