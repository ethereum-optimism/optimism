---
description: Learn how to use Turing hybrid compute
---

<figure><img src="../../assets/hc-under-upgrade.png" alt=""><figcaption></figcaption></figure>

# Implementation

Hybrid Compute (formerly "Turing") is a system for interacting with the outside world from within solidity smart contracts. All data returned from external APIs, such as social networking or weather data are deposited into a public data-storage contract on Ethereum Mainnet. This extra data allows replicas, verifiers, and fraud-detectors to reproduce and validate the Boba L2 blockchain, block by block.

Using Hybrid Compute is as easy as calling specific functions from inside your smart contract. For example, to obtain a random number for minting NFTs, call:

```javascript

  // ERC721.sol
  random_number = turing.getRandom()

  // Test/Debug Response
  Turing NFT Random 256
    256 bit random number as a BigInt = 61245594159531997717158776666900035572992757857563713350570408643552830626492n
    Minted an NFT with Attribute A = 135 and Attribute B = 103
    Minted a pirate with a green hat
    ✓ should mint an NFT with random attributes (65ms)

```

To obtain **Twitter** or **Spotify** data you could set up a system like this:

```javascript

  urlStr = 'https://_myAPIURL_/social'
  likes = social.getCurrentLikes(tweetUniqueID)

  // Test/Debug response
  Tweet 123456789 had: 18 likes by time: 1650534735

```

###

<figure><img src="../../assets/feature highlights.png" alt=""><figcaption></figcaption></figure>

### Feature Highlight 1: Using Turing to mint an NFT with 256 random attributes in a single transaction

With Turing, your ERC721 contract can generate a cryptographically strong 256 bit random number immediately prior to the execution flow moving to the `mint` function. This is an _atomic_ transaction - everything takes places within one transaction:

```javascript

  // modified mint function in a standard ERC721.sol
  function mint(address to, uint256 tokenId) public {
    uint256 result = myHelper.TuringRandom();
    bytes memory i_bytes = abi.encodePacked(result);
    uint8 attribute_1  = uint8(i_bytes[ 0]);
    uint8 attribute_2  = uint8(i_bytes[ 1]);
    ...
    uint8 attribute_32 = uint8(i_bytes[31]);
    // use the attributes here to e.g. set URI/Attributes etc
    _mint(to, tokenId);
    emit MintedRandom(result, attribute_1, attribute_2, ...);
  }

  // pseudocode transaction response from test system (see boba/turing/test/006_NFT_random.ts)
  256 bit random number as a BigInt = 61245594159531997717158776666900035572992757857563713350570408643552830626492n
  Minted an NFT with Attribute A = 135 and Attribute B = 103
  Minted a pirate with a green hat
  ✓ should mint an NFT with random attributes (65ms)

```

To use this functionality, deploy your `TuringHelper` contract, provide its address to your ERC721 contract, and make the `TuringHelper` aware of the new caller:

```javascript

  // deploy your Turing helper
  myTuringHelper = await Factory__Helper.deploy()

  // deploy your ERC721 contract with the
  erc721 = await Factory__ERC721.deploy("RandomERC721", "RER", myTuringHelper.address)

  // restrict your myHelper to accept only requests from your ERC721
  await myTuringHelper.addPermittedCaller(erc721.address)

```

Then, register and fund your Turing Credit account:

```javascript

  const ONE_BOBA = utils.parseEther('1')
  await turingCredit.addBalanceTo(ONE_BOBA, myTuringHelper.address)

```

**All done**! Each Turing request costs 0.01 BOBA, so 1 BOBA is enough for 100 Turing requests. Have fun. You can find [example code and an ERC721 that uses Turing here](test/006\_NFT\_random.ts) and a [fully-featured Turing-ready NFT system here](../../boba\_community/turing-monsters/).

### Feature Highlight 2: Using Turing to access APIs from within your solidity smart contract

