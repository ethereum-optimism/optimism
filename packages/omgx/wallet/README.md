# OMGX Web Wallet, Related Smart Contracts, and Integration Tests

## New Wallet contract changes

There is now a completely different system for spinning up the system and contracts needed for the wallet.

* Spin up the test system and deploy all the right wallet contracts:

```

$ cd ops
$ docker-compose -f docker-compose.yml -f docker-compose-omgx-services.yml up

```

Or,

```bash
- name: Bring the stack up + OMGX
        working-directory: ./ops
        env:
          BUILD: 1
          DAEMON: 1
        run: ./up_local.sh
```

NOTE - the `up_local.sh` taps into ethereumoptimism dockers, be advised.

To get the contract addresses:

```bash
curl http://127.0.0.1:8078/addresses.json | jq
curl http://127.0.0.1:8080/addresses.json | jq
```

**ALERT - the old testing system and the documention below are currently broken, but are being fixed.**

# Working Steps for local setup.
## 1. Set up the repo

At the top level (`/optimism`), run `yarn` and `yarn build`.

```bash
$ git clone git@github.com:omgnetwork/optimism.git
$ cd optimism
$ yarn
$ yarn build
```

## 2. Spin up OMGX

```bash

$ cd /ops
$ docker-compose up --build

```
 OR

 Spin up the test system and deploy all the right wallet contracts:

```bash

$ cd ops
$ docker-compose -f docker-compose.yml -f docker-compose-omgx-services.yml up

```


## 3. Web wallet setup and Configuration - Contracts

Next, open a *second* terminal window and navigate to the wallet folder:

```bash

$ cd /optimism/packages/omgx/wallet

```

Create a `.env` file in the root directory `/optimism/packages/omgx/wallet` of this wallet project. Add environment-specific variables on new lines in the form of `NAME=VALUE`. Examples are given in the `.env.example` file. Just pick which net you want to work on and copy either the "Rinkeby" _or_ the "Local" envs to your `.env`.

Or,

Use below env params.

```bash
NODE_ENV=local
L1_NODE_WEB3_URL=http://localhost:9545
L2_NODE_WEB3_URL=http://localhost:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
TEST_PRIVATE_KEY_1=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
TEST_PRIVATE_KEY_2=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
TEST_PRIVATE_KEY_3=0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=28

```

Now, build and deploy all the needed contracts from wallet:

```bash

$ yarn build
$ yarn deploy

```

You will now see this, if everything worked correctly:

