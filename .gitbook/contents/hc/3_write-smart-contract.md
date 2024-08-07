## Step 3: Writing the Smart Contract

Now we can write the Smart Contract, which will call our previously created offchain-handler.
You can find the needed "HybridAccount"-Contract along with it's dependencies in the provided repository.

In the first part of the contract we are creating a mapping for the counters and we define a ``demoAddr``
This address will then be part of the the ``HybridAccount``.

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

Now let us add the count method. We initialize the `HybridAccount` with the `demoAddr` created prior. We define `x`
and `y`, do a quick check for ``b == 0`` and encode our function ``addsub2()``.
And the magic is going to happen within the ``HA.CallOffChain(userkey, req)``.
This call will return us the two numbers `a` and `b` given the fact that there has been no error. If we encounter an
error during the `CallOffChain` call, we either revert or set the counter. See the two `else if` statements for more
information.

Starting in the "count"-function, we initialize an "HybridAccount" along the with the address used when we deployed
the "Smart Contract"

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

Last but not least, we define a function `countFail` as well as `justemit` - which will be used to emit the
event `CalledFrom`.

And that's about it!

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

The "HybridAccount" contract has been previously registered to provide access to the "addsub2" function on our
offchain-function. But more on that later.

### Calling Offchain

As already mentioned in Step 2, our offchain-server maps the request, made by the bundler, via the hashed representation
of our function-signature. So let's decode the function-signature we want to call on the offchain-server:

```solidity
bytes memory req = abi.encodeWithSignature("addsub2(uint32,uint32)", a, b);
bytes32 userKey = bytes32(abi.encode(msg.sender));
(uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

require(result == HC_ERR_NONE, "Offchain call failed");
(x,y) = abi.decode(ret,(uint256,uint256)); // x=(a+b), y=(a-b)
```

We then generate an `userKey` by encoding "msg.sender". The `userKey` parameter is used to distinguish requests so that
they may be processed concurrently without interefering with each other.

Withing the Hybrid Account contract itself, the `CallOffchain` method calls through to another system contract named
`HCHelper`:

``` solidity
function CallOffchain(bytes32 userKey, bytes memory req) public returns (uint32, bytes memory) {
   require(PermittedCallers[msg.sender], "Permission denied");
   IHCHelper HC = IHCHelper(_helperAddr);
   userKey = keccak256(abi.encodePacked(userKey, msg.sender));
   return HC.TryCallOffchain(userKey, req);
}
```

In this example the HybridAccount implements a simple whitelist of contracts which are allowed to call its methods. It
would also be possible for a HybridAccount to implement additional logic here, such as requiring payment of an ERC20
token to perform an offchain call. Or conversely, the owner of a HybridAccount could choose to make the CallOffchain
method available to all callers without restriction.

There is an opportunity for a HybridAccount contract to implement a billing system here, requiring a payment of ERC20
tokens or some other mechanism of collecting payment from the calling contract. This is optional.

### Helper Contract Implementation

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

In the code above, the contract checks an internal mapping to see if a response exists for the given request. If not
then the method reverts with a special prefix, followed by an encoded version of the request parameters. If a response
does exist then it is removed from the internal mapping and is returned to the caller. The map key encodes the request
parameters, so that a response initiated by one request will not be returned later in response to a modified request
from the caller.

To populate the response mapping, HybridAccount contracts use another method in the Helper:

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

Note that the msg.sender is included in the calculation of the internal map key, ensuring that only that HybridAccount
is able to populate the response which it will later receive back in the `TryCallOffchain() `call. However in the case of
error results `(success == false)` there is also a provision for the HC implementation to insert a result under a
different map key.

`PutResponse()` is called using Account Abstraction and the offchain userOperation must carry a valid signature in order
for the operation to be executed.

### Reading the response data

To retrieve the response from our `AddSub` contract, we handle the offchain call as follows:

```solidity
(uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);

if (error == 0) {
(x, y) = abi.decode(ret, (uint256, uint256)); // x=(a+b), y=(a-b)
}
```

In this snippet, we decode the returned object `ret` into two `uint256` values, as the offchain function returns two
integers. The variables x and y will hold the results of the addition and subtraction, respectively.