You can use Turing as a pipe to any other computer, such as APIs for social networks, weather and location data, or market data. Please keep in mind however that Turing differs sharply from established providers of market trading data, in particular, since **Turing does not provide a decentralized mechanism to verify the accuracy of the data**. **You should therefore not use Turing for production trading or lending use, but should use proven, decentralized data oracles**.

**Data/Oracle best practices** Regardless of your specific use case, minimally, you will need to secure your pipe/contract against data outliers, temporary lack of data, and malicious attempts to distort the data. For example, you could average over multiple on-chain oracles and/or off-chain sources - in this case, the role of Turing could be to 'augment' or separately estimate the reliability and timeliness of on-chain oracles.

**Note - Boba does not provide endpoints for you** You are responsible for setting up an endpoint that Turing can access - read on for more information and example code. Assume you have an API access key to a provider of weather data. First, set up a server or endpoint that queries this API, and stores and analyzes the data, if needed. Your own server/endpoint contains your secrets and API access keys. Next, add a simple interface to allow Turing to interact with your server. Turing calls to your server contain the address of the calling contract and there are multiple ways to control access to your server in very granular manner, if desired. See `.packages/boba/turing/AWS_code/turing_oracle.py` for a copy-paste example for querying data APIs via a wrapper at AWS Lambda:

```python
/AWS_code/turing_oracle.py

# Note - This code is running on YOUR server

...
  api_key = 'YOUR_API_KEY' # Insert your API key here

  authorized_contract = None # for open access
  # or...
  authorized_contract = '0xOF_YOUR_HELPER_CONTRACT' # to restrict access to only your smart contract
...

```

You should lock down your off-chain endpoint to only accept queries from your smart contract. To do this, designate your smart contract's address on Boba as the `authorized_contract`. If you wish to allow open access, set this variable to `None`. You can then call this API in your smart contract:

```javascript

  urlStr = 'https://_myAPIURL_/social'
  likes = social.getCurrentLikes(tweetUniqueID)

    // Test/Debug response
    Tweet 123456789 had: 18 likes by time: 1650534735

```

<figure><img src="../../assets/aws and google cloud functions.png" alt=""><figcaption></figcaption></figure>

Your external API will need to accept calls from the L2Geth and return data in a way that can be understood by the L2Geth. Examples are provided in `./packages/boba/turing/AWS_code`. Specific instructions for setting up AWS lambda endpoints are [here](AWS\_code/AWS\_lambda\_setup.md) - note that _all_ APIs can be used, not just AWS Lambda endpoints.

<figure><img src="../../assets/important properties of hybrid.png" alt=""><figcaption></figcaption></figure>

* Only one Turing call per execution
* There is **1200 ms timeout** on API responses. Please make sure that your API responds promptly. If you are using AWS, note that some of their services take several seconds to spin up from a 'coldstart', resulting in persistent failure of your first call to your endpoint.

#### One Turing call per Transaction

At present, you can only have one Turing call per transaction, i.e. a Turing call cannot call other contracts that invoke Turing as well. Transactions that result in multiple Turing calls in the call stack will revert.

#### TuringTxV2() - V2

The new HybridCompute version allows endpoints to return more complex data structures or just larger payloads in general.

* Maximum allowed length of the modified calldata is 65508 (selector + req + response).

When using the new TuringTxV2() function, then you don't need to supply a length parameter on the backend.

```javascript
  // Payload from the external API
  // 0x
  // 0000000000000000000000000000000000000000000000000000000000418b95 ** first uint256
  // 0000000000000000000000000000000000000000000000000000017e60d3b45f ** second uint256

  // decoding of those data within the smart contract
  (uint256 market_price, uint256 time) = abi.decode(encResponse,(uint256,uint256));
```

To return more complex types, we recommend using an AbiEncoder instead of building the payload manually.

```python
from eth_abi import encode_abi, decode_abi
from web3 import Web3

encoded_str = encode_abi(['string[]', 'string', 'bytes'], [yourArray, aStringVal, someBytes])
res = Web3.toHex(encoded_str)
```

On the smart contract side, it could look like this:

