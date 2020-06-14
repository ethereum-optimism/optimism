# API for OMG Network Vault Plugin

The API is authenticated, so every client must possess a valid Vault token that has permissions for the operation being requested. The plugin must be mounted at a particular path - the actual path is a design choice. For the sake of this documentation, the path will be `immutability-eth-plugin`. Mounting the plugin must be done by the administrator. Follows is an example of how a plugin is mounted assuming that the plugin is in the `$HOME/etc/vault.d/vault_plugins/` folder:

```sh
export SHA256=$(shasum -a 256 "$HOME/etc/vault.d/vault_plugins/immutability-eth-plugin" | cut -d' ' -f1)
vault write sys/plugins/catalog/secret/immutability-eth-plugin \
      sha_256="${SHA256}" \
      command="immutability-eth-plugin --ca-cert=$HOME/etc/vault.d/root.crt --client-cert=$HOME/etc/vault.d/vault.crt --client-key=$HOME/etc/vault.d/vault.key"
vault secrets enable -path=immutability-eth-plugin -plugin-name=immutability-eth-plugin plugin

```

In the examples, it is assumed that the Vault address is `http://127.0.0.1:8900`. This aligns with the Docker-based smoke testing scripts. Thus, for the subsequent API documentation, you will see the URL `http://127.0.0.1:8900/v1/immutability-eth-plugin/` used. This is what the URL is when the plugin is being tested via the Docker-based smoke testing scripts. It should be clear that this URL will vary based on how Vault is configured in your environment.

## CONFIGURATION

Before a plugin can be used, it must be configured. This is where we establish the RPC endpoint for the Ethereum node. This can be an Infura endpoint. Also, the chain ID must be supplied.

### CONFIGURE MOUNT

A PUT to the `/config` path is used configure the mount.

#### INPUTS

* chain_id:

```
Ethereum network - can be one of the following values:

1 - Ethereum mainnet
2 - Morden (disused), Expanse mainnet
3 - Ropsten
4 - Rinkeby (Default)
30 - Rootstock mainnet
31 - Rootstock testnet
42 - Kovan
61 - Ethereum Classic mainnet
62 - Ethereum Classic testnet
1337 - Geth private chains
```

* rpc_url:

```
The RPC address of the Ethereum network. Default = https://rinkeby.infura.io
```

* whitelist:

```
The list of accounts that any account can send ETH to.
```

* blacklist:

```
The list of accounts that any account can't send ETH to.
```

* bound_cidr_list:

```
Comma separated string or list of CIDR blocks. If set, specifies the blocks of
IPs which are allowed to use the plugin.
```

#### EXAMPLE

The following `curl` command will configure the mount to use Ganache as the Ethereum node.

```sh

curl -X PUT -H "X-Vault-Token: $(vault print token)" -H "X-Vault-Request: true" -d '{"chain_id":"5777","rpc_url":"http://ganache:8545"}' http://127.0.0.1:8900/v1/immutability-eth-plugin/config

{
  "request_id": "0f6dcfcd-905b-cb27-9ff2-242722395645",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "blacklist": null,
    "bound_cidr_list": null,
    "chain_id": "5777",
    "rpc_url": "http://ganache:8545",
    "whitelist": null
  },
  "warnings": null
}
```

### READ MOUNT CONFIGURATION

The following `curl` command will read the mount configuration.

```sh
curl -H "X-Vault-Token: $(vault print token)" -H "X-Vault-Request: true" http://127.0.0.1:8900/v1/immutability-eth-plugin/config

{
  "request_id": "b35dd8f3-1d3b-11a9-a723-dc892720efb7",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "blacklist": null,
    "bound_cidr_list": null,
    "chain_id": "5777",
    "rpc_url": "http://ganache:8545",
    "whitelist": null
  },
  "warnings": null
}

```

## WALLETS

A wallet contains the BIP44 key used to derive accounts.

### CREATE WALLET

A PUT to the `/wallets/<NAME>` enpoint creates (or updates) an Ethereum wallet: an wallet controlled by a private key. Also The generator produces a high-entropy passphrase.

#### INPUTS

*name:

```
Name of the wallet - provided in the URI.
```

*mnemonic:

```
The mnemonic to use to create the account. If not provided, one is generated.
```

*whitelist:

```
The list of the only Ethereum accounts that accounts in this wallet can send transactions to.
```

*blacklist:

```
The list of Ethereum accounts that accounts in this wallet can't send transactions to.
```

### EXAMPLE

The following `curl` command will create a wallet named `test-wallet-1`.

```sh
curl -X PUT -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" -d 'null' http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/test-wallet-1

{
  "request_id": "db7d0a6a-2d5b-7b62-2d6c-51efe66d3519",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "blacklist": null,
    "index": 0,
    "whitelist": null
  },
  "warnings": null
}
```

## ACCOUNTS

An Ethereum account with a BIP44 derived key.

### CREATE ACCOUNT

A PUT to the `/wallets/<NAME>/accounts` endpoint creates an Ethereum account.

#### INPUTS

*name:

```
Name of the wallet - provided in the URI.
```

*whitelist:

```
The list of the only Ethereum accounts that this account can send transactions to.
```

*blacklist:

```
The list of Ethereum accounts that this account can't send transactions to.

### EXAMPLE

```sh
curl -X PUT -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" -d 'null' http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/test-wallet-1/accounts

{
  "request_id": "4141b5c3-3ba7-beeb-64fe-cac2d75a1dc0",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "address": "0xA1296d36980058b1fe2Bb177b733FaC763d8405E",
    "blacklist": null,
    "index": 0,
    "whitelist": null
  },
  "warnings": null
}
```

### CHECK ACCOUNT BALANCE

A GET to the `/wallets/<NAME>/accounts/<ADDRESS>/balance` endpoint returns the balance for an Ethereum account.

