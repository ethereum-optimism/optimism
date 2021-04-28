# Integration Tests

Typescript based integration test repo for OMGX.

## CONFIGURATION

Create a `.env` file in the root directory of this project. Add environment-specific variables on new lines in the form of `NAME=VALUE`.

To test on Rinkeby, ChainID4, you will need an Infura key and two accounts with Rinkeby ETH in them. The text wallets must contain enough ETH to cover the tests. 

**The full test suite includes some very slow transactions such as withdrawls, which can take 300 seconds each. Please be patient.**

## PERFORM THE TESTS

```bash
$ yarn install
$ yarn build
$ yarn deploy #if needed - this will test and deploy the contracts, and write their addresses to /deployments/addresses.json
```


## Test local

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

## Test Rinkeby

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

## PERFORM THE TESTS

```bash
$ yarn install
$ yarn build:integration
$ yarn test:integration
```

# Wallet Specific Smart Contracts

## L1liquidityPool.sol

Layer 1 liquidity pool. It accepts ERC20 and ETH. When users deposit into the Layer 2 liquidity pool, `L1liquidityPool.sol` sends tokens to the user on L1 and charges a convenience fee to the user.

### Initial values

* _l2LiquidityPoolAddress_ The address of the Layer 2 liquidity pool 
* _l1messenger_ The address of the Layer 1 messager  
* _l2ETHAddress_ The address of the oWETH  
* _fee_ The convenience fee. The data type of **_fee** is `uint256`. If the fee is 3%, then _fee_ is 3.

### Events

* initiateDepositedTo

  The event of adding funds to the pool by the contract owner. **initiateDepositTo** doesn't send the message to L2. 

* depositedTo

  The event of depositing tokens to the pool. **depositTo** sends the message to L2.

* depositedToFinalized

  The event of sending tokens to the user. **depositToFinalize** is the cross-chain function.

* withdrewFee

  The event of withdrawing fees by the contract owner.

### Functions

#### init

It can only be accessed by the contract owner. The owner can update the **_fee**.

#### receiver

For the contract owner, it doesn't send the message to L2 when the contracts receive the ETH.

For other addresses, it sends the message to L2. The **L2liquidityPool** smart contract on L2 sends **oWETH** to the sender.

#### balanceOf

It returns the balance of ERC20 or ETH of this contract. The default address of ETH is **address(0)**.

#### feeBalanceOf

It returns the fee balance of ERC20 or ETH of this contract.

#### initiateDepositTo

It is used to add tokens to this pool, it doesn't send the message to L2.

#### depositTo

Users call this function to deposit tokens. After successfully receiving the funds, it sends the message to L2. The **L2liquidityPool** sends the corresponding tokens to the user.

#### withdrawFee

It can only be accessed by the contract owner. The contract owner can withdraw the fee and send the fee to others.

#### depositToFinalize

It's the cross-chain function. It can't be accessed by any users. When the layer 2 liquidity pool receives the tokens and sends the message to layer 1,  **depositToFinalize** sends the token to the user and takes the convenience fee.

## L2liquidityPool.sol

> Layer 2 liquidity pool
>
> It accepts ERC20. When users deposit on the Layer 2 liquidity pool, it sends tokens to the user and takes the convenience fee from the user.

### Initial values

****

* _l2CrossDomainMessenger_

  The address of the Layer 2 messenger.

### Event

****

* initiateDepositedTo

  The event of adding funds to the pool by the contract owner. **initiateDepositTo** doesn't send the message to L2. 

* depositedTo

  The event of depositing tokens to the pool. **depositTo** sends the message to L2.

* depositedToFinalized

  The event of sending tokens to the user. **depositToFinalize** is the cross-chain function.

* withdrewFee

  The event of withdrawing fees by the contract owner.

### Function

****

#### init

It can only be accessed by the contract owner. The owner can update the **_fee** and **_L1LiquidityPoolAddress**.

> It must be called after deploying this contract. Otherwise, you can use other functions.

#### balanceOf

It returns the balance of ERC20 of this contract.

#### feeBalanceOf

It returns the fee balance of ERC20 of this contract.

#### initiateDepositTo

It is used to add tokens to this pool, it doesn't send the message to L2.

#### depositTo

Users call this function to deposit tokens. After successfully receiving the funds, it sends the message to L1. The **L1liquidityPool** sends the corresponding tokens to the user.

#### withdrawFee

It can only be accessed by the contract owner. The contract owner can withdraw the fee and send the fee to others.

#### depositToFinalize

It's the cross-chain function. It can't be accessed by any users. When the layer 1 liquidity pool receives the tokens and sends the message to layer 2,  **depositToFinalize** sends the token to the user and takes the convenience fee.

## AtomicSwap

> It's used to swap ERC20 tokens.

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

# Deploy

> Please deploy **L2LiquidityPool.sol** first, then use the address of **L2LiquidityPool** ad the parameter of deploying the **L1Liquidity**.
>
> Please review **oe-deploy.js** to see the whole process.

```javascript
// deploy L2 liquidity pool
const L2_LP = await deploo({contractName: "L2LiquidityPool", rpcUrl: selectedNetwork.l2RpcUrl, ovm: true, _args: [l2MessengerAddress]});
  
// deploy L1 liquidity pool
const L1_LP = await deploy({contractName: "L1LiquidityPool", rpcUrl: selectedNetwork.l1RpcUrl, ovm: false, _args: [L2_LP.address, l1MessengerAddress, l2ETHAddress, 3]});

// initialize the L2 liquidity pool
const initL2LP = await L2_LP.init(L1_LP.address, "3");
await initL2LP.wait();
```

