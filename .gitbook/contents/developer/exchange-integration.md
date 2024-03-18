---
description: >-
  Boba Network docs for Exchanges looking to integrate deposits and withdrawals
  with Boba Network
---

# Exchange Integration



<figure><img src="../../.gitbook/assets/bridging basics.png" alt=""><figcaption></figcaption></figure>

Although Boba Network is an L2 (and therefore fundamentally connected to Ethereum), it's also a separate blockchain. App developers commonly need to move data and assets between Boba Network and Ethereum. We call the process of moving data and assets between the two networks "bridging".

### Sending tokens between L1 and L2

For the most common usecase, moving tokens around, we've created the Standard Token Bridge. The Standard Token Bridge is a simple smart contract with all the functionality you need to move tokens between Boba Network and Ethereum.

Beside the Standard Token Bridge, we created the Fast Token Bridge to allow you to exit assets from L2 in serval hours or even serval minutes based on the number of transactions. The Fast Token Bridge collects a certain percentage of the deposit amount as the transaction fee and distributes them to the liquidity providers.



<figure><img src="../../.gitbook/assets/using the standard token bridge.png" alt=""><figcaption></figcaption></figure>

The standard bridge functionality provides a method for an ERC20 token to be deposited and locked on L1 in exchange of the same amount of an equivalent token on L2. This process is known as "bridging a token", e.g. depositing 100 BOBA on L1 in exchange for 100 BOBA on L2 and also the reverse - withdrawing 100 BOBA on L2 in exchange for the same amount on L1. In addition to bridging tokens the standard bridge is also used for ETH.

