---
description: Learn more about the Boba Network Bridges
---

# Token Bridging

Although Boba Network is an L2 (and therefore fundamentally connected to Ethereum), it's also a separate blockchain system. App developers commonly need to move data and assets between Boba Network and Ethereum. We call the process of moving data and assets between the two networks "bridging".



<figure><img src="../../../.gitbook/assets/sending tokens between l1 and l2.png" alt=""><figcaption></figcaption></figure>

For the most common use case, moving tokens around, we use the Standard Token Bridge. The Standard Token Bridge is a simple smart contract with all the functionality you need to move tokens between Boba Network and Ethereum.

Beside the Standard Token Bridge, we created the Fast Token Bridge to allow you to exit assets from L2 in several hours or even several minutes based on the number of transactions. The Fast Token Bridge collects a percentage of the deposit amount as the transaction fee and distributes them to the liquidity providers.

<figure><img src="../../../.gitbook/assets/sending boba tokens between l1s.png" alt=""><figcaption></figcaption></figure>

The Boba instances are deployed on Ethereum, Avalanche, Moonbeam, BNB and Fantom. To bridge BOBA tokens between these L1s, you can use our cross chain bridge powder by [LayerZero Protocol](https://layerzero.network).
