# DAO explained

## Overview

The DAO is a fork of Compound Finance's current governance module (Governor Bravo), which is comprised of 4 main contracts:

### Token

`Comp.sol` encodes the token, which follows the ERC-20 standard and grants token holders voting power proportional to the number of tokens they own. These votes can be delegated to either themselves or a trusted delegate who will be able to vote on proposals on the token owner's behalf.

### Governor Bravo Delegate

`GovernorBravoDelegate.sol` contains the current implementation of the DAO. This contract allows token holders with voting power greater than the proposal threshold (between 50,000 - 100,000 votes) to create proposals that others can vote on. After a voting delay (up to 1 week) for participants to review the proposal, the voting period opens for at least 1 day and can last up to 2 weeks. For a proposal to pass, the number of for votes must be greater than the number of against votes and the number of for votes must meet the quorum of 400,000 votes. Once a proposal passes, it can be queued and then executed to go into effect.

### Governor Bravo Delegator

`GovernorBravoDelegator.sol` is the proxy that allows the DAO to be upgradeable. This contract is simply a wrapper that points to an implementation (currently `GovernorBravoDelegate`), but can be changed to a newer implementation of the DAO when appropriate.

### Timelock

`Timelock.sol` delays the implementation of actions passed by the governance module. The minimum delay is 2 days and can be increased up to 30 days for major changes. The purpose of this security feature is to ensure that the community is given enough time to react to and prepare for changes that are passed.

## Deployment

The deployment script can be found in `migrations/1_deploy.js` and can be used to deploy the DAO and provides examples of basic calls that can be made to the governance contracts.

First, we deploy the token and pass in the developer address which receives the initial supply of tokens. Then, we deploy the timelock with the developer address and chosen timelock delay (between 2 and 30 days). The developer address is set as the temporary admin of the timelock. Next, we deploy the GovernorBravoDelegate contract. Finally, we pass in the timelock address, token address, developer address, GovernorBravoDelegate address, voting period, voting delay, and proposal threshold to deploy GovernorBravoDelegator.

After deploying these contracts, we set GovernorBravoDelegator as the admin of the timelock contract by first queueing this transaction. The function `queueTransaction` takes in 5 arguments: target contract address, value of ether, function signature, function data, and estimated time of arrival (ETA) where the ETA must satisfy the timelock delay. Once the transaction is queued and the ETA has arrived, the transaction can be executed by calling the function `executeTransaction` with the same 5 arguments. Note: there is a grace period of 14 days from the ETA where the transaction must be executed before it becomes stale.

Once GovernorBravoDelegator has been set as the admin of the Timelock contract, the `_initiate` function can be called which allows proposals to be created and the BOBA DAO is live!

## Changes to Compound's Governance Contracts

In `GovernorBravoDelegate.sol`, modify the `_initiate` function:

- Change line 326 to `proposalCount = 1;`
- Delete line 321
- Delete parameter (`address governorAlpha`) in line 323

```
    /**
      * @notice Initiate the GovernorBravo contract
      * @dev Admin only. Sets initial proposal id which initiates the contract, ensuring a continuous proposal id count
      */
    function _initiate() external {
        require(msg.sender == admin, "GovernorBravo::_initiate: admin only");
        require(initialProposalId == 0, "GovernorBravo::_initiate: can only initiate once");
        proposalCount = 1;
        initialProposalId = proposalCount;
        timelock.acceptAdmin();
    }
```

In `GovernorBravoInterfaces.sol`, delete `GovernorAlpha` Interface:

- Delete lines 179-182

## Testing Notes

- MINIMUM_DELAY in Timelock.sol set to 0 to allow for timely testing
- MIN_VOTING_PERIOD in GovernorBravoDelegate.sol set to 0 to allow for timely testing


# Deploying on Rinkeby-Boba Network and Initiating


Instructions for deploying Compound Governance Protocol on Rinkeby-Boba.
First create a `.env` file that follows the structure of `.env.example`.

```bash
$ yarn
$ yarn compile:ovm
```

You should expect the following output:

