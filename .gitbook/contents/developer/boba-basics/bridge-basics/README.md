---
description: Learn more about the Boba Network Bridges
---

# Token Bridging

Although Boba Network is an L2 (and therefore fundamentally connected to an L1), it's also a separate blockchain system. App developers commonly need to move data and assets between Boba Network and Ethereum or an alt-L1 like BNB Chain. We call the process of moving data and assets between the two networks "bridging".

<figure><img src="../../../../assets/sending tokens between l1 and l2.png" alt=""><figcaption></figcaption></figure>

For the most common use case, moving tokens around, we use the Standard Token Bridge. The Standard Token Bridge is a simple smart contract with all the functionality you need to move tokens between Boba Network and Ethereum.

Beside the Standard Token Bridge, we created the [Light Bridge](contents/developer/boba-basics/bridge-basics/light-bridge.md) to allow you to rapidly bridge assets (including L2 exits to L1).

<figure><img src="../../../../assets/sending boba tokens between l1s.png" alt=""><figcaption></figcaption></figure>

The Boba instances are deployed on Ethereum and BNB. To bridge BOBA tokens between these L1s, you can use our [L1 to L1 bridge](contents/developer/boba-basics/bridge-basics/boba-token-bridge.md) powered by [LayerZero Protocol](https://layerzero.network).
