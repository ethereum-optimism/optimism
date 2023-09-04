## Fault Proof Alpha Deployment Information

### Goerli

Information on the fault proofs alpha deployment to Goerli is not yet available.

### Local Devnet

The local devnet includes a deployment of the fault proof alpha. To start the devnet, in the top level of this repo,
run:

```bash
make devnet-up
```

| Input                | Value                                                       |
|----------------------|-------------------------------------------------------------|
| Dispute Game Factory | Run `jq -r .DisputeGameFactoryProxy .devnet/addresses.json` |
| Absolute Prestate    | `op-program/bin/prestate.json`                              |
| Max Depth            | 30                                                          |
| Max Game Duration    | 1200 (20 minutes)                                           |

See the [op-challenger README](../../op-challenger#running-with-cannon-on-local-devnet) for information on
running `op-challenger` against the local devnet.
