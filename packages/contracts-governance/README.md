<div align="center">
  <a href="https://community.optimism.io"><img alt="Optimism" src="https://user-images.githubusercontent.com/14298799/122151157-0b197500-ce2d-11eb-89d8-6240e3ebe130.png" width=280></a>
  <br />
  <h1> Optimism Governance Contracts</h1>
</div>

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=contracts-governance-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

## TL;DR

The token and governance smart contracts for the Optimism DAO. Built using [OpenZeppelin libraries](https://docs.openzeppelin.com/contracts/4.x/) with some customisations. The token is an [ERC20](https://docs.openzeppelin.com/contracts/4.x/api/token/erc20) that is [permissible](https://docs.openzeppelin.com/contracts/4.x/api/token/erc20#ERC20Permit) and allows for [delegate voting](https://docs.openzeppelin.com/contracts/4.x/api/token/erc20#ERC20Votes). The token is also [burnable](https://docs.openzeppelin.com/contracts/4.x/api/token/erc20#ERC20Burnable). See more in the [Specification section](#specification).

Governance will initially be handled by [Snapshot](https://snapshot.org/#/) before moving to an on chain governance system like [OpenZeppelins Governance contracts](https://docs.openzeppelin.com/contracts/4.x/api/governance).

## Getting set up

### Requirements
You will need the following dependancies installed:
```
nvm
node
yarn
npx
```

Instal the required packages by running:
```
nvm use
yarn
```
#### Compile

To compile the smart contracts run:
```
yarn build
```

#### Test

To run the tests run:
```
yarn test
```

#### Lint

To run the linter run:
```
yarn lint
```

#### Coverage
For coverage run:
```
yarn test:coverage
```

#### Deploying

To deploy the contracts you will first need to set up the environment variables.

Duplicate the [`.env.example`](./.env.example) file. Rename the duplicate to `.env`.

Fill in the missing environment variables, take care with the specified required formatting of secrets.

Then run the command for your desired network:
```
# To deploy on Optimism Kovan
yarn deploy-op-kovan

# To deploy on Optimism
yarn deploy-op-main
```

---

## Specification

Below we will cover the specifications for the various elements of this repository.

### Governance Token

The [`GovernanceToken.sol`](./contracts/GovernanceToken.sol) contract is a basic ERC20 token, with the following modifications:

* ✅ **Non-upgradable**
    * This token is not upgradable.
* ✅ **Ownable**
    * This token has an owner role to allow for permissioned minting functionality.
* ✅ **Mintable**
    * The `OP` token is an inflationary token. We allow for up to 2% annual inflation supply to be minted by the token `MintManager`.
* ✅ **Burnable**
    * The token allows for tokens to be burnt, as well as allowing approved spenders to burn tokens from users.
* 🛠 **Permittable**
    * This token is permittable as defined by [EIP2612](https://eips.ethereum.org/EIPS/eip-2612). This allows users to approve a spender without submitting an onchain transaction through the use of signed messages.
* **Delegate voting**
    * This token inherits Open Zeppelins ERC20Votes.sol to allow users to delegate voting power. This requires the token be permittable.

### Mint Manager

The [`MintManager.sol`](./contracts/MintManager.sol) contract is set as the `owner` of the OP token and is responsible for the token inflation schedule. It acts as the token "mint manager" with permission to the `mint` function only.
The current implementation allows minting once per year of up to 2% of the total token supply.

The contract is also upgradable to allow changes in the inflation schedule.

### Snapshot Voting Strategy

(WIP)

### Governance (DAO) Contracts

(WIP)
