# Superchain Playground

Implementation of the [superchain playground](https://playground.optimism.io). This site contains **non-production** reference and demo components for the various op-stack features

```bash
$ pnpm i
$ VITE_WALLET_CONNECT_PROJECT_ID=foobarbaz pnpm dev
```

## Configuration

If using [wallet connect](https://walletconnect.network/) to connect to the playground, ensure a valid wallet connect is set in the environment

```bash
$ export VITE_WALLET_CONNECT_PROJECT_ID=
```

> The default dev account provided is the first pre-funded private key available in supersim