```solidity
bytes memory byteRes = turingHelper.TuringTxV2(_turingUri, encRequest);

(string[] memory yourArray, string memory aStringVal, bytes memory someBytes) = abi.decode(byteRes, (string[], string, bytes));
```

#### TuringTx() - V1

The legacy version of Hybrid Compute had a smaller maximum response size, which didn't allow you to return larger or more complex payloads from an API such as arrays.

* Strings returned from external endpoints in V1 are limited to 322 characters (`5*64+2=322`)

**String length limit**

The string length cap of 322 is large enough to return, for example, four `uint256` from the external api:

```javascript
//example: returing 4 unit264

  // 0x
  // 0000000000000000000000000000000000000000000000000000000000000080 ** length of the dynamic bytes
  // 0000000000000000000000000000000000000000000000000000000000418b95 ** first uint256
  // 0000000000000000000000000000000000000000000000000000017e60d3b45f **
  // 0000000000000000000000000000000000000000000000000000000000eb7ca3 **
  // 00000000000000000000000000000000000000000000000000000000004c788f ** fourth unit265

```

You can return anything you want - e.g. numbers, strings, ... - and this information will then later be decoded per your `abi.decode`. For example, if the external API sends two `unit256`:

```javascript

  // Payload from the external API
  // 0x
  // 0000000000000000000000000000000000000000000000000000000000000040 ** length of the dynamic bytes
  // 0000000000000000000000000000000000000000000000000000000000418b95 ** first uint256
  // 0000000000000000000000000000000000000000000000000000017e60d3b45f ** second uint256

  // decoding of those data within the smart contract
  (uint256 market_price, uint256 time) = abi.decode(encResponse,(uint256,uint256));

```

<figure><img src="../../assets/hybridcompute architecture.png" alt=""><figcaption></figcaption></figure>

The modified Turing L2Geth, `L2TGeth`, monitors calldata for particular Keccak methodIDs of functions such as `GetRandom(uint32 rType, uint256 _random)` and `GetResponse(uint32 rType, string memory _url, bytes memory _payload)`. Upon finding such methodIDs in the execution flow, at any level, L2TGeth parses the calldata for additional information, such as external URLs, and uses that information to either directly prepare a response (e.g. generate a random number) or to call an external API. After new information is generated (or has returned from the external API), L2TGeth then runs the function with updated inputs, such that the new information flows back to the caller (via overloaded variables and a system for conditionally bypassing `requires`). Put simply, L2TGeth intercepts function calls, adds new information to the inputs, and then runs the function with the updated inputs.

In general, this system would lead to disagreement about the correct state of the underlying blockchain. For example, if replicas and verifiers simply ingested the transactions and re-executed them, then every blockchain would differ, destroying the entire system. Thus, a new data field called `Turing` (aka `turing`, `l1Turing` or `L1Turing` depending on context) has been added to the L2Geth `transactions`,`messages`, `receipts`, `blocks`, `evm.contexts`, and various `codecs` and `encoders/decoders`. This new data field is understood by `core-utils` as well as the `data-translation-layer` and the `batch-submitter`, and allows Turing data to be pushed into, and recovered from, the `CanonicalTransactionChain` (CTC). This extra information allows all verifiers and replicas to enter a new **replay** mode, where instead of generating new random numbers (or calling off-chain for new data), they use the Turing data stored in the CTC (or in the L2 blocks as part of the transaction metadata) to generate a faithful copy of the main Boba L2 blockchain. Thus, the overall system works as before, with all the information needed for restoring the Boba L2 and, just as critically, for public fraud detection, being publicly deposited into Ethereum.

<figure><img src="../../assets/quickstart for hybrid.png" alt=""><figcaption></figcaption></figure>

Open a terminal window and from the top level:

```bash
$ yarn
$ yarn build
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```

This will spin up the stack. Then, open a second terminal window and:

```bash
$ cd packages/boba/turing
$ yarn test:local
```

**Note: Testing on Rinkeby**

