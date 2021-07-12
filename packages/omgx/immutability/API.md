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

| Parameter | Description |
| --- | ----------- |
| chain_id | Ethereum network. Default = 4 (Rinkeby) |
| rpc_url | The RPC address of the Ethereum network. Default = `https://rinkeby.infura.io` |
| whitelist | The list of accounts that any account can send ETH to. |
| blacklist | The list of accounts that any account can't send ETH to. |
| bound_cidr_list | Comma separated string or list of CIDR blocks. If set, specifies the blocks of IPs which are allowed to use the plugin. |

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
    "chain_l2_id": "28",
    "rpc_l2_url": "http://l2geth:9545",
    "whitelist": null
  },
  "warnings": null
}

```

## WALLETS

A wallet contains the BIP44 key used to derive accounts.

### CREATE WALLET

A PUT to the `/wallets/<NAME>` enpoint creates (or updates) an Ethereum wallet: an wallet controlled by a private key. Also The generator produces a high-entropy passphrase.

### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet - provided in the URI. |
| mnemonic | The mnemonic to use to create the account. If not provided, one is generated. |
| whitelist | The list of the only Ethereum accounts that accounts in this wallet can send transactions to. |
| blacklist | The list of Ethereum accounts that accounts in this wallet can't send transactions to. |

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

### DELETE WALLET

A DELETE to the `/wallets/<NAME>` endpoint deletes a wallet.

#### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet to be deleted - provided in the URI. |

### EXAMPLE

```sh
curl -X DELETE -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/temp-wallet
```

## ACCOUNTS

An Ethereum account with a BIP44 derived key.

### CREATE ACCOUNT

A PUT to the `/wallets/<NAME>/accounts` endpoint creates an Ethereum account.

#### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet - provided in the URI. |
| whitelist | The list of the only Ethereum accounts that this account can send transactions to. |
| blacklist |  The list of Ethereum accounts that this account can't send transactions to. |

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

### DEBIT ACCOUNT

A PUT to the `/wallets/<NAME>/accounts/<ACCOUNT>/debit` endpoint debits an Ethereum account.

#### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet - provided in the URI. |
| address | Account address **from** which the funds will be transferred - provided in the URI. |
| to |  Account address **to** which the funds will be transferred. |
| amount |  Amount of ETH (in wei). |
| gas_limit | The gas limit for the transaction - defaults to 21000. |
| gas_price | The gas price for the transaction in wei - will be estimated if not supplied. |

### EXAMPLE

```sh
curl -X PUT -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" -d '{"amount":"100000000000000000","to":"0x9A1Db13bAb531Ea41C30b97E160f7aDe9efb02c8"}' http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/test-wallet-2/accounts/0x71f8f93D25C5A56e2B6810cB84D23E8a2e760D68/debit

{
  "request_id": "8565f6cf-6082-fe02-2793-2aca669fd6c8",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "amount": "100000000000000000",
    "from": "0x71f8f93D25C5A56e2B6810cB84D23E8a2e760D68",
    "gas_limit": "21000",
    "gas_price": "20000000000",
    "nonce": "41",
    "signed_transaction": "0xf86c298504a817c800825208949a1db13bab531ea41c30b97e160f7ade9efb02c888016345785d8a0000801ba0b351bbe2ae6e7f8401fa8ca35f6bf3eba5aa5db527384d5a0b1aa1f24b67c911a028665a1550e6fb706933cc03d173d7321084a050cd054eeebc040723f44da20b",
    "to": "0x9A1Db13bAb531Ea41C30b97E160f7aDe9efb02c8",
    "transaction_hash": "0xca12f0e2cc9ffcd1c42a472eb5baeeabe0c98bd87ae3cae0726a6cedcb68b047"
  },
  "warnings": null
}
```

### DELETE ACCOUNT

A DELETE to the `/wallets/<NAME>/accounts/<ACCOUNT>` endpoint deletes an Ethereum account.

#### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet - provided in the URI. |
| address | Account address which will be deleted - provided in the URI. |

### EXAMPLE

```sh
curl -X DELETE -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/test-wallet-2/accounts/0x4BC91c7fA64017a94007B7452B75888cD82185F7
```

## PLASMA

OMG Networks plasma contract.

### SUBMIT BLOCK

Submits the Merkle root of a Plasma block

#### INPUTS

| Parameter | Description |
| --- | ----------- |
| name | Name of the wallet - provided in the URI. |
| address | Account address which will submit the block - provided in the URI. |
| contract | The address of the Block Controller contract. |
| gas_price | The gas price for the transaction in wei. |
| block_root | The Merkle root of a Plasma block. |
| nonce | Transaction order. |

### EXAMPLE

```sh
curl -X PUT -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" -d '{"block_root":"KW7c+YhqaeXzUSARcnOh0sBSWhAU7l144fF6ls0Y5Vw=","contract":"0xd185aff7fb18d2045ba766287ca64992fdd79b1e", "gas_price: "20000000000", nonce: "0""}' http://127.0.0.1:8900/v1/immutability-eth-plugin/wallets/plasma-deployer/accounts/0x888a65279D4a3A4E3cbA57D5B3Bd3eB0726655a6/plasma/submitBlock

{
  "request_id": "00a614f3-9bd3-60f4-25be-384a8d3cc5ff",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": {
    "contract": "0xd185AFF7fB18d2045Ba766287cA64992fDd79B1e",
    "from": "0x4BC91c7fA64017a94007B7452B75888cD82185F7",
    "gas_limit": 73623,
    "gas_price": 20000000000,
    "nonce": 0,
    "signed_transaction": "0xf889018504a817c80083011f9794d185aff7fb18d2045ba766287ca64992fdd79b1e80a4baa4769431323334717765726164676631323334717765726164676600000000000000001ca04b14e95372a41a74585c04c7967c45f2d1d51e4f5cd59b7c95a2c16ecbd63e79a04fcc461cfd165d8ba1f9cafe37ce7c025c0cec0533880abda3df754c9c749d9a",
    "transaction_hash": "0x6cfad4034bf147accb815922bb4f71ed8ae676e65580ab259d9d1d8713047c7f"
  },
  "warnings": null
}
```