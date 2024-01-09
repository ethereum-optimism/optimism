# `fpac-deploy`

Chain-ops scripts for the Fault Proof Alpha Chad contracts.

## Usage

### Generating the Cannon prestate and artifacts

*Description*: Generates the cannon prestate, tars the relavent artifacts, and sets the absolute prestate field in the network's deploy config.

```sh
make cannon-prestate chain=<chain-name>
```

### Deploying a fresh system

*Description*: Deploys a fully fresh FPAC system to the passed chain. All args after the `chain-name` are forwarded to `forge script`.

```sh
make deploy-fresh chain=<chain-name> [args=<forge-script-args>]
```

### Upgrading the Game Implementation

*Description*: Upgrades the `CANNON` game type's implementation in the `DisputeGameFactory` that was deployed for the passed `chain-name`. All args after the `chain-name` are forwarded to `forge script`.

```sh
make upgrade-game-impl chain=<chain-name> dgf=<dgf-proxy-address> vm=<vm-address> [args=<forge-script-args>]
```

### Updating Init Bonds

*Description*: Updates the initialization bond for a given game type in the `DisputeGameFactory` that was deployed for the passed `chain-name`. All args after the `chain-name` are forwarded to `forge script`.

```sh
make update-init-bond chain=<chain-name> dgf=<dgf-proxy-address> game-type=<game-type> new-init-bond=<new-init-bond> [args=<forge-script-args>]
```
