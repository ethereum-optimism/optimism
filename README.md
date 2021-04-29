# Integration Tests

Typescript based integration test repo for OMGX.

- [1. Basic Setup and Configuration](#1-basic-setup-and-configuration)
  * [1.1 Using Rinkeby Testnet](#11-using-rinkeby-testnet)
  * [1.2 Using Local Net](#12-using-local-net)
- [2. Running the Integration Tests](#2-running-the-integration-tests)
- [3. Wallet Specific Smart Contracts](#3-wallet-specific-smart-contracts)
  * [3.1 L1liquidityPool.sol](#31-l1liquiditypoolsol)
    + [Known Gaps/Problems](#known-gaps/problems)
    + [Initial values](#initial-values)
    + [Events](#events)
    + [Functions](#functions)
  * [3.2 L2liquidityPool.sol](#32-l2liquiditypoolsol)
  * [3.3 Deploy Liquidity Pools](#33-deploy-liquidity-pools)
  * [3.4 AtomicSwap](#34-atomicswap)
    + [Functions](#functions-1)

If you want to run these locally, you need a local OMGX system. 

## 1. Basic Setup and Configuration

Create a `.env` file in the root directory of this project. Add environment-specific variables on new lines in the form of `NAME=VALUE`. 

### 1.1 Using Rinkeby Testnet

If you just want to work on the wallet, you can use the stable testnet on Rinkeby and AWS. To test on Rinkeby (ChainID 4), you will need an Infura key and two accounts with Rinkeby ETH in them. The test wallets must contain enough ETH to cover the tests. The `.env` parameters are

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

### 1.2 Using Local Net

If would like to change the wallet-associated smart contracts and/or work on other aspects of the system, you should run a local development system. You can do this by

```bash
# Git clone with submodules
$ git clone git@github.com:enyalabs/optimism-integration.git
$ cd optimism-integration
$ docker-compose pull
$ yarn install
$ ./up_local.sh
```

If you do not know what these values are, you can get them from the values written to the terminal as your local OMGX spins up. 


As the system boots, you'll see several things zip by in the terminal that you will need to correctly configure your `.env`, namely the values for the

```bash
ETH1_ADDRESS_RESOLVER_ADDRESS=0x______
TEST_PRIVATE_KEY_1=0x_____
TEST_PRIVATE_KEY_2=0x______
```

For the test private keys, we normally use the ones generated by hardhat as it spins up the local L1. Each of those is funded with 1000 ETH. You can get the test private keys from the hardhat (`l1_chain`) Docker terminal or from your main terminal. You will also need the `ADDRESS_RESOLVER_ADDRESS`, which will zip by as your local system deploys (see the `deployer` Docker terminal output or your main terminal window). Fill these values into your `.env`. Your final `.env` should look something like this

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

## 2. Running the Integration Tests

```bash
$ yarn install
$ yarn build #builds the contracts
$ yarn deploy #if needed - this will test and deploy the contracts, and write their addresses to /deployments/addresses.json
```

The information generated during the deploy (e.g the `/deployments/addresses.json`) is used by the web wallet front end to set things up correctly. **The full test suite includes some very slow transactions such as withdrawals, which can take 300 seconds each. Please be patient.**

## 3. Wallet Specific Smart Contracts

These contracts instantiate a simple swap on/off system for fast entry/exit, as well as some basics such as support for atomic swaps.

### 3.1 L1liquidityPool.sol

The Layer 1 liquidity pool accepts ERC20 and ETH. `L1liquidityPool.sol` charges a convenience fee to the user for quickly getting on to the L2.

**L1->L2**: When users **deposit into this contract**, then (1) the pool size grows and (2) corresponding funds are sent to them on the L2 side.  

**L2->L1**: When users **deposit into the corresponding L2 contract**, then (1) the pool size shrinks and (2) corresponding tokens are sent to them at their L1 wallet (minus the fee). 

#### Known Gaps/Problems

- [ ] The contract owner can't currently get funds back out of the liquidy pool.
- [ ] Fee calculations are unsafe.
- [ ] Need to verify correct usage of the withdraw/transfer patterns.
- [ ] Need to verify correct access limitations.
- [ ] Need system to allow _others_ to add liquidity and pay them for their liquidity.

#### Initial values

* _l2LiquidityPoolAddress_. The address of the Layer 2 liquidity pool 
* _l1messenger_. The address of the Layer 1 messenger  
* _l2ETHAddress_. The address of the oWETH contract on the L2 
* _fee_. The convenience fee. The data type of `_fee` is `uint256`. If the fee is 3%, then `_fee` is 3.

#### Events

* `ownerAddERC20Liquidity_EVENT`. The contract owner just added funds to the liquidity pool.
* `clientDepositL1_EVENT`. A user just deposited tokens to the pool. `clientDepositL1` sends a message to L2, which triggers an L2 contract to send funds to the user's L2 wallet.
* `clientPayL1_EVENT`. A user deposited tokens into the system on the L2 side, which then called `clientPayL1` to send the tokens to the user's L1 account. `clientPayL1` can only be accessed by a contract on the L2 side.
* `ownerRecoverFee_EVENT`. The contract owner just withdrew fees.

#### Functions

* `init`. Can only be accessed by the contract owner. The owner can update the `_fee`.
* `receive`. This handles ETH. If called by the contract owner, it allows ETH to be deposited into the ETH pool. For other callers, it also sends a message to the `L2liquidityPool` smart contract on L2, which then sends oWETH to the sender.
* `ownerAddERC20Liquidity`. Used by the owner to provide liquidity into an ERC20 pool. Unlike a normal deposit, it doesn't send a message to L2.
* ` balanceOf` returns the balance of ERC20 or ETH of this contract. The default address of ETH is `address(0)`.
* ` feeBalanceOf` returns the fee balance of ERC20 or ETH of this contract.
* ` clientDepositL1` Users call this function to deposit tokens. After receiving the deposit, the contract sends a cross-domain message to L2. The `L2liquidityPool` sends the corresponding tokens to the user.
* ` clientPayL1` This cross-chain function can only be accessed `onlyFromCrossDomainAccount`. It can't be accessed by any users. When the layer 2 liquidity pool receives tokens and sends a message to L1, `clientPayL1` sends the token to the user and charges a convenience fee.
* ` ownerRecoverFee` can only be accessed by the contract owner. The contract owner can withdraw fees as they accumulate.

### 3.2 L2liquidityPool.sol

Just like the contract for L1, but with small changes e.g. to deal with the fact that the L2 does not have native ETH.

### 3.3 Deploy Liquidity Pools

If you are working on a local testnet, please deploy **L2LiquidityPool.sol** first, then use the address of **L2LiquidityPool** as the parameter for deploying the **L1Liquidity**. See **/test/setup_test.spec.ts** to see the whole process.

```javascript

    L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
      env.watcher.l2.messengerAddress,
    )
    await L2LiquidityPool.deployTransaction.wait()
    
    L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
      L2LiquidityPool.address,
      env.watcher.l1.messengerAddress,
      env.L2ETHGateway.address,
      3
    )
    await L1LiquidityPool.deployTransaction.wait()
    
    const L2LiquidityPoolTX = await L2LiquidityPool.init(L1LiquidityPool.address, "3")
    await L2LiquidityPoolTX.wait()
    console.log(' L2 LP initialized:',L2LiquidityPoolTX.hash);

```

### 3.4 AtomicSwap

Used to swap ERC20 tokens.

#### Functions

* `open` creates **Swap** struct, which contains the information of the buyer and sender.
* `close` closes the swap. It swaps the ERC20 tokens of the sender and the buyer.
* `expire` sets the status of the swap to be **EXPIRED**.
* `check` returns the **Swap** construct

## 4. Running the web wallet

```bash
$ cd /wallet 
$ yarn install
$ yarn start
```

You will need to set up MetaMask to know about the two accounts you are using for local testing. You will need to point MetaMask at your local chains (at :9545 and :8545) and add the account.

```bash
TEST_PRIVATE_KEY=0x23d9aeeaa08ab710a57972eb56fc711d9ab13afdecc92c89586e0150bfa380a6
```

