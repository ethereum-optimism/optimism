# op-faucet

This implements a dev-faucet, with support for multiple chains and faucet configurations.

This faucet focuses on serving acceptance tests:
it can be instantiated in-process in Go, or run in Kurtosis or other deployments,
for acceptance tests to point against for test-user funding.

## Usage

### Build from source

```bash
# from op-faucet dir:
just op-faucet
./bin/op-faucet --help
```

### Run from source

Example config:
```yaml
faucets:
  sepolia:
    el_rpc: https://sepolia.drpc.org
    chain_id: 11155111
    tx_cfg:
      # your funder testnet key (example: first account in test mnemonic)
      private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
```

Run the service:
```bash
go run ./cmd --config=config.yaml --rpc.addr=127.0.0.1 --rpc.port=9000
```

Request funds:
```bash
# sends 1 eth (1e18 wei) to test-account 0x70... (second account in test mnemonic)
cast rpc --rpc-url=http://localhost:9000/chain/11155111 \
  faucet_requestETH 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 1000000000000000000
```

### Build docker image

Not available yet.

## Overview

Faucets are configured in a `config.yaml` file.

Each faucet, by `<name>`, is available as Faucet RPC under the `/faucet/<name>` HTTP route.
Websocket RPC is also supported (dial the given HTTP route, websocket is upgraded from HTTP).

For each known chain, an alias is set up to serve the default faucet for each chain: `/chain/<chainID>`.

Multiple faucets may be configured, for different chains and use-cases.

Different chains are supported to make interop-devnet faucets easy:
only one op-faucet service is needed in a dev environment.

The configuration may also be used to configure different authenticated faucets.

### `faucet` RPC namespace

The faucet uses the create-self-destruct pattern to fund without EVM execution of the target account.

#### `faucet_requestETH`

Funds the requested `addr` with the requested `amount`.

Params:
- `addr`: hex-encoded, 0x-prefixed ethereum address
- `amount`: decimal number, in wei, to fund.

Returns:
- error if the transaction fails to send and confirm.

### `admin` RPC namespace

On the global RPC an `admin` namespace is available,
to change faucets (e.g. enable/disable) at runtime.
These faucet changes do not persist.