To test on Rinkeby, you need a private key with both ETH and BOBA on the Boba L2; the private key needs to be provided in `hardhat.config.js`. Just replace all the zeros with your key:

```javascript
    boba_rinkeby: {
      url: 'https://rinkeby.boba.network',
      accounts: ['0x0000000000000000000000000000000000000000000000000000000000000000']
    },
```

Then, run:

```bash
$ cd packages/boba/turing
$ yarn test:rinkeby
```

The tests will perform some basic floating point math, provide some random numbers, and get the latest BTC-USD exchange rate:

```bash
yarn run v1.22.15
$ hardhat --network boba_local test

  Stableswap at AWS Lambda
    URL set to https://i9iznmo33e.execute-api.us-east-1.amazonaws.com/swapy
    Helper contract deployed as 0x8e264821AFa98DD104eEcfcfa7FD9f8D8B320adA
    Stableswap contract deployed as 0x871ACbEabBaf8Bed65c22ba7132beCFaBf8c27B5
    addingPermittedCaller to TuringHelper 0x000000000000000000000000871acbeabbaf8bed65c22ba7132becfabf8c27b5
    Test contract whitelisted in TuringHelper (1 = yes)? 1
    ✓ contract should be whitelisted (50ms)
    Credit Prebalance 0
    BOBA Balance in your account 300000000000000000000
    ✓ Should register and fund your Turing helper contract in turingCredit (172ms)
    ✓ should return the helper address (116ms)
      result of x_in 12 -> y_out = 50
    ✓ should correctly swap X in for Y out (202ms)

  Turing 256 Bit Random Number
    Helper contract deployed at 0xb185E9f6531BA9877741022C92CE858cDCc5760E
    Test contract deployed at 0xAe120F0df055428E45b264E7794A18c54a2a3fAF
    addingPermittedCaller to TuringHelper 0x000000000000000000000000ae120f0df055428e45b264e7794a18c54a2a3faf
    Test contract whitelisted in TuringHelper (1 = yes)? 1
    ✓ contract should be whitelisted (51ms)
    Credit Prebalance 0
    BOBA Balance in your account 290000000000000000000
    ✓ Should register and fund your Turing helper contract in turingCredit (174ms)
    Turing 42 = 42
    ✓ should get the number 42 (91ms)
    Turing VRF 256 = 11642062518220346831211086370276871135010213271872466428492348202384902597141n
    ✓ should get a 256 bit random number (83ms)
    Turing VRF 256 = 39492154036951735205025381980653780356965271743173916331971607322325246415525n
    ✓ should get a 256 bit random number (83ms)

  Pull Bitcoin - USD quote
    URL set to https://i9iznmo33e.execute-api.us-east-1.amazonaws.com/quote
    Helper contract deployed as 0x7C8BaafA542c57fF9B2B90612bf8aB9E86e22C09
    Lending contract deployed as 0x0a17FabeA4633ce714F1Fa4a2dcA62C3bAc4758d
    addingPermittedCaller to TuringHelper 0x0000000000000000000000000a17fabea4633ce714f1fa4a2dca62c3bac4758d
    Test contract whitelisted in TuringHelper (1 = yes)? 1
    ✓ contract should be whitelisted (53ms)
    ✓ should return the helper address
    Credit Prebalance 0
    BOBA Balance in your account 280000000000000000000
    ✓ Should register and fund your Turing helper contract in turingCredit (176ms)
    Bitcoin to USD price is 36654.89
    timestamp 1643158948154
    ✓ should get the current Bitcoin - USD price (305ms)

  Turing NFT Random 256
    Turing Helper contract deployed at 0xd9fEc8238711935D6c8d79Bef2B9546ef23FC046
    ERC721 contract deployed at 0xd3FFD73C53F139cEBB80b6A524bE280955b3f4db
    adding your ERC721 as PermittedCaller to TuringHelper 0x000000000000000000000000d3ffd73c53f139cebb80b6a524be280955b3f4db
    Credit Prebalance 0
    BOBA Balance in your account 270000000000000000000
    ✓ Should register and fund your Turing helper contract in turingCredit (122ms)
    ERC721 contract whitelisted in TuringHelper (1 = yes)? 1
    ✓ Your ERC721 contract should be whitelisted
    256 bit random number as a BigInt = 61245594159531997717158776666900035572992757857563713350570408643552830626492n
    Minted an NFT with Attribute A = 135 and Attribute B = 103
    Minted a pirate with a green hat
    ✓ should mint an NFT with random attributes (65ms)


  22 passing (3s)

✨  Done in 6.67s.
```

