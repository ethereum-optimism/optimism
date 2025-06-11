---
description: Learn more about the Standard Token Bridge
---

# Using the Standard Token Bridge

The standard bridge functionality provides a method for an ERC20 token to be deposited and locked on L1 in order to mint the same amount of an equivalent representation token on L2. This process is known as "bridging a token", e.g. depositing 100 BOBA on L1 in exchange for 100 BOBA on L2 and also the reverse - withdrawing 100 BOBA on L2 in exchange for the same amount on L1, in which case the representation token on L2 is burned in order to release the funds locked on L1. In addition to bridging tokens(ERC20) the standard bridge can also be used for ETH.

The Standard Bridge is composed of two main contracts the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts-bedrock/src/L1/L1StandardBridge.sol)(for Layer 1) and the [`L2StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts-bedrock/src/L2/L2StandardBridge.sol)(for Layer 2).

Here we'll go over the basics of using this bridge to move ERC20 and ETH assets between Layer 1 and Layer 2.

## Deposits

> Note: **We currently block smart contract wallets from calling the `depositETH` and `depositERC20` functions for security reasons**. If you want to deposit not using an EOA accounts and you know what are doing, you can use `depositETHTo` and `depositERC20To` functions instead.

### Deposit ERC20s

ERC20 deposits into L2 can be triggered via the `depositERC20` and `depositERC20To` functions on the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts-bedrock/src/L1/L1StandardBridge.sol). You **must** approve the Standard Token Bridge to use the amount of tokens that you want to deposit or the deposit will fail.

```js
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

### Deposit ETH

ETH deposits into L2 can be triggered via the `depositETH` and `depositETHTo` functions on the [`L1StandardBridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/contracts-bedrock/src/L1/L1StandardBridge.sol). ETH deposits can alternatively be triggered by sending ETH directly to the `L1StandardBridge`. Once your deposit is detected and finalized on Boba Network, your account will be funded with the corresponding amount of ETH on L2.

```js
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

## Withdrawals

### Withdraw tokens (ERC20s and ETH)

ERC20 withdrawals can be triggered via the `withdraw` or `withdrawTo` functions on the [`L2StandardBridge`](https://github.com/bobanetwork/boba/blob/develop/packages/contracts-bedrock/src/L2/L2StandardBridge.sol)

```js
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
const withdrawTx = await Proxy__L2StandardBridge.withdraw(
  '0x4200000000000000000000000000000000000006', // l2 token address
  ETHAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await withdrawTx.wait()

// Withdraw ETH to another l1 wallet
// ETH address is 0x4200000000000000000000000000000000000006 on L2
const withdrawToTx = await Proxy__L2StandardBridge.withdrawTo(
  '0x4200000000000000000000000000000000000006', // l2 token address
  TargetAddress, // l1 target address
  ETHAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
  {value: ETHAmount}
)
await withdrawToTx.wait()

// Approve amounts
const approveTx = await L2ERC20Contract.approve(Proxy__L2StandardBridge.address, exitAmount)
await approveTx.wait()

// Withdraw ERC20
const withdrawTx = await Proxy__L2StandardBridge.withdraw(
  l2TokenAddress // l2 token address
  exitAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
)
await withdrawTx.wait()

// Withdraw ERC20 to another l1 wallet
const withdrawToTx = await Proxy__L2StandardBridge.withdrawTo(
  l2TokenAddress, // l2 token address
  TargetAddress, // l1 target address
  exitAmount,
  9999999, // l1 gas limit
  ethers.utils.formatBytes32String(new Date().getTime().toString()), // byte data
)
await withdrawToTx.wait()
```

## Boba Token List

The Standard bridge allows a one-to-many mapping between L1 and L2 tokens, meaning that there can be many Boba implementations of an L1 token. However there is always a one-to-one mapping between L1 and L2 tokens in the Boba token list.

| Network | URL                                                          |
| ------- | ------------------------------------------------------------ |
| Mainnet | [Mainnet Boba Token List](../../token-addresses.md#Ethereum `<>` BOBA L2) |
| Sepolia | [Sepolia Boba Token List](../../token-addresses.md#Ethereum Sepolia `<>` BOBA L2) |
