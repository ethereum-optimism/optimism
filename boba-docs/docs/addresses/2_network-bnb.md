# Boba BNB Network Contract Addresses

<!-- * \[Boba BNB Testnet L2 (9728) for the BNB Testnet (97)]
  * [Testnet Bridging](network-bnb.md#testnet-bridging)
  * [Analytics and eth\_getLogs for Boba BNB Testnet](network-bnb.md#analytics-and-eth\_getlogs-for-boba-bnb-testnet)
  * [Boba BNB Testnet Addresses](network-bnb.md#boba-bnb-testnet-addresses)
  * [Boba BNB Testnet Links and Endpoints](network-bnb.md#boba-bnb-testnet-links-and-endpoints)
* \[Boba BNB L2 (56288) for the BNB L1 (56)]
  * [Boba BNB Addresses](network-bnb.md#boba-bnb-addresses)
  * [Analytics and eth\_getLogs for Boba BNB](network-bnb.md#analytics-and-eth\_getlogs-for-boba-bnb)
  * [Boba BNB Links and Endpoints](network-bnb.md#boba-bnb-links-and-endpoints) -->

## Boba BNB Testnet (9728) for the BNB Testnet (97)

### Testnet Bridging

The standard bridges for `BNB` and `BOBA` are active, so you can can both bridge and exit `BNB` and `BOBA` from BNB testnet to Boba BNB testnet and back. The exit delay (the fraud proof window) has been set to 5 minutes (it's normally 7 days) to make development easier.

### Analytics and eth\_getLogs for Boba BNB Testnet

If you have unusual `getLogs` needs, especially calls from `0 to latest`, the main RPC will block you, since this is how most DoS attacks work. In those cases, we encourage you to run your own RPC endpoint on your own replica of Boba BNB testnet. We have prepared Docker images for you, so this should only take a few minutes. To access these images:

* clone the `boba` repo
* switch to `develop` branch.
*   Add `.env` in [boba-node](https://github.com/bobanetwork/boba\_legacy/tree/develop/boba\_community/boba-node) folder

    ```
    RELEASE_VERSION=v0.X.X
    ```

The docker-compose file is in [`boba-community/boba-node/docker-compose-bobabnb-testnet.yml`](https://github.com/bobanetwork/boba\_legacy/tree/develop/boba\_community/boba-node).

```bash
docker compose -f docker-compose-bobabnb-testnet.yml pull
docker compose -f docker-compose-bobabnb-testnet.yml up
```

The DTL will first sync with the chain. During the sync, you will see the DTL and Replica gradually catch up with the Boba L2. This can take several minutes to several hours, depending on which chain you are replicating.

### Boba BNB Testnet Addresses

For **primary contracts and addresses** see [packages/contracts/deployments/bobabnbtestnet/README.md](https://github.com/bobanetwork/boba\_legacy/tree/develop/packages/contracts/deployments/bobabnbtestnet/)

For **secondary addresses**, such as L2 Tokens and Messengers, please see the [Boba BNB testnet address registration dump](https://github.com/bobanetwork/boba\_legaccy/tree/develop/packages/boba/register/addresses/addressBobaBnbTestnet\_0xAee1fb3f4353a9060aEC3943fE932b6Efe35CdAa.json).

For **account abstraction** contract addresses, please refer the list [here](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/account-abstraction/deployments/boba\_bnb\_testnet/addresses.json).

### Boba BNB Testnet Links and Endpoints

|               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ChainID       | 9728                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| RPC           | [https://testnet.bnb.boba.network](https://testnet.bnb.boba.network) <br /><br />This endpoint is not for production systems and is rate limited.<br /><br />We have partnered with Tenderly, which provides subscription endpoints and public endpoints (public endpoints are IP rate limited; subscribe to [Tenderly](https://tenderly.co) if you need more):<br /><br />["https://boba-bnb-testnet.gateway.tenderly.co/](https://boba-bnb-testnet.gateway.tenderly.co) |
| Replica RPC   | [https://replica.testnet.bnb.boba.network](https://replica.testnet.bnb.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Gateway       | [https://gateway.boba.network](https://gateway.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| Blockexplorer | [https://blockexplorer.testnet.bnb.boba.network](https://blockexplorer.testnet.bnb.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Websocket     | [wss://wss.testnet.bnb.boba.network](wss://wss.testnet.bnb.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| AA bundler    | [https://bundler.testnet.bnb.boba.network](https://bundler.testnet.bnb.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Gas Cap       | 50000000                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |

## Boba BNB L2 (56288) for the BNB L1 (56)

### Analytics and eth\_getLogs for Boba BNB

To access these images:

* clone the `boba` repo
* switch to `develop` branch.
*   Add `.env` in [boba-node](https://github.com/bobanetwork/boba\_legacy/tree/develop/boba\_community/boba-node) folder

    ```
    RELEASE_VERSION=v0.X.X
    ```

The Boba BNB's docker-compose file is in [`boba-community/boba-node/docker-compose-bobabnb.yml`](https://github.com/bobanetwork/boba\_legacy/tree/develop/boba\_community/boba-node)

```bash
docker compose -f docker-compose-bobabnb.yml pull
docker compose -f docker-compose-bobabnb.yml up
```

### Boba BNB Addresses

For **primary contracts and addresses** see [packages/contracts/deployments/bobabnb/README.md](https://github.com/bobanetwork/boba\_legacy/tree/develop/packages/contracts/deployments/bobabnb/)

For **secondary addresses**, such as L2 Tokens and Messengers, please see the [Boba BNB address registration dump](https://github.com/bobanetwork/boba\_legacy/tree/develop/packages/boba/register/addresses/addressBobaBnb\_0xeb989B25597259cfa51Bd396cE1d4B085EC4c753.json).

### Boba BNB Links and Endpoints

|               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ChainID       | 56288                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| RPC           | [https://bnb.boba.network](https://bnb.boba.network)<br /><br />This endpoint is not for production systems and is rate limited.<br /><br />We have partnered with Tenderly, which that provides subscription endpoints and public endpoints (public endpoints are IP rate limited, subscribe to [Tenderly](https://tenderly.co) if you need more):<br /><br />[http://boba-bnb.gateway.tenderly.co](https://boba-bnb.gateway.tenderly.co)<br /><br />[http://gateway.tenderly.co/public/boba-bnb](https://gateway.tenderly.co/public/boba-bnb) |
| Gateway       | [https://gateway.boba.network](https://gateway.boba.network)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Blockexplorer | [https://bnb.bobascan.com](https://bnb.bobascan.com)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| Websocket     | [wss://boba-bnb.gateway.tenderly.co](wss://boba-bnb.gateway.tenderly.co)<br />[wss://gateway.tenderly.co/public/boba-bnb](wss://gateway.tenderly.co/public/boba-bnb)                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Gas Cap       | 50000000                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
