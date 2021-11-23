# @eth-optimism/chugsplash

ChugSplash is a smarter smart contract deployment framework.
It's an opinionated tool designed to fix the most common issues that plague smart contract deployments.

## Features

### Deployment

ChugSplash is a declarative contract deployment system.

```ts
const contracts: ChugSplashDeployConfig = {
  // Contract addresses are determined by the address of the primary deployment manager
  MyContract: {
    // Alternatively, contract addresses can be specified manually
    address: `0x${'11'.repeat(20)}`,
    // Contract source should come from an IPFS hash or something
    source: 'MyContract',
    // Specify contract variables. You can do `verify` or `deploy` or `upgrade`.
    variables: {},
  },
}
```
