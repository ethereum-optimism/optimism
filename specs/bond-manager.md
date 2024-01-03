# Bond Manager Interface

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [The Bond Problem](#the-bond-problem)
  - [Simple Bond](#simple-bond)
  - [Variable Bond](#variable-bond)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

In the context of permissionless output proposals, bonds are a value that must
be attached to an output proposal. In this case, the bond will be paid in ether.

By requiring a bond to be posted with an output proposal, spam and invalid outputs
are disincentivized. Explicitly, if invalid outputs are proposed, challenge agents
can delete the invalid output via a [dispute-game](./dispute-game-interface.md) and seize the
proposer's bond. So, posting invalid outputs is directly disincentivized in this way
since the proposer would lose their bond if the challenge agents seize it.

Concretely, outputs will be permissionlessly proposed to the `L2OutputOracle` contract.
When submitting an output proposal, the ether value is sent as the bond. This bond is
then held by a bond manager contract. The bond manager contract is responsible for
both the [dispute-games](./dispute-game-interface.md) and the `L2OutputOracle` (further detailed
in [proposals](./proposals.md)).

The bond manager will need to handle bond logic for a variety of different
[dispute-games](./dispute-game-interface.md). In the simplest "attestation" dispute game,
bonds will not be required since the attestors are a permissioned set of trusted entities.
But in more complex games, such as the fault dispute game, challengers and defenders
perform a series of alternating onchain transactions requiring bonds at each step.

## The Bond Problem

At its core, the bond manager is straightforward - it escrows or holds ether and can be claimed
at maturity or seized if forfeited. But the uncertainty of introducing bonds lies in the
bond _sizing_, i.e. how much should a bond be? Sizing bonds correctly is a function of
the bond invariant: the bond must be greater than or equal to the cost of the next step.
If bonds are priced too low, then the bond invariant is violated and there isn't an economic
incentive to execute the next step. If bonds are priced too high, then the actors posting
bonds can be priced out.

Below, we outline two different approaches to sizing bonds and the tradeoffs of each.

### Simple Bond

The _Simple Bond_ is a very conservative approach to bond management, establishing a **fixed** bond
size. The idea behind simple bond pricing is to establish the worst case gas cost for
the next step in the dispute game.

With this approach, the size of the bond is computed up-front when a dispute game is created.
For example, in an attestation dispute game, this bond size can be computed as such:

```md
bond_size = (signer_threshold * (challenge_gas + security overhead)) + resolution_gas(signer_threshold)
```

Notice that since the bond size is linearly proportional to the number of signers, the economic
security a given bond size provides decreases as the number of signers increases. Also note, the
`resolution_gas` function is split out from the `challenge_gas` cost because only the _last_ challenger
will pay for the gas cost to resolve the game in the attestation dispute game.

Working backwards, if we assume the number of signers to be `5`, a negligible resolution gas cost, and
a `100,000` gas cost to progress the game, then the bond size should cover `500,000` gas. Meaning, a bond
of `1 ether` would cover the cost of progressing the game for `5` signers as long as the gas price
(base fee) does not exceed `2,000 gwei` for the entire finalization window. It would be prohibitively
expensive to keep the settlement layer base fee this high.

### Variable Bond

Better bond heuristics can be used to establish a bond price that accounts for
the time-weighted gas price. One instance of this called _Variable Bonds_ use a
separate oracle contract, `GasPriceFluctuationTracker`, that tracks gas fluctuations
within a pre-determined bounds. This replaces the ideal solution of tracking
challenge costs over all L1 blocks, but provides a reasonable bounds. The initial
actors posting this bond are responsible for funding this contract.