```bash
yarn run v1.22.10
$ truffle migrate --network rinkeby_l2 --config truffle-config-ovm.js

Compiling your contracts...
===========================
> Everything is up to date, there is nothing to compile.



Starting migrations...
======================
> Network name:    'rinkeby_l2'
> Network id:      28
> Block gas limit: 11000000 (0xa7d8c0)


1_migration.js
==============
STARTING HERE
0x21A235cf690798ee052f54888297Ad8F46D3F389

   Replacing 'Comp'
   ----------------
   > transaction hash:    0xff48ed5d650f00e1cd4222a637a1b33b84303c8f90915183b4d76f4f00a9f214
   > Blocks: 0            Seconds: 0
   > contract address:    0x286b85cAcc1dca2AdA813a72De696de141a99bE8
   > block number:        23252
   > block timestamp:     1630690954
   > account:             0x21A235cf690798ee052f54888297Ad8F46D3F389
   > balance:             0.645055149168
   > gas used:            3979070 (0x3cb73e)
   > gas price:           0.015 gwei
   > value sent:          0 ETH
   > total cost:          0.00005968605 ETH

deployed comp

   Replacing 'Timelock'
   --------------------
   > transaction hash:    0xcd6784889f0d673b1490448117eff72b53f7c5c29cd00e5d9042a05f3e61eeb6
   > Blocks: 0            Seconds: 0
   > contract address:    0xC44D6745a1e0Fd5456646E0f05EE4704b283E6B0
   > block number:        23253
   > block timestamp:     1630690954
   > account:             0x21A235cf690798ee052f54888297Ad8F46D3F389
   > balance:             0.632996649168
   > gas used:            3512614 (0x359926)
   > gas price:           0.015 gwei
   > value sent:          0 ETH
   > total cost:          0.00005268921 ETH

deployed timelock

   Replacing 'GovernorBravoDelegate'
   ---------------------------------
   > transaction hash:    0xb0c26ff5fae425b5853bbb31b8ad9285a791589652f975da5d6f50ef5be2422a
   > Blocks: 0            Seconds: 0
   > contract address:    0x1a078ae5651591BA4A5c447D29eA68D44Bf30f62
   > block number:        23254
   > block timestamp:     1630690954
   > account:             0x21A235cf690798ee052f54888297Ad8F46D3F389
   > balance:             0.620938149168
   > gas used:            8720274 (0x850f92)
   > gas price:           0.015 gwei
   > value sent:          0 ETH
   > total cost:          0.00013080411 ETH

deployed delegate

   Replacing 'GovernorBravoDelegator'
   ----------------------------------
   > transaction hash:    0x7447d9db393c5066f1c17946d19e9ee06dfcb4a1180d7e3bc4efda56b2c04144
   > Blocks: 0            Seconds: 0
   > contract address:    0xCD7239aeCBc66b1A77D5b19e7CF00380fA9Bf529
   > block number:        23255
   > block timestamp:     1630690954
   > account:             0x21A235cf690798ee052f54888297Ad8F46D3F389
   > balance:             0.608879649168
   > gas used:            2044985 (0x1f3439)
   > gas price:           0.015 gwei
   > value sent:          0 ETH
   > total cost:          0.000030674775 ETH

deployed delegator
Queue setPendingAdmin
Time transaction was made: 1630690954
Time at which transaction may be executed: 1630691254
Attempt: 1
	Timestamp: 1630690954
	Transaction hasn't surpassed time lock

Attempt: 2
	Timestamp: 1630691149
	Transaction hasn't surpassed time lock

Attempt: 3
	Timestamp: 1630691149
	Transaction hasn't surpassed time lock

Attempt: 4
	Timestamp: 1630691149
	Transaction hasn't surpassed time lock

Attempt: 5
	Timestamp: 1630691149
	Transaction hasn't surpassed time lock

Attempt: 6
	Timestamp: 1630691149
	executed setPendingAdmin
Current time: 1630691344
Time at which transaction can be executed: 1630691644
queued initiate
execute initiate
Timestamp: 1630691344
	Transaction hasn't surpassed time lock

Timestamp: 1630691539
	Transaction hasn't surpassed time lock

Timestamp: 1630691539
	Transaction hasn't surpassed time lock

Timestamp: 1630691539
	Transaction hasn't surpassed time lock

Timestamp: 1630691539
Executed initiate
   > Saving artifacts
   -------------------------------------
   > Total cost:      0.000273854145 ETH


Summary
=======
> Total deployments:   4
> Final cost:          0.000273854145 ETH


✨  Done in 768.54s.
```

# Using Compound Governance Protocol
This section will guide you in delegating votes, submitting a poroposal, voting on it, queueing it and executing it. The files in the `scripts` folder can be used to accomplish all of these tasks and more.