<figure><img src="../../assets/technical appendix.png" alt=""><figcaption></figcaption></figure>

#### Step 1: Invoking Turing for inside a Smart contract

A Turing cycle starts with specific function calls inside solidity smart contracts deployed on Boba:

```javascript

  random_number = turing.getRandom()

```

The modified `L2TGeth` detects these function calls, intercepts them, and obtains requested data from other sources (strong random number generators, off-chain APIs and datafeeds, ...).

```go
/l2geth/core/vm/evm.go

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {

...

  //methodID for GetResponse is 7d93616c -> [125 147 97 108]
  isTuring2 := bytes.Equal(input[:4], []byte{125, 147, 97, 108})

  //methodID for GetRandom is 493d57d6 -> [73 61 87 214]
  isGetRand2 := bytes.Equal(input[:4], []byte{73, 61, 87, 214})

  // TuringCall takes the original calldata, figures out what needs
  // to be done, and then synthesizes a 'updated_input' calldata
  var updated_input hexutil.Bytes

  if isTuring2 {
    if len(evm.Context.Turing) < 3 {
      // This is the first run of Turing for this transaction
      // We sometimes use a short evm.Context.Turing payload for debug purposes.
      // A real modified callData is always much much > 2 bytes
      // This case _should_ never happen in Verifier/Replica mode, since the sequencer will already have run the Turing call
      updated_input = bobaTuringCall(input, caller.Address())
      ret, err = run(evm, contract, updated_input, false)
      // and now, provide the updated_input to the context so that the data can be sent to L1 and the CTC
      /**************** CRITICAL LINE ****************/
      evm.Context.Turing = updated_input
      /**************** CRITICAL LINE ****************/
    } else {
      // Turing for this Transaction has already been run elsewhere - replay using
      // information from the EVM context
      ret, err = run(evm, contract, evm.Context.Turing, false)
    }
  } else if isGetRand2 {
    if len(evm.Context.Turing) < 3 {
      // See above - they apply 1:1 here too
      updated_input = bobaTuringRandom(input)
      ret, err = run(evm, contract, updated_input, false)

      /**************** CRITICAL LINE ****************/
      evm.Context.Turing = updated_input
      /**************** CRITICAL LINE ****************/
    } else {
      // Turing for this Transaction has already been run elsewhere - replay using
      // information from the EVM context
      ret, err = run(evm, contract, evm.Context.Turing, false)
    }
  } else {
    ret, err = run(evm, contract, input, false)
  }

...

```

The random number generation is done locally, inside the Geth, and off-chain APIs are queried with standard calls:

