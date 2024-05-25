## Fault Proof Alpha Deployment Information

### Goerli

**Deployments**
| Contract                   | Address                                      |
|--------------------------- |--------------------------------------------- |
| DisputeGameFactory (proxy) | `0xad9e5E6b39F55EE7220A3dC21a640B089196a89f` |
| DisputeGameFactory (impl)  | `0x666C3d298B9c360990F901b3Eded4e7a9d7AD446` |
| BlockOracle                | `0x7979b2D824A6A682D1dA25bD02E544bB66536032` |
| PreimageOracle             | `0xE214d974dE12Cc8d096170AbC5EEBD18F08a044a` |
| MIPS VM                    | `0x78760b9A1Df5DDe037D64376BD4d1d675dC30f0f` |
| FaultDisputeGame (impl)    | `0xBE2827A6c62d39b4C7933D592B6913412D5aBC77` |

**Configuration**
- Absolute prestate hash:  `0x0357393f50acca498e446f69292fce66c93a6d9038aa277b47c93fa46ce85108`
- Max game depth: `40`
    - Supports an instruction trace up to `1,099,511,627,776` instructions long.
- Max game duration: `172,800 seconds` (2 days)

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
