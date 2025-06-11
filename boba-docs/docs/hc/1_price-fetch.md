# Price Fetch Example

To dip our toes into Hybird Compute's capabilities, consider the following simple smart contract which fetches the price of different cryptocurrencies. You can view the full example in our [repository](https://github.com/bobanetwork/aa-hc-example/blob/main/contracts/contracts/TokenPrice.sol).

We can start by importing some libraries (one to allow the use of Hybrid Compute, one to give exclusive access to certain functions as an "owner") and providing the general structure of our contract:

```solidity
// SPDX-License-Identifier: GPL-3.0
pragma solidity 0.8.19;

import "./samples/HybridAccount.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract TokenPrice is Ownable {
    mapping(string => TokenPriceStruct) public tokenPrices;
    HybridAccount public HA;

    event FetchPriceError(uint32 err);
    event FetchPriceRet(bytes err);

    struct TokenPriceStruct {
        string price;
        uint256 timestamp; // Timestamp of when the price was fetched.
    }

    constructor(
        address _hybridAccount
    ) Ownable() {
        HA = HybridAccount(payable(_hybridAccount));
    }
}
```

To this, we can add our price fetching logic:

```solidity
    function fetchPrice(string calldata token) public {
        string memory price;

        // Encode function signature, which is called on the offchain rpc server.
        bytes memory req = abi.encodeWithSignature("getprice(string)", token);
        bytes32 userKey = bytes32(abi.encode(msg.sender));
        (uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

        if (error != 0) {
            emit FetchPriceError(error);
       
            revert(string(ret));
        }

        // Decode price, which was encoded as a string on the offchain rpc server.
        (price) = abi.decode(ret, (string));
        tokenPrices[token] = TokenPriceStruct({
            price: price,
            timestamp: block.timestamp
        });
    }
```

Note the call `HA.CallOffchain()`; this leverages Hybrid Compute's ability to access data and compute off-chain. This reduces the time and complexity it takes to perform this price fetch, evident in how quickly this fetching happens. You can test this contract with [this app](https://aa-hc-example-fe.onrender.com/), which includes a UI to clearly show the time (or lack thereof) Hybrid Compute takes to fetch the prices.
