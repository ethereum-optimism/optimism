---
description: Learn more about the BOBA Token Bridge Between L1s
---

# Using the BOBA Token Bridge Between L1s

The BOBA token bridge functionality provides a method for the BOBA token to be deposited and locked on Ethereum in order to mint the same amount of an equivalent representation token on Alt L1s. This process is known as "bridging a token", e.g. depositing 100 BOBA to the bridge contract on Ethereum in exchange for 100 BOBA on an Alt L1 and also the reverse - withdrawing 100 BOBA on an Alt L1 in exchange for the same amount on Ethereum, in which case the BOBA token on the Alt L1 is burned in order to release the funds locked on Ethereum.

The BOBA Token Bridge is composed of two main contracts the [`EthBridge` (opens new window)](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/contracts/contracts/lzTokenBridge/EthBridge.sol)(for Ethereum) and the [`AltL1Bridge` (opens new window)](https://github.com/bobanetwork/boba/blob/develop/packages/boba\_legacy/contracts/contracts/lzTokenBridge/AltL1Bridge.sol)(for Alt L1s).

Here we'll go over the basics of using this bridge to move BOBA tokens between Layer 1s.

## Bridge Boba Tokens from Ethereum

For normal users, bridging can be accomplished simply via the [Light Bridge](light-bridge).

For developers, you can interact with [`EthBridge` (opens new window)](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/contracts/contracts/lzTokenBridge/EthBridge.sol) to deposit BOBA tokens from Ethereum to Alt L1s.

```javascript
const PRIVATE_KEY, L1_NODE_WEB3_URL, DEPOSIT_AMOUNT, BOBA_TOEKN_ADDRESS_ON_ETHEREUM, BOBA_TOKEN_ADDRESS_ON_ALT_L1, ALT_L1_LARYER_ZERO_CHAIN_ID, ETHEREUM_LARYER_ZERO_CONTRACT, PROXY__ETHEREUM_BRIDGE

const L1Provider = new ethers.providers.StaticJsonRpcProvider(L1_NODE_WEB3_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L2Provider)

const Proxy__EthBridge = new ethers.Contract(
  PROXY__ETHEREUM_BRIDGE,
  EthBridgeJson.abi,
  L1Wallet,
);

const EthBOBA = new ethers.Contract(
  BOBA_TOEKN_ADDRESS_ON_ETHEREUM,
  BobaTokenJson.abi,
  L1Wallet,
);

const ETHLayzerZeroEndpoint = new ethers.Contract(
  ETHEREUM_LARYER_ZERO_CONTRACT,
  LZEndpointMockJson.abi,
  L1Wallet,
);

// approve boba
const approveTx = await EthBOBA.approve(Proxy__EthBridge.address, ethers.utils.parseEther(DEPOSIT_AMOUNT));
await approveTx.wait();
console.log(`-> Approved ${DEPOSIT_AMOUNT} BOBA tokens for transfer`);

// estimate fee
let payload = ethers.utils.defaultAbiCoder.encode(
  ["address", "address", "address", "address", "uint256", "bytes"],
  [
    BOBA_TOEKN_ADDRESS_ON_ETHEREUM,
    BOBA_TOKEN_ADDRESS_ON_ALT_L1,
    L1Wallet.address,
    Target_Wallet_Address_On_Alt_L1,
    ethers.utils.parseEther(DEPOSIT_AMOUNT.toString()),
    '0x',
  ]
);

let estimatedFee = await ETHLayzerZeroEndpoint.estimateFees(
  ALT_L1_LARYER_ZERO_CHAIN_ID,
  PROXY__ETHEREUM_BRIDGE.address,
  payload,
  false,
  '0x',
);
console.log(`!!! Estimated fee: ${ethers.utils.formatEther(estimatedFee._nativeFee)}!!!`);

await Proxy__EthBridge.depositERC20To(
  EthBOBA.address,
  BOBA_TOKEN_ADDRESS_ON_ALT_L1,
  ethers.utils.parseEther(DEPOSIT_AMOUNT.toString()),
  ethers.constants.AddressZero,
  '0x', // adapterParams
  '0x',
  { value: estimatedFee._nativeFee }
);
console.log(`-> Sent ${DEPOSIT_AMOUNT} BOBA tokens to the bridge contract....`);
```

Example code can be found here: [bridgeFromEthereumToAltL.js](https://github.com/bobanetwork/boba-cross-chain-bridges/blob/main/scripts/bridgeFromEthereumToAltL1.js).

## Bridge Boba Tokens from Alt L1s

For normal users, bridging can be accomplished simply via the [Light Bridge](light-bridge).

For developers, you can interact with [`AltL1Bridge` (opens new window)](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/contracts/contracts/lzTokenBridge/AltL1Bridge.sol) to deposit BOBA tokens from Alt L1 to Ethereum.

```javascript
const PRIVATE_KEY, L1_NODE_WEB3_URL, DEPOSIT_AMOUNT, BOBA_TOEKN_ADDRESS_ON_ETHEREUM, BOBA_TOKEN_ADDRESS_ON_ALT_L1, ETH_LARYER_ZERO_CHAIN_ID, Alt_L1_LARYER_ZERO_CONTRACT, PROXY__ALT_L1_BRIDGE

const L1Provider = new ethers.providers.StaticJsonRpcProvider(L1_NODE_WEB3_URL)
const L1Wallet = new ethers.Wallet(PRIVATE_KEY).connect(L2Provider)

const Proxy__AltL1Bridge = new ethers.Contract(
  PROXY__ALT_L1_BRIDGE,
  AltBridgeJson.abi,
  L1Wallet,
);

const AltL1BOBA = new ethers.Contract(
  BOBA_TOEKN_ADDRESS_ON_Alt_L1,
  BobaTokenJson.abi,
  L1Wallet,
);

const AltL1LayzerZeroEndpoint = new ethers.Contract(
  Alt_L1_LARYER_ZERO_CONTRACT,
  LZEndpointMockJson.abi,
  L1Wallet,
);

// approve boba
const approveTx = await AltL1BOBA.approve(Proxy__AltL1Bridge.address, ethers.utils.parseEther(DEPOSIT_AMOUNT));
await approveTx.wait();
console.log(`-> Approved ${DEPOSIT_AMOUNT} BOBA tokens for transfer`);

// estimate fee
let payload = ethers.utils.defaultAbiCoder.encode(
  ["address", "address", "address", "address", "uint256", "bytes"],
  [
    BOBA_TOKEN_ADDRESS_ON_ETHREUM,
    BOBA_TOKEN_ADDRESS_ON_ALT_L1,
    L1Wallet.address,
    Target_Wallet_Address_On_ETHEREUM,
    ethers.utils.parseEther(DEPOSIT_AMOUNT.toString()),
    '0x',
  ]
);

let estimatedFee = await AltL1LayzerZeroEndpoint.estimateFees(
  ETH_LARYER_ZERO_CHAIN_ID,
  Proxy__AltL1Bridge.address,
  payload,
  false,
  '0x',
);
console.log(`!!! Estimated fee: ${ethers.utils.formatEther(estimatedFee._nativeFee)}!!!`);

await Proxy__AltL1Bridge.depositERC20To(
  AltL1BOBA.address,
  BOBA_TOKEN_ADDRESS_ON_ETHEREUM,
  ethers.utils.parseEther(DEPOSIT_AMOUNT.toString()),
  ethers.constants.AddressZero,
  '0x', // adapterParams
  '0x',
  { value: estimatedFee._nativeFee }
);
console.log(`-> Sent ${DEPOSIT_AMOUNT} BOBA tokens to the bridge contract....`);
```

## Links

### Mainnet

#### Layer Zero Contracts

| Network  | Layer Zero Contract Address                | Chain ID |
| -------- | ------------------------------------------ | -------- |
| Ethereum | 0x66A71Dcef29A0fFBDBE3c6a460a3B5BC225Cd675 | 101      |
| BNB      | 0x3c2269811836af69497E5F486A85D7316753cf62 | 102      |

#### Ethereum Contracts

| Contract Name                 | Contract Address                           |
| ----------------------------- | ------------------------------------------ |
| Proxy\_\_EthBridgeToBNB       | 0x1A36E24D61BC1aDa68C21C2Da1aD53EaB8E03e55 |
| L1\_BOBA                      | 0x42bBFa2e77757C645eeaAd1655E0911a7553Efbc |

#### BNB Contract

| Contract Name           | Contract Address                           |
| ----------------------- | ------------------------------------------ |
| Proxy\_\_BNBBridgeToEth | 0x819FF4d9215C9dAC76f5eC676b1355973157eBBa |
| L1\_BOBA                | 0xE0DB679377A0F5Ae2BaE485DE475c9e1d8A4607D |
