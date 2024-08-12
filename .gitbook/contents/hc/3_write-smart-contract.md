# Write the Smart Contract

Now we can write the smart contract, which will call our previously created off-chain handler. You can find the needed `HybridAccount` Contract along with its dependencies in the provided repository.

In the first part of the contract, we create a mapping for the counters and define a `demoAddr`. This address will then be part of the `HybridAccount`.

```solidity
// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.12;

import "../samples/HybridAccount.sol";

contract TestCounter {
    mapping(address => uint256) public counters;

    address payable immutable demoAddr;

    constructor(address payable _demoAddr) {
        demoAddr = _demoAddr;
    }
}

```

Now let's add the `count()` method. We initialize the `HybridAccount` with the `demoAddr` created prior. We define `x` and `y`, do a quick check for ``b == 0``, and encode our function ``addsub2()``. The real magic happens with ``HA.CallOffChain(userkey, req)``; this call will return us the two numbers `a` and `b` (assuming no error). If we encounter an error during the `CallOffChain()` call, we either revert or set the counter. See the two `else if` statements for more information.

Starting in the `count()` function, we initialize a `HybridAccount` along the with the address used when we deployed the smart contract:

```solidity

    function count(uint32 a, uint32 b) public {
        HybridAccount HA = HybridAccount(demoAddr);
        uint256 x;
        uint256 y;
        if (b == 0) {
            counters[msg.sender] = counters[msg.sender] + a;
            return;
        }
        bytes memory req = abi.encodeWithSignature("addsub2(uint32,uint32)", a, b);
        bytes32 userKey = bytes32(abi.encode(msg.sender));
        (uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

        if (error == 0) {
            (x, y) = abi.decode(ret, (uint256, uint256)); // x=(a+b), y=(a-b)

            this.gasWaster(x, "abcd1234");
            counters[msg.sender] = counters[msg.sender] + y;
        } else if (b >= 10) {
            revert(string(ret));
        } else if (error == 1) {
            counters[msg.sender] = counters[msg.sender] + 100;
        } else {
            //revert(string(ret));
            counters[msg.sender] = counters[msg.sender] + 1000;
        }
    }
```

Lastly, we define a function `countFail()` as well as `justemit()`, which will be used to emit the event `CalledFrom`.

```solidity
  function countFail() public pure {
      revert("count failed");
  }

  function justemit() public {
      emit CalledFrom(msg.sender);
  }

  event CalledFrom(address sender);

  //helper method to waste gas
  // repeat - waste gas on writing storage in a loop
  // junk - dynamic buffer to stress the function size.
  mapping(uint256 => uint256) public xxx;
  uint256 public offset;

  function gasWaster(uint256 repeat, string calldata /*junk*/) external {
      for (uint256 i = 1; i <= repeat; i++) {
          offset++;
          xxx[offset] = i;
      }
  }
```

The `HybridAccount` contract has been previously registered to provide access to the `addsub2()` function on our off-chain function. We'll explain more about this later.

## Calling Offchain

As already mentioned in the previous section, our off-chain server maps the request made by the bundler via the hashed representation of our function-signature. Let's decode the function-signature we want to call on the off-chain server:

```solidity
bytes memory req = abi.encodeWithSignature("addsub2(uint32,uint32)", a, b);
bytes32 userKey = bytes32(abi.encode(msg.sender));
(uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

require(result == HC_ERR_NONE, "Offchain call failed");
(x,y) = abi.decode(ret,(uint256,uint256)); // x=(a+b), y=(a-b)
```

:::tip
The above code block presents us with our first encounter of the acronym "HC". This stands for Hybrid Compute.
:::

We then generate a `userKey` by encoding `msg.sender`. The `userKey` parameter is used to distinguish requests so that they may be processed concurrently without interefering with each other.

Within the Hybrid Account contract itself, the `CallOffchain()` method calls through to another system contract named `HCHelper`:

``` solidity
function CallOffchain(bytes32 userKey, bytes memory req) public returns (uint32, bytes memory) {
   require(PermittedCallers[msg.sender], "Permission denied");
   IHCHelper HC = IHCHelper(_helperAddr);
   userKey = keccak256(abi.encodePacked(userKey, msg.sender));
   return HC.TryCallOffchain(userKey, req);
}
```

In this example, the `HybridAccount` implements a simple whitelist of contracts which are allowed to call its methods. It would also be possible for a `HybridAccount` to implement additional logic here, such as requiring payment of an `ERC20` token to perform an off-chain call. Conversely, the owner of a `HybridAccount` could choose to make the `CallOffchain()` method available to all callers without restriction.

There is an opportunity for a `HybridAccount` contract to implement a billing system here, requiring a payment of `ERC20` tokens or some other mechanism of collecting payment from the calling contract. This is optional.

## Helper Contract Implementation

```solidity
function TryCallOffchain(bytes32 userKey, bytes memory req) public returns (uint32, bytes memory) {
    bool found;
    uint32 result;
    bytes memory ret;

    bytes32 subKey = keccak256(abi.encodePacked(userKey, req));
    bytes32 mapKey = keccak256(abi.encodePacked(msg.sender, subKey));

    (found, success, ret) = getEntry(mapKey);

    if (found) {
        return (result, ret);
    } else {
        // If no off-chain response, check for a system error response.
        bytes32 errKey = keccak256(abi.encodePacked(address(this), subKey));

        (found, result, ret) = getEntry(errKey);
        if (found) {
            require(result != HC_ERR_NONE, "Invalid error code");
            return (result, ret);
        } else {
            // Nothing found, so trigger a new request.
            bytes memory prefix = "_HC_TRIG";
            bytes memory r2 = bytes.concat(prefix, abi.encodePacked(msg.sender, userKey, req));
            assembly {
                revert(add(r2, 32), mload(r2))
            }
        }
    }
}
```

In the code above, the contract checks an internal mapping to see if a response exists for the given request. If not, the method reverts with a special prefix, followed by an encoded version of the request parameters. 

If a response does exist, it's removed from the internal mapping and is returned to the caller. The map key encodes the request parameters, so that a response initiated by one request will not be returned later in response to a modified request from the caller.

To populate the response mapping, `HybridAccount` contracts use another method in the Helper:

```solidity
function PutResponse(bytes32 subKey, bytes calldata response) public {
    //require(msg.sender == address(this)); // _requireFromEntryPointOrOwner();
    require(RegisteredCallers[msg.sender].owner != address(0), "Unregistered caller");
    //require(ResponseCache[mapKey].length == 0, "Cache entry already exists");

    require(response.length >= 32, "Response too short");
    bytes32 mapKey = keccak256(abi.encodePacked(msg.sender, subKey));
    ResponseCache[mapKey] = response;
}
```

Note that the `msg.sender` is included in the calculation of the internal map key, ensuring that only `HybridAccount` is able to populate the response (which it will later receive back in the `TryCallOffchain()` call). However, in the case of an error result of `(success == false)`, there's also a provision for the HC implementation to insert a result under a different map key.

Account Abstraction calls `PutResponse()` and the off-chain `userOperation` must carry a valid signature in order to execute the operation.

## Read the Response Data

To retrieve the response from our `AddSub` contract, we handle the off-chain call as follows:

```solidity
(uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

if (error == 0) {
(x, y) = abi.decode(ret, (uint256, uint256)); // x=(a+b), y=(a-b)
}
```

In this snippet, we decode the returned object `ret` into two `uint256` values, as the off-chain function returns two integers. The variables `x` and `y` will hold the results of the addition and subtraction, respectively.

Now that we've written the smart contract, proceed to the next section to learn how to deploy it.
