# OMGX Smart Contracts


## 1. Automatic contract deployment and serving

This spins up the entire stack, with all contracts deployed, and all the right things needed for the wallet to function, and for development work on the wallet.

```bash

$ cd ops 
$ ./up_local.sh

```

**Note - please provide syntax for setting the .env variables (BUILD: 1, DAEMON: 1)**

To get the contract addresses for the basic stack, and the OMGX-specific contracts:

```bash

curl http://127.0.0.1:8078/addresses.json | jq #basic stack
curl http://127.0.0.1:8080/addresses.json | jq #OMGX-specific contracts

```

## 2. Manual Deployment and Testing

Spin up the base local L1/L2:

```

$ cd ops
$ docker-compose up -V

```

Create a `.env` file in the root directory of the contracts folder. Add environment-specific variables on new lines in the form of `NAME=VALUE`. Examples are given in the `.env.example` file. Just pick which net you want to work on and copy either the "Rinkeby" _or_ the "Local" envs to your `.env`.

```bash

# Local
NODE_ENV=local
L1_NODE_WEB3_URL=http://localhost:9545
L2_NODE_WEB3_URL=http://localhost:8545
ETH1_ADDRESS_RESOLVER_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
TEST_PRIVATE_KEY_1=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
TEST_PRIVATE_KEY_2=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
TEST_PRIVATE_KEY_3=0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a
TARGET_GAS_LIMIT=9000000000
CHAIN_ID=28
TEST=1 #This deploys the ERC20 test token

```

Build and deploy all the needed contracts:

```bash

$ yarn build
$ yarn deploy

```

You will now see this, if everything worked correctly:

```bash

 % yarn deploy
yarn run v1.22.10
$ ts-node "./bin/deploy.ts"
Starting OMGX core contracts deployment...
ADDRESS_MANAGER_ADDRESS was set to 0x5FbDB2315678afecb367f032d93F642f64180aa3
Nothing to compile
Deploying...
🌕 L2LiquidityPool deployed to: 0x7A9Ec1d04904907De0ED7b6839CcdD59c3716AC9
🌕 L1LiquidityPool deployed to: 0xe8D2A1E88c91DCd5433208d4152Cc4F399a7e91d
⭐️ L1 LP initialized: 0x511a3000131b6d3ac16a22d12707dc4121a62c198679300a081cfa9586b32d89
⭐️ L2 LP initialized: 0x0797a3c93960e62a84c59f7f49c91916e430488b08afd38519ef9ac057eabec6
L1 and L2 pools have registered ETH and OETH
🌕 L1ERC20 deployed to: 0x4b6aB5F819A515382B0dEB6935D793817bB4af28
🌕 L2ERC20 deployed to: 0x86A2EE8FAf9A840F7a2c64CA3d51209F9A02081D
🌕 L2TokenPool deployed to: 0xA4899D35897033b927acFCf422bc745916139776
⭐️ L2TokenPool registered: 0x43f4d7adec84555ef5548adf41a75c8bdf45798a993cf4d5e42e2b31ab140d01
🌕 L1_CrossDomainMessenger_Fast deployed to: 0xCace1b78160AE76398F486c8a18044da0d66d86D
⭐️ Fast L1 Messager initialized: 0x6480f00ca7f65d207bbbf4831074a71fb7ca4b6d999aeb78fbb5fb3841938362
⭐️ Fast L1 Messager initialized: 0xc44a3f18a3e6c10ceb6da3cebd4b23ee491ef6947ab3bf05574260e3c7f8c206
🌕 AtomicSwap deployed to: 0xAA292E8611aDF267e563f334Ee42320aC96D0463
🌕 L1 Message deployed to: 0xc0F115A19107322cFBf1cDBC7ea011C19EbDB4F8
🌕 L2 Message deployed to: 0x5c74c94173F05dA1720953407cbb920F3DF9f887
⭐️ L1 Message initialized: 0x37fbbe3ef0ed3f1f4ae6c94fcd1f1825ea6c425091b038c4d4541c8760ea2c53
⭐️ L2 Message initialized: 0xa736724e36f8098f70f737ac0c643490732a9ff350bd0fc9629a6face73178a8
✨  Done in 10.84s.

```
