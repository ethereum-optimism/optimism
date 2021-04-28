# Integration Tests

Typescript based integration test repo for OMGX.

- [Integration Tests](#integration-tests)
        * [Table of Contents](#table-of-contents)
  * [Headers](#headers)
  * [1. Basic Configuration](#1-basic-configuration)
    + [Test Local](#test-local)
    + [Test Rinkeby](#test-rinkeby)
  * [PERFORM THE TESTS](#perform-the-tests)
  * [Wallet Specific Smart Contracts](#wallet-specific-smart-contracts)
  * [L1liquidityPool.sol](#l1liquiditypoolsol)
    + [Initial values](#initial-values)
    + [Events](#events)
    + [Functions](#functions)
      - [init](#init)
      - [receive](#receive)
      - [ownerAddERC20Liquidity](#owneradderc20liquidity)
      - [balanceOf](#balanceof)
      - [feeBalanceOf](#feebalanceof)
      - [clientDepositL1](#clientdepositl1)
      - [clientPayL1](#clientpayl1)
      - [ownerRecoverFee](#ownerrecoverfee)
  * [L2liquidityPool.sol](#l2liquiditypoolsol)
    + [Deploy Liquidity Pools (probably outdated)](#deploy-liquidity-pools--probably-outdated-)
  * [AtomicSwap](#atomicswap)
    + [Function](#function)
      - [open](#open)
      - [close](#close)
      - [expire](#expire)
      - [check](#check)

<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>

## 1. Basic Configuration

Create a `.env` file in the root directory of this project. Add environment-specific variables on new lines in the form of `NAME=VALUE`.

### Test Local

```bash
NODE_ENV=local
L1_NODE_WEB3_URL=http://localhost:9545
L2_NODE_WEB3_URL=http://localhost:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0x3e4CFaa8730092552d9425575E49bB542e329981
TEST_PRIVATE_KEY_1=0x754fde3f5e60ef2c7649061e06957c29017fe21032a8017132c0078e37f6193a
TEST_PRIVATE_KEY_2=0x23d9aeeaa08ab710a57972eb56fc711d9ab13afdecc92c89586e0150bfa380a6
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=420
```

### Test Rinkeby

```bash
NODE_ENV=local
L1_NODE_WEB3_URL=https://rinkeby.infura.io/v3/KEY
L2_NODE_WEB3_URL=http://54.161.5.63:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0xa32cf2433ba24595d3aCE5cc9A7079d3f1CC5E0c
TEST_PRIVATE_KEY_1=0xPRIVATE KEY OF THE FIRST TEST WALLET
TEST_PRIVATE_KEY_2=0xPRIVATE KEY OF THE SECOND TEST WALLET
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=420
```

To test on Rinkeby, ChainID4, you will need an Infura key and two accounts with Rinkeby ETH in them. The test wallets must contain enough ETH to cover the tests. **The full test suite includes some very slow transactions such as withdrawls, which can take 300 seconds each. Please be patient.**

## 2. Running the Integration Tests

```bash
$ yarn install
$ yarn build
$ yarn deploy #if needed - this will test and deploy the contracts, and write their addresses to /deployments/addresses.json
```

## 3. Wallet Specific Smart Contracts

These contracts instnatiate a simple swap on/off system for fast entry/exit, as well as some basics such as XXXXXXXXXXXXX

### 3.1 L1liquidityPool.sol

The Layer 1 liquidity pool accepts ERC20 and ETH. 

**L1->L2**: When users **deposit into this contract**, then (1) the pool size grows and (2) corresponding funds are sent to them on the L2 side.  

**L2->L1**: When users **deposit into the corresponding L2 contract**, then (1) the pool size shrinks and (2) corresponding tokens are sent to them at their L1 wallet. `L1liquidityPool.sol` charges a convenience fee to the user.   

#### Initial values

* _l2LiquidityPoolAddress_. The address of the Layer 2 liquidity pool 
* _l1messenger_. The address of the Layer 1 messager  
* _l2ETHAddress_. The address of the oWETH contract on the L2 
* _fee_ The convenience fee. The data type of **_fee** is `uint256`. If the fee is 3%, then _fee_ is 3. (This needs to be improved)

#### Events

* `ownerAddERC20Liquidity_EVENT`. The event of adding funds to the pool by the contract owner. `ownerAddERC20Liquidity` doesn't send any messages to L2. 
* `clientDepositL1_EVENT`. The event of depositing tokens to the pool. `clientDepositL1` sends a message to L2, which triggers a contract on the L2 side to send funds to the user's L2 wallet.
* `clientPayL1_EVENT`. The event of sending tokens to the user. `clientPayL1` is a cross-chain function - it's triggered by actions on the L2 side, which then call clientPayL1 to send funds to the user's L1 accounut.
* `ownerRecoverFee_EVENT`. The event of withdrawing fees by the contract owner.

#### Functions

* `init`. It can only be accessed by the contract owner. The owner can update the **_fee**.
* `receive`. This handles ETH. If called by the contract owner, it allows ETH to be desposited into the ETH pool. For other callers, it also sends a message to the `L2liquidityPool` smart contract on L2, which then sends oWETH to the sender.
* `ownerAddERC20Liquidity`. Used by the owner to provide liquidity into an ERC20 pool. Unlike a normal deposit, it doesn't send a message to L2.

#### balanceOf

It returns the balance of ERC20 or ETH of this contract. The default address of ETH is **address(0)**.

#### feeBalanceOf

It returns the fee balance of ERC20 or ETH of this contract.

#### clientDepositL1

Users call this function to deposit tokens. After receiving the deposit, the contract sends a cross-domain message to L2. The **L2liquidityPool** sends the corresponding tokens to the user.

#### clientPayL1

This cross-chain function can only be accessed `onlyFromCrossDomainAccount`. It can't be accessed by any users. When the layer 2 liquidity pool receives tokens and sends a message to L1,  **clientPayL1** sends the token to the user and charges a convenience fee.

#### ownerRecoverFee

It can only be accessed by the contract owner. The contract owner can withdraw fees as they accumulate.

## L2liquidityPool.sol

Just like the contract for L1, but with small changes e.g. to deal with the fact that the L2 does not have native ETH.

### Deploy Liquidity Pools (probably outdated)

> Please deploy **L2LiquidityPool.sol** first, then use the address of **L2LiquidityPool** ad the parameter of deploying the **L1Liquidity**.
>
> Please review **oe-deploy.js** to see the whole process.

```javascript
// deploy L2 liquidity pool
const L2_LP = await deploy({contractName: "L2LiquidityPool", rpcUrl: selectedNetwork.l2RpcUrl, ovm: true, _args: [l2MessengerAddress]});
  
// deploy L1 liquidity pool
const L1_LP = await deploy({contractName: "L1LiquidityPool", rpcUrl: selectedNetwork.l1RpcUrl, ovm: false, _args: [L2_LP.address, l1MessengerAddress, l2ETHAddress, 3]});

// initialize the L2 liquidity pool
const initL2LP = await L2_LP.init(L1_LP.address, "3");
await initL2LP.wait();
```

## AtomicSwap

Used to swap ERC20 tokens.

### Function

****

#### open

It creates **Swap** struct, which contains the information of the buyer and sender.

#### close

It closes the swap. It swaps the ERC20 tokens of the sender and the buyer.

#### expire

It sets the status of the swap to be **EXPIRED**.

#### check

It returns the **Swap** contruct