The Standard Bridge is composed of two main contracts the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts/contracts/L1/messaging/IL1StandardBridge.sol)(for Layer 1) and the [`L2StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts/contracts/L2/messaging/L2StandardBridge.sol)(for Layer 2).

Here we'll go over the basics of using this bridge to move ERC20 and ETH assets between Layer 1 and Layer 2.

### Deposits

> Note: **We currently block smart contract wallets from calling the `depositETH` and `depositERC20` functions for security reasons**. If you want to deposit not using an EOA accounts and you know what are doing, you can use `depositETHTo` and `depositERC20To` functions instead.

#### Deposit ERC20s

ERC20 deposits into L2 can be triggered via the `depositERC20` and `depositERC20To` functions on the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts/contracts/L1/messaging/IL1StandardBridge.sol). You **must** approve the Standard Token Bridge to use the amount of tokens that you want to deposit or the deposit will fail.

```
const PRIVATE_KEY, L1_NODE_WEB3_URL, PROXY_L1_STANDARD_BRIDGE_ADDRESS

const L1Provider = new ethers.providers.StaticJsonRpcProvider(L1_NODE_WEB3_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L1Provider)

const Proxy__L1StandardBridge = new ethers.Contract(
  PROXY_L1_STANDARD_BRIDGE_ADDRESS,
  L1StandardBridgeABI,
  L1Wallet
)

// Approve amounts
const approveTx = await L1ERC20Contract.approve(Proxy__L1StandardBridge.address, depositAmount)
await approveTx.wait()

// Deposit ERC20
const depositTx = await Proxy__L1StandardBridge.depositERC20(
  l1TokenAddress,
  l2TokenAddress,
  depositAmount,
  1300000, // l2 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()) // byte data
)
await depositTx.wait()

// Deposit ERC20 to another l2 wallet
const depositToTx = await Proxy__L1StandardBridge.depositERC20To(
  l1TokenAddress,
  l2TokenAddress,
  TargetAddress, // l2 target address
  depositAmount,
  1300000, // l2 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()) // byte data
)
await depositToTx.wait()
```

#### Deposit ETH

ETH deposits into L2 can be triggered via the `depositETH` and `depositETHTo` functions on the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts/contracts/L1/messaging/IL1StandardBridge.sol). ETH deposits can alternatively be triggered by sending ETH directly to the `L1StandardBridge`. Once your deposit is detected and finalized on Boba Network, your account will be funded with the corresponding amount of ETH on L2.

```
const PRIVATE_KEY, L1_NODE_WEB3_URL, PROXY_L1_STANDARD_BRIDGE_ADDRESS

const L1Provider = new ethers.providers.StaticJsonRpcProvider(L1_NODE_WEB3_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L1Provider)

const Proxy__L1StandardBridge = new ethers.Contract(
  PROXY_L1_STANDARD_BRIDGE_ADDRESS,
  L1StandardBridgeABI,
  L1Wallet
)

// Deposit ETH
const depositTx = await Proxy__L1StandardBridge.depositETH(
  1300000, // l2 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await depositTx.wait()

// Deposit ETH to another l2 wallet
const depositToTx = await Proxy__L1StandardBridge.depositETHTo(
  TargetAddress, // l2 target address
  depositAmount,
  1300000, // l2 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await depositToTx.wait()
```

### Withdrawals

#### Withdraw tokens (ERC20s and ETH)

ERC20 withdrawals can be triggered via the `withdraw` or `withdrawTo` functions on the [`L2StandardBridge`](https://github.com/bobanetwork/boba/blob/develop/packages/contracts/contracts/L2/messaging/L2StandardBridge.sol)

```
const PRIVATE_KEY, L2_NODE_WEB3_URL, PROXY_L2_STANDARD_BRIDGE_ADDRESS

const L2Provider = new ethers.providers.StaticJsonRpcProvider(L2_NODE_WEB3_URL)
const L2Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L2Provider)

const Proxy__L2StandardBridge = new ethers.Contract(
  PROXY_L2_STANDARD_BRIDGE_ADDRESS,
  L2StandardBridgeABI,
  L2Wallet
)

// Withdraw ETH
// ETH address is 0x4200000000000000000000000000000000000006 on L2
const depositTx = await Proxy__L2StandardBridge.withdraw(
  '0x4200000000000000000000000000000000000006', // l2 token address
  ETHAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await depositTx.wait()

// Withdraw ETH to another l1 wallet
// ETH address is 0x4200000000000000000000000000000000000006 on L2
const depositToTx = await Proxy__L2StandardBridge.withdrawTo(
  '0x4200000000000000000000000000000000000006', // l2 token address
  TargetAddress, // l1 target address
  ETHAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await depositToTx.wait()

// Approve amounts
const approveTx = await L2ERC20Contract.approve(Proxy__L2StandardBridge.address, exitAmount)
await approveTx.wait()

// Withdraw ERC20
const depositTx = await Proxy__L2StandardBridge.withdraw(
  l2TokenAddress // l2 token address
  exitAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
)
await depositTx.wait()

// Withdraw ERC20 to another l1 wallet
const depositToTx = await Proxy__L2StandardBridge.withdrawTo(
  l2TokenAddress, // l2 token address
  TargetAddress, // l1 target address
  exitAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
)
await depositToTx.wait()
```

### The Boba token list

The Standard bridge allows a one-to-many mapping between L1 and L2 tokens, meaning that there can be many Boba implementations of an L1 token. However there is always a one-to-one mapping between L1 and L2 tokens in the Boba token list.

| Network | URL                                                                                                                                                                            |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Mainnet | [Mainnet Boba Token List](https://github.com/bobanetwork/boba/blob/develop/packages/boba/register/addresses/addressesMainnet\_0x8376ac6C3f73a25Dd994E0b0669ca7ee0C02F089.json) |
| Goerli  | [Goerli Boba Token List](https://github.com/bobanetwork/boba/blob/develop/packages/boba/register/addresses/addressesGoerli\_0x6FF9c8FF8F0B6a0763a3030540c21aFC721A9148.json)   |

### Links

#### Mainnet

| Contract Name             | Contract Address                           |
| ------------------------- | ------------------------------------------ |
| Proxy\_\_L1StandardBridge | 0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00 |
| Proxy\_\_L2StandardBridge | 0x4200000000000000000000000000000000000010 |

#### Goerli

| Contract Name             | Contract Address                           |
| ------------------------- | ------------------------------------------ |
| Proxy\_\_L1StandardBridge | 0xDBD71249Fe60c9f9bF581b3594734E295EAfA9b2 |
| Proxy\_\_L2StandardBridge | 0x4200000000000000000000000000000000000010 |



<figure><img src="../../.gitbook/assets/using the fast token bridge.png" alt=""><figcaption></figcaption></figure>

The fast bridge provides a method for both side users to add liquidities for the L1 Fast Bridge Pool and the L2 Fast Bridge Pool. When an ERC20 token is deposited and added to L1 Fast Bridge Pool, the L2 Fast Bridge releases the token on L2 and charges a certain percentage of the deposit amount as the transaction fee. This process is known as "fast bridge a token". e.g. depositing 100 BOBA on L1 in exchange for 99.7 BOBA on L2 and also the reverse - withdrawing 100 BOBA on L2 in exchange for the 99.7 BOBA on L1. In addition to bridging tokens the standard bridge is also used for ETH.

The Standard Bridge is composed of two main contracts the [`L1LiquidityPool` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/boba/contracts/contracts/LP/L1LiquidityPool.sol)(for Layer 1) and the [`L2LiquidityPool` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/boba/contracts/contracts/LP/L2LiquidityPool.sol)(for Layer 2).

Here we'll go over the basics of using this bridge to move ERC20 and ETH assets between Layer 1 and Layer 2.

### Deposits

> Please check the liquidity balance of the L2 Liquidity Pool first before depositing tokens on the L1 Liquidity Pool. If the L2 Liquidity Pool doesn't have enough balance, your funds will be fast exited from L2 and the L1 Liquidity Pool charges a certain percentage of deposit amounts.

#### Deposit ERC20s or ETH

ERC20 deposits into L2 can be triggered via the `clientDepositL1` functions on the [`L1LiquidityPool` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/boba/contracts/contracts/LP/L1LiquidityPool.sol). You **must** approve the Standard Token Bridge to use the amount of tokens that you want to deposit or the deposit will fail.

```
const PRIVATE_KEY, L1_NODE_WEB3_URL, PROXY_L1_LIQUIDITY_POOL_ADDRESS

const L1Provider = new ethers.providers.StaticJsonRpcProvider(L1_NODE_WEB3_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L1Provider)

const Proxy__L1LiquidityPool = new ethers.Contract(
  PROXY_L1_LIQUIDITY_POOL_ADDRESS,
  L1LiquidityPoolABI,
  L1Wallet
)

// Approve amounts
const approveTx = await L1ERC20Contract.approve(Proxy__L1LiquidityPool.address, depositAmount)
await approveTx.wait()

// Deposit ERC20
const depositERC20Tx = await Proxy__L1LiquidityPool.clientDepositL1(
  depositAmount,
  l1TokenAddress,
)
await depositERC20Tx.wait()

// Deposit ETH
// We defined that ETH address is 0x0000000000000000000000000000000000000000 on L1
const depositETHTx = await Proxy__L1LiquidityPool.clientDepositL1(
  depositAmount,
  '0x0000000000000000000000000000000000000000', // ETH Address
  {value: depositAmount}
)
await depositETHTx.wait()
```

### Withdraws

> Please check the liquidity balance of the L1 Liquidity Pool first before depositing tokens on the L2 Liquidity Pool. If the L1 Liquidty Pool doesn't have enough balance, your funds will be fast deposited from L1 and the L2 Liquidity Pool charges a certain percentage of exit amounts.

#### Withdraw ERC20s or ETH

ERC20 and ETH withdrawals can be triggered via the `clientDepositL2` functions on the [`L2LiquidityPool` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/boba/contracts/contracts/LP/L2LiquidityPool.sol)

```
const PRIVATE_KEY, L2_NODE_WEB3_URL, PROXY_L2_LIQUIDITY_POOL_ADDRESS

const L2Provider = new ethers.providers.StaticJsonRpcProvider(L2_NODE_WEB3_URL)
const L2Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L2Provider)

const Proxy__L2LiquidityPool = new ethers.Contract(
  PROXY_L2_LIQUIDITY_POOL_ADDRESS,
  L2LiquidityPoolABI,
  L2Wallet
)

// Approve amounts
const approveTx = await L2ERC20Contract.approve(Proxy__L2LiquidityPool.address, depositAmount)
await approveTx.wait()

// Deposit ERC20
const depositERC20Tx = await Proxy__L2LiquidityPool.clientDepositL2(
  depositAmount,
  l2TokenAddress,
)
await depositERC20Tx.wait()

// Deposit ETH
// ETH address is 0x4200000000000000000000000000000000000006 on L2
const depositETHTx = await Proxy__L2LiquidityPool.clientDepositL2(
  depositAmount,
  '0x4200000000000000000000000000000000000006', // ETH Address
  {value: depositAmount}
)
await depositETHTx.wait()
```

### The Boba token list

The Fast bridge allows a one-to-one mapping between L1 and L2 tokens.

| Network | URL                                                                                                                                                                            |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Mainnet | [Mainnet Boba Token List](https://github.com/bobanetwork/boba/blob/develop/packages/boba/register/addresses/addressesMainnet\_0x8376ac6C3f73a25Dd994E0b0669ca7ee0C02F089.json) |
| Goerli  | [Goerli Boba Token List](https://github.com/bobanetwork/boba/blob/develop/packages/boba/register/addresses/addressesGoerli\_0x6FF9c8FF8F0B6a0763a3030540c21aFC721A9148.json)   |