```go
/l2geth/core/vm/evm.go

// In response to an off-chain Turing request, obtain the requested data and
// rewrite the parameters so that the contract can be called without reverting.
func bobaTuringRandom(input []byte) hexutil.Bytes {

  var ret hexutil.Bytes

  rest := input[4:]

  //some things are easier with a hex string
  inputHexUtil := hexutil.Bytes(input)

  // If things fail, we'll return an integer parameter which will fail a
  // "require" in the contract.
  retError := make([]byte, len(inputHexUtil))
  copy(retError, inputHexUtil)

  // Check the rType
  // 1 for Request, 2 for Response, integer >= 10 for various failures
  rType := int(rest[31])
  if rType != 1 {
    log.Warn("TURING-1 bobaTuringRandom:Wrong state (rType != 1)", "rType", rType)
    retError[35] = 10 // Wrong input state
    return retError
  }

  rlen := len(rest)
  if rlen < 2*32 {
    log.Warn("TURING-2 bobaTuringRandom:Calldata too short", "len < 2*32", rlen)
    retError[35] = 11 // Calldata too short
    return retError
  }

  // Generate cryptographically strong pseudo-random int between 0 - 2^256 - 1
  one := big.NewInt(1)
  two := big.NewInt(2)
  max := new(big.Int)
  // Max random value 2^256 - 1
  max = max.Exp(two, big.NewInt(int64(256)), nil).Sub(max, one)
  n, err := rand.Int(rand.Reader, max)

  if err != nil {
    log.Warn("TURING bobaTuringRandom: Random Number Generation Failed", "err", err)
    retError[35] = 16 // RNG Failure
    return retError
  }

  //generate a BigInt random number
  randomBigInt := n

  // build the calldata
  methodID := make([]byte, 4)
  copy(methodID, inputHexUtil[0:4])
  ret = append(methodID, hexutil.MustDecode(fmt.Sprintf("0x%064x", 2))...) // the usual prefix and the rType, now changed to 2
  ret = append(ret, hexutil.MustDecode(fmt.Sprintf("0x%064x", randomBigInt))...)

  return ret
}

// In response to an off-chain Turing request, obtain the requested data and
// rewrite the parameters so that the contract can be called without reverting.
func bobaTuringCall(input []byte, caller common.Address) hexutil.Bytes {

  var responseStringEnc string
  var responseString []byte

  rest := input[4:]
  inputHexUtil := hexutil.Bytes(input)
  restHexUtil := inputHexUtil[4:]

  retError := make([]byte, len(inputHexUtil))
  copy(retError, inputHexUtil)

  // Check the rType
  // 1 for Request, 2 for Response, integer >= 10 for various failures
  rType := int(rest[31])
  if rType != 1 {
    retError[35] = 10 // Wrong input state
    return retError
  }

  rlen := len(rest)
  if rlen < 7*32 {
    retError[35] = 11 // Calldata too short
    return retError
  }

  // A micro-ABI decoder... this works because we know that all these numbers can never exceed 256
  // Since the rType is 32 bytes and the three headers are 32 bytes each, the max possible value
  // of any of these numbers is 32 + 32 + 32 + 32 + 64 = 192
  // Thus, we only need to read one byte

  // 0  -  31 = rType
  // 32  -  63 = URL start
  // 64  -  95 = payload start
  // 96  - 127 = length URL string
  // 128 - ??? = URL string
  // ??? - ??? = payload length
  // ??? - end = payload

  startIDXurl := int(rest[63]) + 32
  // the +32 means that we are going directly for the actual string
  // bytes 0 to 31 are the string length

  startIDXpayload := int(rest[95]) // the start of the payload
  lengthURL := int(rest[127])      // the length of the URL string

  // Check the URL length
  // Note: we do not handle URLs that are longer than 64 characters
  if lengthURL > 64 {
    retError[35] = 12 // URL string > 64 bytes
    return retError
  }

  // The URL we are going to query
  endIDX := startIDXurl + lengthURL
  url := string(rest[startIDXurl:endIDX])
  // we use a specific end value (startIDXurl+lengthURL) since the URL is right-packed with zeros

  // At this point, we have the API endpoint and the payload that needs to go there...
  payload := restHexUtil[startIDXpayload:] //using hex here since that makes it easy to get the string

  log.Debug("TURING-4 bobaTuringCall:Have URL and payload",
    "url", url,
    "payload", payload)

  client, err := rpc.Dial(url)

  if client != nil {
    if err := client.Call(&responseStringEnc, caller.String(), payload); err != nil {
      retError[35] = 13 // Client Error
      return retError
    }
    responseString, err = hexutil.Decode(responseStringEnc)
    if err != nil {
      retError[35] = 14 // Client Response Decode Error
      return retError
    }
  } else {
    retError[35] = 15 // Could not create client
    return retError
  }

  // build the modified calldata
  ret := make([]byte, startIDXpayload+4)
  copy(ret, inputHexUtil[0:startIDXpayload+4]) // take the original input
  ret[35] = 2                                  // change byte 3 + 32 = 35 (rType) to indicate a valid response
  ret = append(ret, responseString...)         // and tack on the payload

  return ret
}

```

