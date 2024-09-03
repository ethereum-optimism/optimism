# `fpac-deploy`

Chain-ops scripts for the Fault Proof Alpha Chad contracts.

## Usage

### Generating the Cannon prestate and artifacts

_Description_: Generates the cannon prestate, tars the relevant artifacts, and sets the absolute prestate field in the network's deploy config.

```sh
make cannon-prestate chain=<chain-name>
```

### Deploying a fresh system

_Description_: Deploys a fully fresh FPAC system to the passed chain. All args after the `args=` are forwarded to `forge script`.

```sh
make deploy-fresh chain=<chain-name> proxy-admin=<chain-proxy-admin-addr> system-owner-safe=<chain-safe-addr> [args=<forge-script-args>]
```
