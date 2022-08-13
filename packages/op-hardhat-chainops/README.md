# @eth-optimism/op-hardhat-chainops

`op-hardhat-chainops` is a Hardhat plugin that will automatically connect
to all of the Optimism contracts given a hardhat network.

## Installation

```bash
yarn add @eth-optimism/op-hardhat-chainops
```

## Usage

In the `hardhat.config.ts`, import the package.

```typescript
import '@eth-optimism/op-hardhat-chainops'
```

In a hardhat task, be sure to also import the package so that
the typings resolve. The `HardhatRuntimeEnvironment` is entended
with an additional `optimism` field that contains useful things
for interacting with Optimism.

An example is shown below that prints off the Optimism contract
addresses for goerli. Each contract is an `ethers.Contract`
that is attached to the correct network (L1/L2).

```typescript
task('addresses', 'Get the addresses of the Optimism contracts')
  .setAction(async (args, hre) => {
      await hre.optimism.init()
      const contracts = hre.optimism.contracts

      for (const [name, contract] of Object.entries(contracts)) {
          console.log(`${name} ${contract.address}`)
      }
  })
```

```bash
npx hardhat addresses --network goerli
```