#### Step 2: Flow of Turing data out of the evm.context

`l2geth/core/state_processor.go:core.ApplyTransaction` moves the Turing data from `Context.Turing` into the `transaction.meta.L1Turing` byte array:

```go
l2geth/core/state_processor.go

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
...
  109   // Apply the transaction to the current state (included in the env)
  110   _, gas, failed, err := ApplyMessage(vmenv, msg, gp)
  111:  // TURING Update the tx metadata, if a Turing call took place...
  112   if len(vmenv.Context.Turing) > 1 {
  113     tx.SetL1Turing(vmenv.Context.Turing)
  114   }

```

The Turing data are subsequently incorporated into new L2 blocks via `w.engine.FinalizeAndAssemble` - the Turing data are in the `w.current.txs` input.

```go
l2geth/miner/worker.go:

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (w *worker) commit(uncles []*types.Header, interval func(), start time.Time) error {
...
 1110   s := w.current.state.Copy()
 1111   // log.Debug("TURING worker.go final block", "depositing_txs", w.current.txs)
 1112:  block, err := w.engine.FinalizeAndAssemble(w.chain, w.current.header, s, w.current.txs, uncles, w.current.receipts)
 1113   if err != nil {
 1114     return err

```

At this point, the data are circulated to various places throughout the system as part of the block/transaction data. Notably, calls to the L2 for block/transaction data now return a new field, `l1Turing` to all callers.

#### Step 3: Batch submitter Turing data injection

The batch submitter receives an new block/transaction from `L2TGeth`, obtains the raw call string (`rawTransaction`) and the Turing data (`l1Turing`), and if there was a Turing event (as judged from the length of the Turing string), the modified `batch-submitter` appends those data to the `rawTransaction` string. From the perspective of the CTC, it is receiving its normal batch payload.

```javascript
// batch-submitter tx-batch-submitter.ts

private async _getL2BatchElement(blockNumber: number): Promise<BatchElement> {

  // Idea - manipulate the rawTransaction as early as possible, so we do not have to change even more of the encode/decode
  // logic - note that this is basically adding a second encoder/decoder before the 'normal' one, which encodes total length
  //
  // The 'normal' one will now specify the TOTAL length (new_turing_header + rawTransaction + turing (if != 0)) rather than
  // just remove0x(rawTransaction).length / 2

...

  if (this._isSequencerTx(block)) {
    batchElement.isSequencerTx = true
    const turing = block.transactions[0].l1Turing
    let rawTransaction = block.transactions[0].rawTransaction
    if (turing.length > 4) {
      // FYI - we sometimes use short (length <= 4) non-zero Turing strings for debug purposes
      // Chop those off at this stage
      // Only propagate the data through the system if it's a real Turing payload
      const headerTuringLengthField = remove0x(BigNumber.from(remove0x(turing).length / 2).toHexString()).padStart(6, '0')
      rawTransaction = '0x' + headerTuringLengthField + remove0x(rawTransaction) + remove0x(turing)
    } else {
      rawTransaction = '0x' + '000000' + remove0x(rawTransaction)
    }
    batchElement.rawTransaction = rawTransaction
  }

```

#### Step 4: Writing to the CTC

The batch-submitter writes the data to the CTC as usual. **The CTC does not know about Turing** - that was one of the goals, so we do not have to modify the L1 contracts.

#### Step 5: DTL Turing data extraction; Reading from the CTC

The DTL reads from the CTC and unpacks the modified `rawTransaction` (which is now called `sequencerTransaction`). The DTL uses a Turing length metadata field in the `sequencerTransaction` string. Critically, the DTL writes a slightly modified `TransactionEntry` into its database, which has a new field called `turing`. When the database is queried, it thus returns the Turing data in addition to all the usual fields.

