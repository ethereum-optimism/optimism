---
description: Adding a new ERC20 token to Boba
---

# Adding a new ERC20 to Boba

<figure><img src="../../../assets/deploy L2 ERC20.png" alt=""><figcaption></figcaption></figure>

### Deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via block explorer

The [L2StandardTokenFactory](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/contracts/contracts/L2/messaging/L2StandardTokenFactory.sol) is deployed and verified in the block explorer, so you can interact with the block explorer to deploy a new ERC20 token.

#### Mainnet Address

| Network                    | Contract Address                           | Block Explorer URL                                           |
| -------------------------- | ------------------------------------------ | ------------------------------------------------------------ |
| Boba Mainnet (Ethereum L2) | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://bobascan.com/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 |
| Bobabnb (BNB L2)           | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://bobascan.com/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 |

#### Testnet Address

| Network                            | Contract Address                           | Block Explorer URL                                           |
| ---------------------------------- | ------------------------------------------ | ------------------------------------------------------------ |
| Boba Sepolia (Ethereum Sepolia L2) | 0x4200000000000000000000000000000000000012 | https://testnet.bobascan.com/address/0x4200000000000000000000000000000000000012 |
| Bobabnb Testnet (BNB Testnet L2)   | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://testnet.bobascan.com/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 |

### Deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via the script

You can deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via the following script

```js
const Factory__L2StandardERC20 = new ethers.ContractFactory(
  L2StandardERC20Json.abi,
  L2StandardERC20Json.bytecode,
  L2Wallet
)
const L2StandardERC20 = await Factory__L2StandardERC20.deploy(
  '0x4200000000000000000000000000000000000010',
  L1_TOKEN_ADDRESS,
  TOKEN_NAME,
  TOKEN_SYMBOL,
  TOKEN_DECIMAL
)
```



<figure><img src="../../../assets/Bridge the new token L2.png" alt=""><figcaption></figcaption></figure>

ERC20 deposits into L2 can be triggered via the `depositERC20` and `depositERC20To` functions on the [`L1StandardBridge`](https://github.com/bobanetwork/boba\_legacy/blob/release/v0.2.2/packages/contracts/contracts/L1/messaging/L1StandardBridge.sol). You **must** approve the Standard Token Bridge to use the amount of tokens that you want to deposit or the deposit will fail.

```js
const L1StandardERC20 = new ethers.Contract(
  PROXY__L1STANDARDBRIDGE_ADDRESS,
  L1StandardBridgeJson.abi,
  L1Wallet,
)

const depositTxStatus = await L1StandardERC20.depositERC20(
  L1_TOKEN_ADDRESS,
  '0x4200000000000000000000000000000000000006',
  L1_TOKEN_AMOUNT,
  9999999,
  ethers.utils.formatBytes32String(new Date().getTime().toString())
)
```

### Mainnet

| L1       | Contract Name             | Contract Address                           |
| -------- | ------------------------- | ------------------------------------------ |
| Ethereum | Proxy\_\_L1StandardBridge | 0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00 |
| BNB      | Proxy\_\_L1StandardBridge | 0x1E0f7f4b2656b14C161f1caDF3076C02908F9ACC |

### Testnet

| L1               | Contract Name             | Contract Address                           |
| ---------------- | ------------------------- | ------------------------------------------ |
| Ethereum Sepolia | Proxy\_\_L1StandardBridge | 0x244d7b81EE3949788Da5F1178D911e83bA24E157 |
| BNB Testnet      | Proxy\_\_L1StandardBridge | 0xBf0939120b4F5E3196b9E12cAC291e03dD058e9a |
