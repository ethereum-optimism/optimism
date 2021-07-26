## Front End Development

- [Front End Development](#front-end-development)
  * [1. Starting a local L1/L2](#1-starting-a-local-l1-l2)
  * [2. Initial state of preconfigured accounts](#2-initial-state-of-preconfigured-accounts)
  * [3. Advanced configuration of test accounts](#3-advanced-configuration-of-test-accounts)
  * [4. Test accounts for the swap on/off and the liquidity pools](#4-test-accounts-for-the-swap-on-off-and-the-liquidity-pools)
  * [5. Starting the react app](#5-starting-the-react-app)
  
### 1. Starting a local L1/L2

Run:

```bash
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```

You can change the BUILD and DAEMON values to control if everything is rebuilt (`BUILD=1`, very slow) and if you want to see all the debug information (`DAEMON=0`). Typically, you will only have to build everything once, and after that, you can save time by setting `BUILD` to `2`. In that case, we'll use the docker images you built earlier.

```bash
$ cd ops
$ BUILD=2 DAEMON=0 ./up_local.sh
```

### 2. Initial state of preconfigured accounts 

To facilitate development and testing, there are three accounts that will receive defined tokens and NFTs:

* Deployer (aka Bob) - This one starts out with 5000 ETH on L1 and 4999.9036 oETH on L2. During deployment, a test token is created called TEST, and the deployer (aka Bob) holds the entire initial supply (10000000000) on the L1. This account also has the *owner permissions* for the test NFT. 

_Account Address: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266  
Private Key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80_    

* Alice (PK_2)

This one starts out with 4999.9967 ETH on L1 and 5000 oETH on L2. NFTs: zero.

_Account Address: 0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc  
Private Key: 0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba_  

* Kate (PK_3) 

This one starts out with 4999.9967 ETH on L1 and 5000 oETH on L2. NFTs: zero.

_Account Address: 0x976EA74026E726554dB657fA54763abd0C3a0aa9  
Private Key: 0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e_  

### 3. Advanced configuration of test accounts

You can generate some test NFTs and dispense some TEST tokens by running:

```bash
$ cd /packages/omgx/contracts
$ yarn test:integration
```

After running this command, the account balances will change (changes in bold-face):

Deployer/Bob (PK_1):	**4999.7403** L1 ETH,	**4999.9036** L2 oETH,	**9999987655** L1 TEST,	**10235** L2 TEST  
Alice (PK_2):	4999.9967 L1 ETH,	5000.0000 L2 oETH,	**1111** TEST on L2,	**two** NFTs  
Kate: (PK_3):	4999.9967 L1 ETH,	5000.0000 L2 oETH,	**999** TEST on L2,	zero NFTs  

**Note - To add these accounts to your MetaMask wallet, go to MetaMask, click on the round icon on the right top, and select "import acconut", and paste in private key #1. Then do this twice more, leaving you with three test accounts with different holdings.**

### 4. Test accounts for the swap on/off and the liquidity pools

To work on the swap on/off and the liquidity pool UI, you should also pre-configure the pools. To do this, run:

```bash
$ cd /packages/omgx/message-relayer-fast
$ yarn test:integration
```

### 5. Starting the react app

Open a second terminal window, navigate to `packages/omgx/wallet-frontend`, and run

```bash
$ yarn get_artifacts #this will get all the contract artifacts - 
                     #note that this will only work correctly if you 
                     #ran `yarn build` at the top level per instructions
$ yarn start
```

and the frontend should start up in a local browser. You can also develop on the Rinkeby testnet - in that case, you do not need to run a local L1/L2. If you would like to do that, just change the .env settings:

```bash
# This is for working on the wallet, pointed at the OMGX Rinkeby testnet
REACT_APP_INFURA_ID=
REACT_APP_ETHERSCAN_API=
REACT_APP_POLL_INTERVAL=20000
SKIP_PREFLIGHT_CHECK=true
REACT_APP_WALLET_VERSION=1.0.10
```