```bash

  System setup
l1MessengerAddress: 0x4ed7c70F96B99c776995fB64377f0d4aB3B0e1C1
0x9A676e781A523b5d0C0e43731313A708CB607508
🌕 L2LiquidityPool deployed to: 0x5FbDB2315678afecb367f032d93F642f64180aa3
🌕 L1LiquidityPool deployed to: 0x8f86403A4DE0BB5791fa46B8e795C547942fE4Cf
⭐️ L1 LP initialized: 0x0623281b4259fcdd7f048c38e996a5e03dd67274316cf20864dbd94f9acbdfd7
⭐️ L2 LP initialized: 0xf768af3519026fb7927fa3b667b152470891020fb112c4182e898128b6d48d0f
🌕 L1ERC20 deployed to: 0x5eb3Bc0a489C5A8288765d2336659EbCA68FCd00
🌕 L2DepositedERC20 deployed to: 0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
🌕 L1ERC20Gateway deployed to: 0x36C02dA8a0983159322a80FFE9F24b1acfF8B570
⭐️ L2DepositedERC20 initialized: 0x4a4efa9911a4c00ea19da79d31abeb898a0375e1a1cb6010fd993f56ba363863
🌕 L2TokenPool deployed to: 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
⭐️ L2TokenPool registered: 0x100763fa07a124c8386728f92fd1240f05463f0c379c593ae3f393b46bd2dc46
🌕 AtomicSwap deployed to: 0x0165878A594ca255338adfa4d48449f69242Eb8F
🌕 L1 Message deployed to: 0x809d550fca64d94Bd9F66E60752A544199cfAC3D
🌕 L2 Message deployed to: 0xa513E6E4b8f2a923D98304ec87F64353C4D5C853
⭐️ L1 Message initialized: 0x89b5ecb9eb2febe4f37a88fdca3a95fb6d5c8c3f82340adecca33c2e1a0d109a
⭐️ L2 Message initialized: 0x7da4146158d28bf5cb375c4713b573f365b7861a5671cd78a30b5548b28a94cd
    ✓ should deploy contracts (1572ms)


********************************
{
  "L1LiquidityPool": "0x8f86403A4DE0BB5791fa46B8e795C547942fE4Cf",
  "L2LiquidityPool": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
  "L1ERC20": "0x5eb3Bc0a489C5A8288765d2336659EbCA68FCd00",
  "L2DepositedERC20": "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
  "L1ERC20Gateway": "0x36C02dA8a0983159322a80FFE9F24b1acfF8B570",
  "l1ETHGatewayAddress": "0x4826533B4897376654Bb4d4AD88B7faFD0C98528",
  "l1MessengerAddress": "0x4ed7c70F96B99c776995fB64377f0d4aB3B0e1C1",
  "L2TokenPool": "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
  "AtomicSwap": "0x0165878A594ca255338adfa4d48449f69242Eb8F",
  "L1Message": "0x809d550fca64d94Bd9F66E60752A544199cfAC3D",
  "L2Message": "0xa513E6E4b8f2a923D98304ec87F64353C4D5C853"
}

🚨 Successfully wrote addresses to file

********************************
    ✓ should write addresses to file

  Messenge Relayer Test
l1MessengerAddress: 0x4ed7c70F96B99c776995fB64377f0d4aB3B0e1C1
0x9A676e781A523b5d0C0e43731313A708CB607508
accountNonceBob1: 50
accountNonceBob2: 9
 Preparing to wait for Message Hashes
 Looking for L2 to L1
 Got L2->L1 message hash 0x0f758119e28087753cd3d8c9b7728335c03e893fd46769c521479834763b7680
 Completed Withdrawal - L1 tx hash: 0x6feb8ebe87d3092034cf3064bf5c6f10e4dd6785863144d11b4e5309c09c2f95
    ✓ should send message from L2 to L1 (4166ms)
 Preparing to wait for Message Hashes
 Looking for L1 to L2
 Got L1->L2 message hash 0x891a86879a8783ef47f6c153edfdb6cf0fb4a494bdf00d589fc2c7f4c11c894f
 Completed Deposit - L2 tx hash: 0x8c2b45c0201220c39db250012d88d4fbe42501dfc2ed2f893de85088c3f7b394
    ✓ should send message from L1 to L2 (4709ms)

  Liquidity Pool Test
l1MessengerAddress: 0x4ed7c70F96B99c776995fB64377f0d4aB3B0e1C1
0x9A676e781A523b5d0C0e43731313A708CB607508
 Preparing to wait for Message Hashes
 Looking for L1 to L2
 Got L1->L2 message hash 0x2020085d58f20e0d51cca12cc8cc680eba0c11a97528b10eb4d82578ce693c55
 Completed Deposit - L2 tx hash: 0xf6c2ce01aeb5e650c7a1b07b1362c9ff88a5660fd4d279de82556b792b54b528
    ✓ should deposit ERC20 token to L2 (4180ms)
    ✓ should transfer ERC20 token to Alice and Kate (112ms)
    ✓ should add ERC20 token to token pool (79ms)
    ✓ should register L1 the pool (217ms)
    ✓ should register L2 the pool (90ms)
    ✓ shouldn't update the pool
    ✓ should add L1 liquidity (261ms)
    ✓ should add L2 liquidity (217ms)
 Preparing to wait for Message Hashes
 Looking for L2 to L1
 Got L2->L1 message hash 0xce81fe9cc3fcc2f22b90f5b75b9ac6528de9131bc6adfc0160c3b2d5cfcd5fe2
 Completed Withdrawal - L1 tx hash: 0x9548ae9cc2ef64280adbb90200b4976586106e322410769a25d78372b0eeda8e
    ✓ should fast exit L2 (4486ms)
    ✓ should withdraw liquidity (70ms)
    ✓ shouldn't withdraw liquidity
    ✓ should withdraw reward (70ms)
    ✓ shouldn't withdraw reward
 Preparing to wait for Message Hashes
 Looking for L1 to L2
 Got L1->L2 message hash 0xba0b4ea8f927dc3b83caa33c71e706969c4e021d03b50fa74e88430a9e9eae38
 Completed Deposit - L2 tx hash: 0x9617e9d31df25e41a3eb741147d8ae6745314fc41f212f24c7359cec882ac9a0
    ✓ should fast onramp (4931ms)
 Preparing to wait for Message Hashes
 Looking for L2 to L1
 Got L2->L1 message hash 0x4943bf329692343cd9a474f9e0a355d9c1a738f459af025011fde2d99341c4c6
 Completed Withdrawal - L1 tx hash: 0xbafbd4afbe2731d655030db6b3fe37fe1c98d0f778863c9d28af97a5cffd836b
    ✓ should revert unfulfillable swaps (4209ms)

  NFT Test

l1MessengerAddress: 0x4ed7c70F96B99c776995fB64377f0d4aB3B0e1C1
0x9A676e781A523b5d0C0e43731313A708CB607508
 🌕 NFT L2ERC721 deployed to: 0x09635F643e140090A9A8Dcd712eD6285858ceBef
 🔒 ERC721 owner: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
 ⚽️ meta: Henrietta Lacks#1623793970828#https://www.atcc.org/products/all/CCL-2.aspx
 📬 Successfully added NFT address to file
 ⚽️ balanceOwner: 0
 ⚽️ balanceRecipient: 1
 ⚽️ nftURL: Henrietta Lacks#1623793970828#https://www.atcc.org/products/all/CCL-2.aspx
 ⚽️ TID: 1
 ⚽️ TID: 2
    ✓ should mint a new ERC721 and transfer it from Bob to Alice (134ms)


  20 passing (32s)

✨  Done in 37.20s.

```

