# @eth-optimism/chugsplash

ChugSplash is a smarter smart contract deployment framework.
It's an opinionated tool designed to fix the most common issues that plague smart contract deployments.

## Specification

### Configuration Format

```typescript
interface ChugSplashConfig {
  options: {
    name: string
    owner: address
  }
  contracts: {
    [name: string]: {
      source: string
      address?: string
      variables: {
        [name: string]: any
      }
    }
  }
}
```
