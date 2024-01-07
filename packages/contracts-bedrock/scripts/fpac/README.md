# `fpac-deploy`

Chain-ops scripts for the Fault Proof Alpha Chad contracts.

## Dependencies

* [just](github.com/casey/just)

## Usage

### Generating the Cannon prestate and artifacts

*Description*: Generates the cannon prestate, tars the relavent artifacts, and sets the absolute prestate field in the network's deploy config.

```sh
just cannon-prestate <chain-name>
```

### Deploying a fresh system

*Description*: Deploys a fully fresh FPAC system to the passed chain. All args after the `chain-name` are forwarded to `forge script`.

```sh
just deploy-fresh <chain-name> [--broadcast]
```

### Upgrading the Game Implementation

*Description*: Upgrades the `CANNON` game type's implementation in the `DisputeGameFactory` that was deployed for the passed `chain-name`. All args after the `chain-name` are forwarded to `forge script`.

```sh
just upgrade-game-impl <chain-name> <dgf-proxy-address> <vm-address> [--broadcast]
```

### Updating Init Bonds

*Description*: Updates the initialization bond for a given game type in the `DisputeGameFactory` that was deployed for the passed `chain-name`. All args after the `chain-name` are forwarded to `forge script`.

```sh
just update-init-bond <chain-name> <dgf-proxy-address> <game-type> <new-init-bond> [--broadcast]
```
