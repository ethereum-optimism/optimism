## Deploy the Smart Contract

With a few adjustments to the `deploy.py` script, we can easily deploy our newly created smart contract.

First, we need to configure the script to connect to our L1 and L2 networks:

```solidity
l1 = Web3(Web3.HTTPProvider("http://127.0.0.1:8545"))
assert(l1.is_connected)
l1.middleware_onion.inject(geth_poa_middleware, layer = 0)

l2 = Web3(Web3.HTTPProvider("http://127.0.0.1:9545"))
assert(l2.is_connected)
l2.middleware_onion.inject(geth_poa_middleware, layer = 0)
```

Next, we load the contract using the `loadContract()` function:

```solidity
TC = loadContract(w3, "TestCounter", path_prefix + "test/TestCounter.sol")
```

Here `path_prefix` indicates where the contract is located.

After loading the contract, we can deploy it as follows:

```solidity
epAddr = deploy2("EntryPoint", EP.constructor(), 0)
hhAddr = deploy2("HCHelper", HH.constructor(epAddr, boba_addr, 0), 0)
saAddr = deploy2("SimpleAccount", SA.constructor(epAddr), 0)
ha0Addr = deploy2("HybridAccount.0", HA.constructor(epAddr, hhAddr), 0)
ha1Addr = deploy2("HybridAccount.1", HA.constructor(epAddr, hhAddr), 1)
tcAddr = deploy2("TestCounter", TC.constructor(ha1Addr), 0)
```

The constructor of our smart contract takes an address as an argument. Therefore, we pass the address of
`HybridAccount.1`, which, along with other necessary contracts, is deployed as shown above.

## Additional Examples

The documentation above was precisely written for the addition of two numbers. The `hybrid-compute/` folder contains more examples that can be used and experimented with.

Let's integrate them into our `server-loop`:

```python
  def server_loop():
    server = SimpleJSONRPCServer(
        ('0.0.0.0', PORT),
        requestHandler=RequestHandler
    )

    // Add Sub
    server.register_function(offchain_addsub2, selector("addsub2(uint32,uint32)"))  # 97e0d7ba

    // Ramble
    server.register_function(offchain_ramble,  selector("ramble(uint256,bool)"))

    // CheckKyc
    server.register_function(offchain_checkkyc, selector("checkkyc(string)"))

    // getPrice
    server.register_function(offchain_getprice, selector("getprice(string)"))
```

You've now reached the end of this tutorial! These examples aim to demonstrate the different functions and best practices you should keep in mind as you develop your own efficient, flexible smart contracts with Hybrid Compute.
