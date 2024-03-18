---
description: Adding a new ERC20 token to Boba
---

# Adding a new ERC20 to Boba

<figure><img src="../../.gitbook/assets/deploy L2 ERC20.png" alt=""><figcaption></figcaption></figure>

### Deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via block explorer

The [L2StandardTokenFactory](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/contracts/contracts/L2/messaging/L2StandardTokenFactory.sol) is deployed and verified in the block explorer, so you can interact with the block explorer to deploy a new ERC20 token.

#### Mainnet Address

| Network                    | Contract Address                           | Block Explorer URL                                                                                                          |
| -------------------------- | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| Boba Mainnet (Ethereum L2) | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://bobascan.com/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826#writeContract                                       |
| Bobaavax (Avalanche L2)    | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.avax.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs      |
| Bobabeam (Moonbeam L2)     | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.bobabeam.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs  |
| Bobabnb (BNB L2)           | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.bnb.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs       |
| Bobaopera (Fantom L2)      | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.bobaopera.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs |

#### Testnet Address

| Network                               | Contract Address                           | Block Explorer URL                                                                                                                  |
| ------------------------------------- | ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| Boba Goerli (Ethereum Goerli L2)      | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://testnet.bobascan.com/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826#writeContract                                       |
| Bobaavax Testnet (Avalanche Fuji L2)  | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.testnet.avax.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs      |
| Bobabase (Moonbase L2)                | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.bobabase.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs          |
| Bobabnb Testnet (BNB Testnet L2)      | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.testnet.bnb.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs       |
| Bobaopera Testnet (Fantom Testnet L2) | 0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826 | https://blockexplorer.testnet.bobaopera.boba.network/address/0xD2ae16D8c66ac7bc1Cf3c9e5d6bfE5f76BeDb826/write-contract#address-tabs |

### Deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via the script

You can deploy [L2StandardERC20.sol](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/contracts/contracts/standards/L2StandardERC20.sol) via the following script

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



<figure><img src="../../.gitbook/assets/Bridge the new token L2.png" alt=""><figcaption></figcaption></figure>

ERC20 deposits into L2 can be triggered via the `depositERC20` and `depositERC20To` functions on the [`L1StandardBridge`](https://github.com/bobanetwork/boba/blob/release/v0.2.2/packages/contracts/contracts/L1/messaging/L1StandardBridge.sol). You **must** approve the Standard Token Bridge to use the amount of tokens that you want to deposit or the deposit will fail.

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

| L1        | Contract Name             | Contract Address                           |
| --------- | ------------------------- | ------------------------------------------ |
| Ethereum  | Proxy\_\_L1StandardBridge | 0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00 |
| Avalanche | Proxy\_\_L1StandardBridge | 0xf188F1e92B2c78956D2859b84684BFD17103e22c |
| Moonbeam  | Proxy\_\_L1StandardBridge | 0xAf5297f68D48cd2DE37Ee5cbaC0647fbA4132985 |
| BNB       | Proxy\_\_L1StandardBridge | 0x1E0f7f4b2656b14C161f1caDF3076C02908F9ACC |
| Fantom    | Proxy\_\_L1StandardBridge | 0xb7629EF94B991865940E8A840Aa7d68fa88c3Fe8 |

### Testnet

| L1              | Contract Name             | Contract Address                           |
| --------------- | ------------------------- | ------------------------------------------ |
| Ethereum Goerli | Proxy\_\_L1StandardBridge | 0xDBD71249Fe60c9f9bF581b3594734E295EAfA9b2 |
| Avalanche Fuji  | Proxy\_\_L1StandardBridge | 0x07B606934b5B5D6A9E1f8b78A0B26215FF58Ad56 |
| Moonbase        | Proxy\_\_L1StandardBridge | 0xEcca5FEd8154420403549f5d8F123fcE69fae806 |
| BNB Testnet     | Proxy\_\_L1StandardBridge | 0xBf0939120b4F5E3196b9E12cAC291e03dD058e9a |
| Fantom Testnet  | Proxy\_\_L1StandardBridge | 0x86FC7AeFcd69983A8d82eAB1E0EaFD38bB42fd3f |
