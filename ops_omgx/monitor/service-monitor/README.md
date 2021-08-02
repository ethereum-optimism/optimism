## Description
Script'll subscribe l1 and l2. Every new block, callback'll get balance of l1 pool and l2 pool, then log to console. Datadog agent can collect logs from docker host.

## How to setup monitoring
1. Create .env follow example:
    ```
    NODE_ENV=rinkeby
    L1_NODE_WEB3_WS=ws://localhost:9545
    L2_NODE_WEB3_WS=ws://localhost:8546
    L1_LIQUIDITY_POOL_ADDRESS=0x1383fF5A0Ef67f4BE949408838478917d87FeAc7
    L2_LIQUIDITY_POOL_ADDRESS=0x88b3743A9e1FdB3C8C92Cec7A6A370c1403c7C60
    RELAYER_ADDRESS=0x3C8b7FdbF1e5B2519B00A8c9317C4BA51d6a4f9d
    SEQUENCER_ADDRESS=0xE50faB5E5F46BB3E3e412d6DFbA73491a2D97695
    RECONNECT_TIME=10000 // delay time to reconect provider
    ```
2. run `npm start`

## How to setup dummy-transaction
1. Create .env follow example:
    ```
    NODE_ENV=rinkeby
    L1_NODE_WEB3_URL=https://rinkeby.infura.io/v3/d64a23da1a714a0f9f8bf6c9352235a8
    L2_NODE_WEB3_URL=http://ec2-54-226-193-17.compute-1.amazonaws.com:8545
    L1_LIQUIDITY_POOL_ADDRESS=0x473d2bbF979D0BFA39EBAB320c3216408386e68d
    L2_LIQUIDITY_POOL_ADDRESS=0x1eCD5FBbb64F375A74670A1233CfA74D695fD861
    L1_GAS_USED=229932
    L1_ADDRESS_MANAGER=0x93A96D6A5beb1F661cf052722A1424CDDA3e9418
    L2_DEPOSITED_ERC20=0x0e52DEfc53ec6dCc52d630af949a9b6313455aDF
    DUMMY_DELAY_MINS=5 // delay after every transaction in minutes
    DUMMY_ETH_AMOUNT=0.0005 // transaction amount in eth
    DUMMY_TIMEOUT_MINS=1 // time out when transaction error in minutes
    ```
2. run `npm run dummy-transaction`

## How to deploy image
1. Build new image: `docker build -t enyalabs/omgx-monitor:{version-number} .` version-number:
    - format `v1.1.2` is for monitoring service
    - format `dummy-v1.1.2` is for dummy-transaction service
2. Push new image to docker hub: `docker push enyalabs/omgx-monitor:{version-number}`
