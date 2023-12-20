## Chains

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=contracts-bedrock-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

Chain constants generated from the [superchain registry](https://github.com/ethereum-optimism/superchain-registry)

The chains are shaped as a [viem chain](https://github.com/wevm/viem/tree/main/src/chains/definitions) for easy interop with [viem](https://viem.sh/op-stack#3-consume-op-stack-actions) and [op-wagmi](https://github.com/base-org/op-wagmi)

## Installation

```bash
npm install @eth-optimism/chains
```

## Usage

```typescript
import { base, optimism, pgn } from '@eth-optimism/chains'
```