## 4. Firing up the wallet

The web wallet is a react front end that makes it easy to see your balances, transfer funds, and build on for your own uses. The code is deliberately basic, to make it easy for you to repurpose it for your own needs. It's a work in progress - for example, we are adding some basic support for NFTs and an interface for people to contribute to the conjoined liquidity pools that live on the L1 and L2.

Now navigate to child wallet folder

```bash
$ cd /optimism/packages/omgx/wallet/wallet
```

First create `.env` file and provide your Infura and Etherscan keys: along with below environment parameters

```bash
REACT_APP_INFURA_ID=
REACT_APP_ETHERSCAN_API=
REACT_APP_POLL_INTERVAL=20000
SKIP_PREFLIGHT_CHECK=true
```

Then,

```bash


$ yarn start

```

At that point, the wallet will start when you run `$ yarn start`. You can interact with the wallet at `http://localhost:3000.`

Install metamask by following the instruction on login page connect it with metamask so you can access the wallet.

# Common Wallet Setup Problems

**Nothing works** Rebuild the stack.

```bash

$ cd /ops
$ docker-compose up --build

```

**Wallet does not show balances** Did you set the correct ChainIDs in the custom RPC in MetaMask? Please make sure the ChainIDs are correct (Rinkeby = 4, OMGX L2 = 28, local hardhat L1 = 31337).

**I checked that and the wallet still does not show balances** Did you generate a `.env` and provide your `REACT_APP_INFURA_ID` and `REACT_APP_ETHERSCAN_API`?

### Integration Tests

Note that the integration tests also set up parts of the system that the web wallet will need to work, such as liquidity pools and bridge contracts.

```bash

$ yarn build  #builds the contracts
$ yarn deploy #if needed. this will test/deploy the contracts and write their addresses to /deployments/addresses.json
              #you do not need to deploy onto Rinkeby (unless you really want to) since all the needed contracts are already deployed and tested

```

The information generated during the deploy (e.g the `/deployment/local/addresses.json`) is used by the web wallet to set things up correctly. **The full test suite includes some very slow transactions such as withdrawals, which can take 100 seconds to complete. Please be patient.**

### 3. Wallet Specific Smart Contracts

These contracts instantiate a simple swap on/off system for fast entry/exit, as well as some basics such as support for atomic swaps.

### 3.1 L1liquidityPool.sol

The Layer 1 liquidity pool accepts ERC20 and ETH. `L1liquidityPool.sol` charges a convenience fee to the user for quickly getting on to the L2.

**L1->L2**: When users **deposit into this contract**, then (1) the pool size grows and (2) corresponding funds are sent to them on the L2 side.

**L2->L1**: When users **deposit into the corresponding L2 contract**, then (1) the pool size shrinks and (2) corresponding tokens are sent to them at their L1 wallet (minus the fee).

#### Known Gaps/Problems

- [ ] The contract owner can't currently get funds back out of the liquidy pool. This is a bug/feature, depending.
- [ ] Need system to allow _others_ to add liquidity and pay them for their liquidity.

#### Initial values

* _l2LiquidityPoolAddress_. The address of the Layer 2 liquidity pool
* _l1messenger_. The address of the Layer 1 messenger
* _l2ETHAddress_. The address of the oWETH contract on the L2
* _fee_. The convenience fee. The data type of `_fee` is `uint256`. If the fee is 3%, then `_fee` is 3.

#### Events

