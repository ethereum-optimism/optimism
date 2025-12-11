# Deploy the Smart Contracts

A [`deploy-local.py`](https://github.com/bobanetwork/rundler-hc/blob/boba-develop/hybrid-compute/deploy-local.py) script is provided to deploy and configure the necessary contracts on a local stack, launched from `make devnet-up` in a Boba/Optimism repository.

This is an example of how to execute it:

```bash
cd rundler-hc/hybrid-compute
python ./deploy-local.py --boba-path /home/user/boba  # Supply the path where you checked out the repo
```

This script uses the Foundry toolkit to deploy the base contracts and a set of examples, via Forge scripts located in `rundler-hc/crates/types/contracts/hc_scripts`.

The Forge scripts are written in Solidity, with toolkit extensions to support scripting, such as reading system environment variables and using them within Solidity functions. Code which executes within a block (like the following) is translated into a transaction which is first simulated locally and then executed on-chain:

```solidity
vm.startBroadcast(deployerPrivateKey);

// ... do stuff

vm.stopBroadcast();
```

The addresses of newly deployed contracts are then returned as an array of values:

```solidity
// These contracts are one example each
ret[0] = address(new AuctionFactory(ha1Addr));
ret[1] = address(new TestCaptcha(ha1Addr));
ret[2] = address(new TestCounter(ha1Addr));
ret[3] = address(new RainfallInsurance(ha1Addr));
ret[4] = address(new SportsBetting(ha1Addr));
```

Within `deploy-local.py` the Forge scripts are invoked like this:

```python
  args = ["forge", "script", "--json", "--broadcast", "--silent"]
  args.append ("--rpc-url=http://127.0.0.1:9545")
  args.append("hc_scripts/LocalDeploy.s.sol")
  cmd_env = os.environ.copy()
  cmd_env['PRIVATE_KEY'] = deploy_key
  cmd_env['HC_SYS_OWNER'] = env_vars['HC_SYS_OWNER']
  cmd_env['DEPLOY_ADDR'] = deploy_addr
  cmd_env['DEPLOY_SALT'] = cli_args.deploy_salt  # Update to force redeployment
```

The Forge scripts may be run manually by setting the corresponding environment variables
and then using the `forge script` command as documented at https://book.getfoundry.sh/reference/forge/forge-script

:::info
We deploy with Foundry, but you may deploy with whatever tooling you wish.
:::

Further system setup is performed through the `web3.py` API, first by connecting to our L1 and L2 networks:

```python
l1 = Web3(Web3.HTTPProvider("http://127.0.0.1:8545"))
assert (l1.is_connected)
l1.middleware_onion.inject(geth_poa_middleware, layer=0)

w3 = Web3(Web3.HTTPProvider(env_vars['NODE_HTTP']))
assert (w3.is_connected)
```

Next, we access the contracts using a `loadContract()` function, such as
for our `HybridAccount` contract:

```python
HA = loadContract(w3, "HybridAccount", path_prefix+"samples/HybridAccount.sol")
```

Here `path_prefix` indicates where the contract's source code is located. The current deployer script
will compile the Solidity code itself to determine the contract ABI rather than requiring a JSON file
from a previous deployment.

After loading the contract, we can interact with it like this:

```python
def permitCaller(acct, caller):
    if not acct.functions.PermittedCallers(caller).call():
        print("Permit caller {} on {}".format(caller, acct.address))
        tx = acct.functions.PermitCaller(caller, True).build_transaction({
            'nonce': w3.eth.get_transaction_count(env_vars['OC_OWNER']),
            'from': env_vars['OC_OWNER'],
            'gas': 210000,
            'chainId': HC_CHAIN,
        })
        signAndSubmit(w3, tx, env_vars['OC_PRIVKEY'])

# ...

for a in example_addrs:
  permitCaller(HA, a)

```

Here we perform a sequence of transactions to add individual example contract addresses to the
list of contracts which are permitted to use our `HybridAccount`. There is a check so that this operation
is only performed for contracts which have not yet been registered.

## Additional Examples

The preceding documentation was written to focus on the example showing addition and subtraction of
two numbers. Additionally, the `hybrid-compute/` folder contains more examples that can be used and
experimented with. As Hybrid Compute in general is under active development, some of these examples
may not be up to date with recent changes.

To integrate additional methods into our offchain-RPC `server-loop`:

```python
  def server_loop():
    server = SimpleJSONRPCServer(
        ('0.0.0.0', PORT),
        requestHandler=RequestHandler
    )

    // fetchPrice
    server.register_function(offchain_getprice, selector("fetchPrice(string)"))

    // Add Sub
    server.register_function(offchain_addsub2, selector("addsub2(uint32,uint32)"))  # 97e0d7ba

    // Ramble
    server.register_function(offchain_ramble,  selector("ramble(uint256,bool)"))

    // CheckKyc
    server.register_function(offchain_checkkyc, selector("checkkyc(string)"))
```

You can see how these methods are invoked from the `TestCounter.sol` contract. For example, "ramble" is 
used in a simple word-guessing game:

```solidity
    function wordGuess(string calldata myGuess, bool cheat) public payable {
        HybridAccount HA = HybridAccount(payable(demoAddr));
        uint256 entries = msg.value / EntryCost;
        require(entries > 0, "No entries purchased");
        require(entries <= 100, "Excess payment");
        Pool += msg.value;
        require(bytes(myGuess).length == 4, "Game uses 4-letter words");

        bytes memory req = abi.encodeWithSignature("ramble(uint256,bool)", entries, cheat);
        bytes32 userKey = bytes32(abi.encode(msg.sender));
        (uint32 error, bytes memory ret) = HA.CallOffchain(userKey, req);
        if (error != 0) {
            revert(string(ret));
        }

        uint256 win = 0;
        string[] memory words = abi.decode(ret,(string[]));

        for (uint256 i=0; i<words.length; i++) {
            if (keccak256(bytes(words[i])) == keccak256(bytes(myGuess))) {
	        win = Pool;  // Safe if there's more than one match in the list
            }
        }

        if (win == Pool) {
            Pool = 0;
            (bool sent,) = msg.sender.call{value: win}("");
            require(sent, "Payment failed");
        }
        emit GameResult(msg.sender,win,Pool);
    }
```

Note the common pattern of preparing an ABI-encoded request, invoking `HA.CallOffchain(userKey, req)`, and then processing the response. This is the core of the Hybrid Compute framework, and the rest is up to you. We look forward to seeing your ideas and applications.

You've now reached the end of this tutorial! These examples aim to demonstrate the different functions and best practices you should keep in mind as you develop your own efficient, flexible smart contracts with Hybrid Compute.
