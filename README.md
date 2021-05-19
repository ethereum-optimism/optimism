# OMGX_Alt_Messenger

Customized L1 Cross Domain Messenger without dispute period time restrictions and associated message relayer service.

The custom `OVM_L1CrossDomainMessenger` works with the default `OVM_L2CrossDomainMessenger`. The messages from the L2_Messenger can be relayed by the custom messenger instead to skip the dispute period and to do that, the bridge/token contract should specify the custom messenger to be the messenger for relays. The custom messenger cannot be used to send cross domain messages. For sending messages the bridge contracts use the default L1_Messenger.

## Using the custom messenger

- Deploy `contracts/OVM_L1_CrossDomainMessenger.sol` on L1, this will be the contract used by your contracts for L2->L1 message passing

- Your bridge/gateway contract must implement `contracts/libraries/OVM_CrossDomainEnabled.sol` instead. This uses the default L1 Messenger to send messages and the custom L1 Messenger to relay.

## Running the Alt_Relayer

Env Settings -

```
ADDRESS_MANAGER_ADDRESS= <address manager contract address>
L1_NODE_WEB3_URL= <l1 node url>
L2_NODE_WEB3_URL= <l2 node url>
L1_WALLET_KEY= <l1 wallet key>
L1_MESSENGER_ADDRESS= <l1 custom messenger address>
L1_TARGET= <target contract to allow relays to>
```

Running the message relayer
```
cd relayer_service
yarn install
yarn build
yarn start
```