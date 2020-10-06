# immutability-eth-plugin docker

Previously, when you executed `docker-compose -f docker/docker-compose.yml up --build` it would:

* Spin up a ganache container
* Deploy the plasma contracts
* Spin up an `ephemeral` vault instance
* Execute some smoke tests

Now, the behavior is different.

The first time you bring up the `docker-compose`, it will:

* Spin up a ganache container
* Deploy the plasma contracts
* Spin up a `persistent` vault instance
* Initialize the vault instance - with one key shard
* Unseal the vault instance
* Execute some smoke tests

From then on, however, all subsequent runs of `docker-compose` will:

* Spin up a ganache container
* Deploy the plasma contracts
* Spin up a `persistent` vault instance
* Unseal the vault instance

If you interact with vault from the outside - say change the rpc_url, create new accounts, etc - all that will be persisted. so, each subsequent time you run docker-compose, your previous interactions will be retained.

### How to interact with Vault from the outside

If this project is in `$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin`, then the persisted vault data is located in 3 places:

1. TLS Certificates and Key: `$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/ca`

You will need to trust the signer to use the vault CLI:

```bash
export VAULT_CACERT="$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/ca.crt"
```

2. Unseal Key and Root Token: `$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/unseal.json`

You will need to use the root token (or another token derived from it) to test:

```bash
export VAULT_TOKEN=$(cat $GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/unseal.json | jq -r .root_token)
```

3. The Vault data store: `$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/data`

Don't mess with this.

If you want to re-initialize the Vault, then delete these:

```bash
rm $GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/config/docker/unseal.json
rm -fr $GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/data
```

I would strongly advise using the Vault CLI. This way you can use vault with the same semantics that you see in the smoke tests:

```bash
export VAULT_ADDR="https://127.0.0.1:8200"
export VAULT_CACERT="$GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/ca.crt"
export VAULT_TOKEN=$(cat $GOPATH/src/github.com/omgnetwork/immutability-eth-plugin/docker/config/unseal.json | jq -r .root_token)

vault read -format=json immutability-eth-plugin/config

``` 

Unless of course, you are using elixir... in which case, you are on your own. 