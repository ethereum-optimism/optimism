---
description: A comprehensive guide to use DIA
---

# Introduction to DIA Oracles

#### Introduction to DIA&#x20;

[DIA](https://diadata.org/) is a cross-chain oracle provider that sources granular market data from diverse exchanges, including CEXs, DEXs, and NFT marketplaces. Its data sourcing is thorough, enabling unparalleled transparency and customizability for resilient price feeds for 20,000+ assets. Its versatile data processing and delivery ensures adaptability and reliability for any decentralized application.

#### Oracle configuration settings

Oracle specifications:

**Methodology: VWAPIR**

The final price point for each asset is calculated by computing the assets' trade information across multiple DEXs and CEXs. This is done using a Volume Weighted Average Price with Interquartile Range (VWAPIR) methodology. [Learn more about VWAPIR](https://docs.diadata.org/products/token-price-feeds/exchangeprices/vwapir-volume-weighted-average-price-with-interquartile-range-filter).

**Update frequency: 15 seconds (if 0.2% deviation) + 60 minute heartbeat**

The oracles scan for price changes every 15 seconds. If it detects a price fluctuation exceeding 0.2% from the last published rate, it promptly sends an update on-chain. Moreover, a consistent heartbeat refreshes all asset prices every hour.

#### Included price feeds + data sources

The Boba-ETH oracle includes the following price feeds:

* BTC/USD | [See data sources](https://www.diadata.org/app/price/asset/Bitcoin/0x0000000000000000000000000000000000000000/)
* ETH/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0x0000000000000000000000000000000000000000/)
* USDC/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/)
* USDT/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0xdAC17F958D2ee523a2206206994597C13D831ec7/)
* DAI/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0x6B175474E89094C44Da98b954EedeAC495271d0F/)
* BOBA/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0x42bBFa2e77757C645eeaAd1655E0911a7553Efbc/)

The Boba-BNB oracle includes the following price feeds:

* BTC/USD | [See data sources](https://www.diadata.org/app/price/asset/Bitcoin/0x0000000000000000000000000000000000000000/)
* ETH/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0x0000000000000000000000000000000000000000/)
* USDC/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/)
* USDT/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0xdAC17F958D2ee523a2206206994597C13D831ec7/)
* BNB/USD | [See data sources](https://www.diadata.org/app/price/asset/BinanceSmartChain/0x0000000000000000000000000000000000000000/)
* BOBA/USD | [See data sources](https://www.diadata.org/app/price/asset/Ethereum/0x42bBFa2e77757C645eeaAd1655E0911a7553Efbc/)

Learn more about DIA’s [data sourcing](https://docs.diadata.org/introduction/dia-technical-structure/data-sourcing) and [data computation](https://docs.diadata.org/introduction/dia-technical-structure/data-computation) architecture.

#### Deployed contracts

Access the oracles in the smart contracts below:

* Boba-ETH: https://bobascan.com/address/0xB3519a1760F3A16934a890F19E67A98cBF1e0859
* Boba-BNB: https://bobascan.com/address/0xCF06Ac8C6FFb7b7E313D43e8Fb2E740D5aE82e6E
* Boba-Sepolia: https://testnet.bobascan.com/address/0x907e7f6bd9653E4188da89E4f2D3EA949dcEc076

#### How to access DIA oracles?

Here is an example of how to access a price value on DIA oracles:

1. Access your custom oracle smart contract on Boba.
2. Call getValue(pair\_name) with pair\_name being the full pair name such as BTC/USD. You can use the "Read" section on the explorer to execute this call.
3. The response of the call contains two values:
4.
   1. The current asset price in USD with a fix-comma notation of 8 decimals.
   2. The UNIX timestamp of the last oracle update.

You can find DIA's oracle integration samples in Solidity and Vyper languages by visiting:&#x20;

→ [Access the Oracle | DIA Documentation](https://docs.diadata.org/products/token-price-feeds/access-the-oracle)&#x20;

### Support

For assistance, connect with the DIA team directly on [Discord](https://discord.gg/ZvGjVY5uvs) or [Telegram](https://t.me/diadata\_org). Developers seeking other specialized, production-grade oracle with tailored price feeds and configurations can initiate the request here: [Request a Custom Oracle | DIA Documentation](https://docs.diadata.org/introduction/intro-to-dia-oracles/request-an-oracle)