```javascript
// DTL services/l1-ingestion/handles/sequencer-batch-appended.ts

        for (let j = 0; j < context.numSequencedTransactions; j++) {
...

        // need to keep track of the original length so the pointer system for accessing
        // the individual transactions works correctly
        const sequencerTransaction_original_length = sequencerTransaction.length

        // This MIGHT have a Turing payload inside of it...
        // First, parse the new length field...
        const sTxHexString = toHexString(sequencerTransaction)
        const turingLength = parseInt(remove0x(sTxHexString).slice(0,6), 16)

        let turing = Buffer.from('0')

        if (turingLength > 0) {
          //we have Turing payload
          turing = sequencerTransaction.slice(-turingLength)
          sequencerTransaction = sequencerTransaction.slice(3, -turingLength)
          // The `3` chops off the Turing length header field, and the `-turingLength` chops off the Turing bytes
          console.log('Found a Turing payload at (neg) position:', {
            turingLength,
            turing: toHexString(turing),
            restoredSequencerTransaction: toHexString(sequencerTransaction),
          })
        } else {
          // The `3` chops off the Turing length header field, which is zero in this case (0: 00 1: 00 2: 00)
          sequencerTransaction = sequencerTransaction.slice(3)
        }

        transactionEntries.push({
...
          turing: toHexString(turing),
        })

```

#### Step 6: Verifier data ingestion

The Verifier receives all the usual data from the DTL, but, if there was a Turing call, there is now an additional data field containing the rewritten callData as a HexString. The Turing data are obtained from incoming `json` data and are written into the transaction metadata, `meta.L1Turing = turing`:

```go
/l2geth/core/types/transaction_meta.go:
   38   L1Timestamp     uint64          `json:"l1Timestamp"`
   39:  L1Turing        []byte          `json:"l1Turing" gencodec:"required"`
   40   L1MessageSender *common.Address `json:"l1MessageSender" gencodec:"required"`
   ..
   55   l1Timestamp uint64,
   56:  l1Turing []byte,
   57   l1MessageSender *common.Address,
   ..
   64       L1Timestamp:     l1Timestamp,
   65:      L1Turing:        l1Turing,
   66       L1MessageSender: l1MessageSender,
   ..
  145   }
  146
  147:  turing, err := common.ReadVarBytes(b, 0, 2048, "Turing")
  148   if err != nil {
  149       return nil, err
  150   }
  151:  if !isNullValue(turing) {
  152:      meta.L1Turing = turing
  153   }
```

At this point, the Turing data can be passed into the `evm.context`, which then triggers the `else` logic in the `evm.go`:

```go
/l2geth/core/vm/evm.go

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {

...

  //methodID for GetResponse is 7d93616c -> [125 147 97 108]
  isTuring2 := bytes.Equal(input[:4], []byte{125, 147, 97, 108})

  //methodID for GetRandom is 493d57d6 -> [73 61 87 214]
  isGetRand2 := bytes.Equal(input[:4], []byte{73, 61, 87, 214})

  // TuringCall takes the original calldata, figures out what needs
  // to be done, and then synthesizes a 'updated_input' calldata
  var updated_input hexutil.Bytes

  if isTuring2 {
    if len(evm.Context.Turing) < 3 {
...
    } else {
      // Turing for this Transaction has already been run elsewhere - replay using
      // information from the EVM context
      ret, err = run(evm, contract, evm.Context.Turing, false)
    }
  } else if isGetRand2 {
    if len(evm.Context.Turing) < 3 {
...
    } else {
      // Turing for this Transaction has already been run elsewhere - replay using
      // information from the EVM context
      ret, err = run(evm, contract, evm.Context.Turing, false)
    }
  } else {
    ret, err = run(evm, contract, input, false)
  }

...

```

The Turing data flow from out from the `evm.context` through the rest of the system as before, so the data are incorporated into verifier and replica blocks, resulting in correct/consistent state roots and replica and verifier blocks.