* `ownerAddERC20Liquidity_EVENT`. The contract owner just added funds to the liquidity pool.
* `clientDepositL1_EVENT`. A user just deposited tokens to the pool. `clientDepositL1` sends a message to L2, which triggers an L2 contract to send funds to the user's L2 wallet.
* `clientPayL1_EVENT`. A user deposited tokens into the system on the L2 side, which then called `clientPayL1` to send the tokens to the user's L1 account. `clientPayL1` can only be accessed by a contract on the L2 side.
* `ownerRecoverFee_EVENT`. The contract owner just withdrew fees.

#### Functions

* `init`. Can only be accessed by the contract owner. The owner can update the `_fee`.
* `receive`. This handles ETH. If called by the contract owner, it allows ETH to be deposited into the ETH pool. For other callers, it also sends a message to the `L2liquidityPool` smart contract on L2, which then sends oWETH to the sender.
* `ownerAddERC20Liquidity`. Used by the owner to provide liquidity into an ERC20 pool. Unlike a normal deposit, it doesn't send a message to L2.
* ` balanceOf` returns the balance of ERC20 or ETH of this contract. The default address of ETH is `address(0)`.
* ` feeBalanceOf` returns the fee balance of ERC20 or ETH of this contract.
* ` clientDepositL1` Users call this function to deposit tokens. After receiving the deposit, the contract sends a cross-domain message to L2. The `L2liquidityPool` sends the corresponding tokens to the user.
* ` clientPayL1` This cross-chain function can only be accessed `onlyFromCrossDomainAccount`. It can't be accessed by any users. When the layer 2 liquidity pool receives tokens and sends a message to L1, `clientPayL1` sends the token to the user and charges a convenience fee.
* ` ownerRecoverFee` can only be accessed by the contract owner. The contract owner can withdraw fees as they accumulate.

### 3.2 L2liquidityPool.sol

Just like the contract for L1, but with changes e.g. to deal with the fact that the L2 does not have native ETH.

### 3.3 Deploy Liquidity Pools

If you are working on a local testnet, please deploy **L2LiquidityPool.sol** first, then use the address of **L2LiquidityPool** as the parameter for deploying the **L1Liquidity**. See **/test/a_setup.spec.ts** to see the whole process.

```javascript

L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
  env.watcher.l2.messengerAddress,
)
await L2LiquidityPool.deployTransaction.wait()

L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
  L2LiquidityPool.address,
  env.watcher.l1.messengerAddress,
  env.L2ETHGateway.address,
  3 //this is the 3% fee
)
await L1LiquidityPool.deployTransaction.wait()

const L2LiquidityPoolTX = await L2LiquidityPool.init(L1LiquidityPool.address, "3")
await L2LiquidityPoolTX.wait()
console.log(' L2 LP initialized:',L2LiquidityPoolTX.hash);

```

### 3.4 AtomicSwap

Used to swap ERC20 tokens.

#### Functions

* `open` creates **Swap** struct, which contains the information of the buyer and sender.
* `close` closes the swap. It swaps the ERC20 tokens of the sender and the buyer.
* `expire` sets the status of the swap to be **EXPIRED**.
* `check` returns the **Swap** construct.

### 3.5 ERC721Mock aka NFTs

This contract sets up a very rudimentary interface to the @OpenZeppelin ERC721 contracts. Used to set up a very basic NFT system where the owner can mint NFTs and send them to others, who can then see their NFTs. Have a look at Alice's NFTs.

### MetaMask Settings

On the MetaMask side, some set up is needed.

1. Add your two test accounts to MetaMask (through **MetaMask>Import Account**). In the test code, PK_1 is the `Bob` account, and PK_2 is the `Alice`  account.

2. You also need to point Metamask at the correct chain.
  * For work on Rinkeby L1, chose **MetaMask>Networks>Rinkeby Test Network**.
  * For work on the OMGX Rinkeby L2, chose **MetaMask>Networks>Custom RPC** and enter `https://rinkeby.omgx.network/` with a ChainID of 28.
  * For work on a local L1, chose **MetaMask>Networks>Custom RPC** and enter `http://localhost:9545` with a ChainID of 31337.
  * For work on a local OMGX L2, chose **MetaMask>Networks>Custom RPC** and enter `http://localhost:8545` with a ChainID of 28.

*NOTE* You might have to reset MetaMask when you re-start the local network. The reset button is in **MetaMask>Settings>Advanced>Reset Account**.

### Wallet Use and Supported Functions

1. Open the MetaMask browser extension and select the chain you want to work on.

2. On the top right of the wallet landing page, select either `local` or `rinkeby`. You will then be taken to your account page. Here you can see your balances and move tokens from L1 to L2, and back, for example.
