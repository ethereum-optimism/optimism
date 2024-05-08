# Price Oracles

Price Feed oracles allow smart contracts to work with external data and open the path to many more use cases. Boba has several options to get real world price data directly into your contracts - each different in the way they operate to procure data for smart contracts to consume:

1. Witnet
2. Hybrid Compute

<figure><img src="../../../assets/witnet price feeds.png" alt=""><figcaption></figcaption></figure>

The Witnet multichain decentralized oracle enables smart contracts to realize their true potential by giving them access to all sorts of valuable data sets, and by attesting and delivering that information securely thanks to its strong cryptoeconomic guarantees.

Witnet can power most DeFi primitives like price feeds, stablecoins, synthetics, etc., as well as acting as a reliable source of randomness for creating uniqueness in NFTs.

### Feeds supported

A complete list of publicly available Witnet data feeds on Boba can be found in the Witnet Data Feeds website: [https://feeds.witnet.io/boba](https://feeds.witnet.io/boba)

[Request a new price feed on Boba](https://tally.so/r/wMZDAn) or [Create your own data feed](https://docs.witnet.io/smart-contracts/witnet-web-oracle/make-a-get-request).

### How To Use Witnet Price Feeds

Witnet price feeds can be integrated into your own Boba Mainnet contracts in two different ways:

1. [Integrate through proxy](https://docs.witnet.io/smart-contracts/witnet-data-feeds/using-witnet-data-feeds#reading-multiple-currency-pairs-from-the-router) Recommended for testing and upgradability. This is the preferred way to consume the Witnet-powered price feeds. Through using the _**Price Feeds Router**_.
2. [Integrate directly](https://docs.witnet.io/smart-contracts/witnet-data-feeds/using-witnet-data-feeds#reading-last-price-and-timestamp-from-a-price-feed-contract-serving-a-specific-pair) Optimized for gas cost and decentralization

The _**WitnetPriceRouter**_ smart contract is deployed in all the [supported chains](https://docs.witnet.io/smart-contracts/witnet-data-feeds/addresses) and allows your own smart contracts and Web3 applications to get the latest price of any of the [supported currency pairs](https://docs.witnet.io/smart-contracts/witnet-data-feeds/price-feeds-registry#currency-pairs) by providing the identifier of the pair to a single Solidity method. This removes the need to know the actual contract addresses handling the price updates from the Witnet oracle.

### Reading multiple price pairs from the router

**WitnetPriceRouter**documentation

_Mainnet_: [0x93f61D0D5F623144e7C390415B70102A9Cc90bA5](https://bobascan.com/address/0x93f61D0D5F623144e7C390415B70102A9Cc90bA5/)

_Goerli_: [0x36928Aeedaaf7D85bcA39aDfB2A39ec529ce221a](https://testnet.bobascan.com/address/0x36928Aeedaaf7D85bcA39aDfB2A39ec529ce221a/)

The Price Router contract is the easiest and most convenient way to consume Witnet price feeds on any of the [supported chains](../../smart-contracts/supported-chains/).

#### Solidity example

The example below shows how to read the price of two different assets from the Witnet Price Router:

```javascript
 // SPDX-License-Identifier: MIT
pragma solidity ^0.8.11;
import "witnet-solidity-bridge/contracts/interfaces/IWitnetPriceRouter.sol";
contract MyContract {
    IWitnetPriceRouter immutable public router;
    /**
     * IMPORTANT: pass the WitnetPriceRouter address depending on
     * the network you are using! Please find available addresses here:
     * https://docs.witnet.io/smart-contracts/price-feeds/contract-addresses
     */
    constructor(IWitnetPriceRouter _router))
        router = _router;
    }
    /// Returns the BTC / USD price (6 decimals), ultimately provided by the Witnet oracle.
    function getBtcUsdPrice() public view returns (int256 _price) {
        (_price,,) = router.valueFor(bytes4(0x24beead4));
    }
    /// Returns the ETH / USD price (6 decimals), ultimately provided by the Witnet oracle.
    function getEthUsdPrice() public view returns (int256 _price) {
        (_price,,) = router.valueFor(bytes4(0x3d15f701));
    }
    /// Returns the BTC / ETH price (6 decimals), derived from the ETH/USD and
    /// the BTC/USD pairs that were ultimately provided by the Witnet oracle.
    function getBtcEthPrice() public view returns (int256 _price) {
        return (1000000 * getBtcUsdPrice()) / getEthUsdPrice();
    }
}
```

#### Javascript example

You may also read the latest price of any of the supported currency pairs from your **Web3** application by interacting directly with the Price Router contract:

```javascript
web3 = Web3(Web3.HTTPProvider('https://mainnet.boba.network'))
abi = '[{ "inputs": [{ "internalType": "bytes32", "name": "_id", "type": "bytes32" }], "name": "valueFor", "outputs": [{ "internalType": "int256", "name": "", "type": "int256" }, { "internalType": "uint256", "name": "", "type": "uint256" }, { "internalType": "uint256", "name": "", "type": "uint256" }], "stateMutability": "view", "type": "function" }]'
addr = '0x36928Aeedaaf7D85bcA39aDfB2A39ec529ce221a'
contract = web3.eth.contract(address=addr, abi=abi)
// get last value for "Price-BOBA/USDT-6"
valueFor = contract.functions.valueFor().call("0xf723bde1")
print("Price-BOBA/USDT-6:", valueFor[0])
print("> lastTimestamp:", valueFor[1])
print("> latestUpdateStatus:", valueFor[2])
```

For more information about Witnet please refer to:

[website](https://witnet.io/) | [docs](https://docs.witnet.io/) | [github](https://github.com/witnet) | [twitter](https://twitter.com/witnet\_io) | [telegram](https://t.me/witnetio) | [discord](https://discord.gg/witnet)

<figure><img src="../../../assets/hc-under-upgrade.png" alt=""><figcaption></figcaption></figure>

<figure><img src="../../../assets/hybridcompute.png" alt=""><figcaption></figcaption></figure>

Hybrid Compute is Boba's off-chain compute system and among many other things you can fetch real-world market price data. Hybrid Compute gives you the flexibility to select and set up your own data source. Or even select and work with any other reliable service that can help provide such data. In the background, Hybrid Compute works with a modified L2Geth, by intercepting and injecting real world responses into the transaction. Learn more about Hybrid Compute [here](hybrid\_compute.md).

Note: Unlike a feed contract where every data query remains on-chain, Hybrid Compute requests are a call to an external endpoint to retrieve data - which are subject to unavailability or distortion. **Best practices include using decentralized on-chain oracles and/or off-chain 'augmentation' where off-chain compute is used to estimate the reliability of on-chain oracles**.

### Feeds supported

_Goerli/Mainnet_: potentially everything, dependent on your source

_Fee_: 0.01 BOBA per Hybrind Compute request