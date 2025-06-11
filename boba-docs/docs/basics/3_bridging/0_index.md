# Introduction

Although Boba Network is an L2 (and therefore fundamentally connected to an L1), it's also a separate blockchain system. App developers commonly need to move data and assets between Boba Network and Ethereum or an alt-L1 like BNB Chain. We call the process of moving data and assets between the two networks "bridging".

## Sending Tokens Between L1 and L2

For the most common use case, moving tokens around, we use the Standard Token Bridge. The Standard Token Bridge is a simple smart contract with all the functionality you need to move tokens between Boba Network and Ethereum.

Beside the Standard Token Bridge, we created the [Light Bridge](./light-bridge) to allow you to rapidly bridge assets (including L2 exits to L1).

## Sending Boba Tokens Between L1s

The Boba instances are deployed on Ethereum and BNB. To bridge BOBA tokens between these L1s, you can use our [L1 to L1 bridge](./boba-token-bridge) powered by [LayerZero Protocol](https://layerzero.network).

## Bridge Contract Addresses

### Mainnet

| Contract Name             | Contract Address                           |
| ------------------------- | ------------------------------------------ |
| Proxy\_\_L1StandardBridge | 0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00 |
| Proxy\_\_L2StandardBridge | 0x4200000000000000000000000000000000000010 |

### Sepolia

| Contract Name             | Contract Address                           |
| ------------------------- | ------------------------------------------ |
| Proxy\_\_L1StandardBridge | 0x244d7b81EE3949788Da5F1178D911e83bA24E157 |
| Proxy\_\_L2StandardBridge | 0x4200000000000000000000000000000000000010 |
