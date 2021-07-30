# OMGX_Messenger_Relayer_Fast

Customized L1 Cross Domain Messenger without dispute period time restrictions and associated message relayer service.

The custom `OVM_L1CrossDomainMessenger` works with the default `OVM_L2CrossDomainMessenger`. The messages from the L2_Messenger can be relayed by the custom messenger instead to skip the dispute period and to do that, the bridge/token contract should specify the custom messenger to be the messenger for relays. The custom messenger cannot be used to send cross domain messages. For sending messages the bridge contracts use the default L1_Messenger.

## Using the custom messenger

- Deploy `contracts/OVM_L1_CrossDomainMessenger.sol` on L1, this will be the contract used by your contracts for L2->L1 message passing

- Your bridge/gateway contract must implement `contracts/libraries/OVM_CrossDomainEnabled.sol` instead. This uses the default L1 Messenger to send messages and the custom L1 Messenger to relay.

## Running the Custom Messenger + Relayer

- To deploy the custom messenger and start up the associated relayer run-
```
yarn start
```

- To run tests
```
yarn test:integration
```

## env settings

Use .env.example for quick tests

```

ADDRESS_MANAGER_ADDRESS= <address manager contract address>
L1_NODE_WEB3_URL= <l1 node url>
L2_NODE_WEB3_URL= <l2 node url>
L1_MESSENGER_ADDRESS= <l1 custom messenger address>
FAST_RELAYER_PRIVATE_KEY= <private_key account for relayer>
L1_TARGET= <target contract to allow relays to, set to 0x0 to skip>

```

for tests
```

TEST_PRIVATE_KEY_1=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
TEST_PRIVATE_KEY_2=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
TEST_PRIVATE_KEY_3=0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a

```

optional - while running message relayer for specific custom messenger
```
L1_MESSENGER_FAST= <optional>
```

### Additional Setting for stopping failed relay attempts

- Add the target contract to the default message relayer's blacklist
- Set the L1_TARGET=<target_contract> for custom message relayer

### Deploying only Custom messenger

Deploy with the deployer private key to also register the custom messenger
```
yarn deploy:contracts
```

#### Deploying only Custom Relayer

```
yarn start:service
```

This starts the service for the registered custom messenger. Specify L1_MESSENGER_FAST=<messenger> to spin up the relayer for your messenger

## Build a DockerHub Message Relayer Fast

To build the Message Relayer Fast docker image:

```bash

docker build . --file Dockerfile.message-relayer-fast --tag omgx/message-relayer-fast:latest
docker push omgx/message-relayer-fast:latest

```