First paste the contract addresses, generated from the previous section, into the file `networks/rinkeby-l2.json`. Using the addresses above the file should look as follows.

```json
{
"Comp":"0x286b85cAcc1dca2AdA813a72De696de141a99bE8",
"Timelock":"0xC44D6745a1e0Fd5456646E0f05EE4704b283E6B0",
"GovernorBravoDelegate":"0x1a078ae5651591BA4A5c447D29eA68D44Bf30f62",
"GovernorBravoDelegator":"0xCD7239aeCBc66b1A77D5b19e7CF00380fA9Bf529"
}
```

## Delegating Votes

First Comp must be trasnferred to other entities so that they may have voting power. Then these entities can delegate votes.
The file `scripts/delegateVotes.js` accomplishes this goal.
Run the following command.

```bash
$ yarn delegateVotes
```
You should expect output similar to the following:

```bash
yarn run v1.22.10
$ node scripts/delegateVotes.js
Wallet1: Comp power:  10000000000000000000000000
Wallet2: Comp power:  1000000000000000000000000
Wallet3: Comp power:  1000000000000000000000000
wallet1 current votes:  8000000000000000000000000
wallet2 current votes:  1000000000000000000000000
wallet3 current votes:  1000000000000000000000000
Wait 5 minutes to make sure votes are processed.
✨  Done in 405.07s.
```

## Submitting a Proposal

After the voting power has been allocated proposals can be submitted. The proposal to be submitted by `scripts/submitProposal.js` is to reduce the number of votes necessary to make a proposal to 65000 votes.
Run the following command.

```bash
$ yarn submitProposal
```
You should expect output similar to the following:
```bash
```

## Casting Votes
After a proposal has been submitted votes must cast during the voting period. If enough votes are in favor of the proposal, then the proposal can be queued and executed. Votes will be cast by `scripts/castVotes.js`.


Run the following command.

```bash
$ yarn castVotes
```
You should expect output similar to the following:
```bash
yarn run v1.22.10
$ node scripts/castVotes.js
Proposed. Proposal ID: 0x03
State of Proposal 0x03 is : Pending
Casting Votes
Attempt: 1
	State of Proposal 0x03 is : Pending
	Voting is closed

Attempt: 2
	State of Proposal 0x03 is : Pending
	Voting is closed

Attempt: 3
	State of Proposal 0x03 is : Active
	Success: vote cast by wallet1
	Success: vote cast by wallet2
	Success: vote cast by wallet3
Waiting for voting period to end.
✨  Done in 206.16s.
```

## Queueing a Proposal
After the voting period has ended a proposal can be queued if it has succeed, if it has recieved enough votes in favor. The proposal can be queued using `scripts/queueProposal.js`.

Run the following command.

```bash
$ yarn queueProposal
```
You should expect output similar to the following:
```bash
yarn run v1.22.10
$ node scripts/queueProposal.js
Proposed. Proposal ID: 0x03
Queuing Proposal
Attempt: 1
Success: Queued
State is :  Queued
✨  Done in 5.44s.

```

## Executing a Proposal
The last step is to execute the proposal, this means that the proposal will take effect. This can only happen after the proposal has been queued.

Run the following command.

```bash
$ yarn executeProposal
```
You should expect output similar to the following:
```bash
yarn run v1.22.10
$ node scripts/executeProposal.js
Proposed. Proposal ID: 0x03
Executing Proposal
Attempt: 1
Success: Executed
3
State is :  Executed
[["0x19e824199b5B13D33561c85978560E82F3D07106"],[{"type":"BigNumber","hex":"0x00"}],["_setProposalThreshold(uint256)"],["0x000000000000000000000000000000000000000000000dc3a8351f3d86a00000"]]
BlockNum :  23342
Proposal Threshold :  65000000000000000000000
proposalId :  0x03
✨  Done in 6.26s.
```
Congratulations! You have successfuly executed a proposal on a Decentralized Autonomous Organization!

## Canceling a Proposal
If at any point you wish to cancel a proposal you can cancel it by using the proposal id. Only the entity that proposed a proposal can cancel the proposal. The proposal can be canceled with `scripts/cancelProposal.js`.

In order to cancel a proposal first change line 33 of `scripts/cancelProposal.js`.

```js
const proposalID = 1; // proposal to cancel
```

Then run the following command.

```bash
$ yarn cancelProposal
```



