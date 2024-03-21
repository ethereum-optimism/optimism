# Price Oracles

Price Feed oracles allow smart contracts to work with external data and open the path to many more use cases. Boba has several options to get real world price data directly into your contracts - each different in the way they operate to procure data for smart contracts to consume:

1. Boba-Straw
2. Witnet
3. Bobalink
4. Hybrid Compute

<figure><img src="../../../assets/boba straw.png" alt=""><figcaption></figcaption></figure>

Boba-Straw, Boba's self-operated price feed oracle is based on ChainLink's implementation and can handle price data aggregation from multiple trusted external entities (data oracles), on-chain. Currently, Boba-Straw is powered by Folkvang, our first data oracle. The price data is submitted based on the 0.25% price change threshold, but the maximal frequency is once every 10 minutes per market. To further increase reliability and precision, we are adding more data-sources. Data oracles accumulate BOBA for every submission to offset operational and gas costs. To be a data-provider oracle and earn BOBA refer to the section below.

### Feeds supported

_Mainnet_: \[ETH/USD, BOBA/USD, WBTC/USD, OMG/USD]

_Goerli_: \[ETH/USD, BOBA/USD, WBTC/USD, OMG/USD]

_Fee_: free

[\[_Quick-Link - Mainnet_\]](https://bobascan.com/address/0x01a109AB8603ad1B6Ef5f3B2B00d4847e6E554b1)

[\[_Quick-Link - Goerli_\]](https://testnet.bobascan.com/address/0xE84AAb853C4FBaafd3eD795F67494d4Da1539492)

### I want to be a data source

To be a data oracle and help Boba-Straw by submitting price data:

* You must have reliable, independent price data. While price data from all oracles is aggregated and determined on-chain, more layers of data-aggregation helps build reliability
* You must react to the rounds for aggregation, to seamlessly work with other independent data providers

To find the when and hows of submitting data, let's take a quick look at the round structure first.

### Rounds and on-chain aggregation

Token price data aggregation happens in rounds, triggered by oracles when there is a need for price update. The 'price' answer of the latest _finalized round_ is the latest price. Here, `finalized round` refers to a round with >= min submissions.

For a round of aggregation, independent oracles submit their 'price' answers. When a round receives enough submissions (>= min submissions), the price update is accepted and computed to the median of all submissions for the specific round. Then for a further price update we move to the next round.

The 'price' answer for the round isn't finalized/accepted until the round has received a certain 'min no of answer submissions' from separate oracles. While the round moves between having min < submissions < max, the computed answer can vary depending on the data received up till that point. And after the 'max no of submissions on the round' the 'price' answer is finalized and fixed. If a round does not receive 'min no of answer submissions', the round can be superseded after a timeout period (currently 3mins).

### Submitting price data

To be eligible to submit price data, the oracle (and the oracle admin) addresses needs to be added by the admin

* oracle - the address that will submit the price
* oracle admin - the address that will control withdrawals of accumulated $BOBA

The rules of the game to manage the round co-ordination between all the independent data oracles are:

* Trigger a round if you notice a price update that needs to be recorded
* Keep checking and provide your answer for the round if someone else has triggered a round
* Make sure you do not try to trigger a new round when the last round is unfinished and the timeout period hasn't elapsed. There is a 'restartDelay' - the minimum number of rounds you have to wait before you can trigger a new round.

Use the **`function oracleRoundState(address _oracle, uint32 _queriedRoundId)`** to determine eligibility for specific roundId. The same method **`oracleRoundstate(_oracle, 0)`** can also be used to suggest the next eligible round for the oracle.

The main contracts to interact with are the respective FluxAggregators for each feed.

_To submit data to the feed_ the oracles need to call **`submit(roundId, value)`**.

Here, `value` is the price to submit, note: in decimals as returned by the contract (currently set to 8) and `roundId` refers to round, which is consecutive and starts from 1 for the specific feed. The oracle can only submit once for a specific round.

For more info please refer to:

[\[contracts\]](../../oracles/oracle.md)

[\[examples\]](../../boba\_examples/boba-straw/)

### I want my contracts to receive data

To fetch price feed data directly into your contracts, make your contract call the "Feed Registry" to extract the current and historical prices for all the feeds on Boba-Straw:

_Feed Registry (Mainnet)_: 0x01a109AB8603ad1B6Ef5f3B2B00d4847e6E554b1

_Feed Registry (Goerli)_: 0xE84AAb853C4FBaafd3eD795F67494d4Da1539492

Feeds are registered to the registry in the form of base/quote pairs, these terms used here and throughout - 'base' refers to the crypto asset/token and 'quote' refers to the asset (or fiat currency) to use as a reference for the price.

A quick note on fees and subscription: Currently the feed is free to use for the contracts. Once we transition to the BOBA subscription model, you would have to pay BOBA and pre-subscribe your contracts (time based) to extract data from the feed.

### Extracting the price

To get the latest price data call method **`latestRoundData(base, quote)`**. To get the price data from a certain past round (historical price) call method **`getRoundData(base, quote, roundId)`**. The `roundId` supplied here is phaseId plus aggregator roundId, for reference query the latest `roundId`. The answer returned will be of the form of decimals specified on the contract call method **`decimals(base, quote)`**. For example,

```javascript
import "@boba/contracts/oracle/FeedRegistry.sol"

contract MyContract {

address feedRegistryAddress = '0x01a109AB8603ad1B6Ef5f3B2B00d4847e6E554b1';

    function readFromPriceFeed() external view returns(int256) {
        FeedRegistry feedRegistry = FeedRegistry(feedRegistryAddress);

        address bobaTokenAddress = '0xa18bF3994C0Cc6E3b63ac420308E5383f53120D7';
        address USD = address(840);

        (,int256 value,,uint256 time,) = feedRegistry.latestRoundData(bobaTokenAddress, USD);
        // do something with time

        return value;
    }

}
```

`base` is always the token address and `quote` is fiat in the ISO\_4217 form.

### Alternate data queries

While the above is the recommended way to ask for the price data, and check time along with it, there is also the option to only query the price:

For the latest price call **`latestAnswer(base, quote)`**.

For the price from a certain past round call **`getAnswer(base, quote, roundId)`**. `roundId` supplied here is `phaseId` plus aggregator `roundId`.

For the latest completed round call **`latestRound(base, quote)`**.

To get the latest timestamp call **`latestTimestamp(base, quote)`**.

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

_Mainnet_: [0x93f61D0D5F623144e7C390415B70102A9Cc90bA5](https://bobascan.com/address/0x93f61D0D5F623144e7C390415B70102A9Cc90bA5/read-contract)

_Goerli_: [0x36928Aeedaaf7D85bcA39aDfB2A39ec529ce221a](https://testnet.bobascan.com/address/0x36928Aeedaaf7D85bcA39aDfB2A39ec529ce221a/read-contract)

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

<figure><img src="../../../assets/hybridcompute.png" alt=""><figcaption></figcaption></figure>

Hybrid Compute is Boba's off-chain compute system and among many other things you can fetch real-world market price data. Hybrid Compute gives you the flexibility to select and set up your own data source. Or even select and work with any other reliable service that can help provide such data. In the background, Hybrid Compute works with a modified L2Geth, by intercepting and injecting real world responses into the transaction. Learn more about Hybrid Compute [here](hybrid\_compute.md).

Note: Unlike a feed contract where every data query remains on-chain, Hybrid Compute requests are a call to an external endpoint to retrieve data - which are subject to unavailability or distortion. **Best practices include using decentralized on-chain oracles and/or off-chain 'augmentation' where off-chain compute is used to estimate the reliability of on-chain oracles**.

### Feeds supported

_Goerli/Mainnet_: potentially everything, dependent on your source

_Fee_: 0.01 BOBA per Hybrind Compute request
