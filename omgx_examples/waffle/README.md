
## 1. Compiling, running, and deploying.

For example, to compile, test, and the deploy tyhe contract on a local L2"

```bash
$ yarn
$ yarn compile:ovm
$ yarn test:integration:ovm
```

NOTE: you can deploy the contract on OMGX Rinkeby, by changing the target in `/test/erc20.spec.js`:

````bash
const config = {
  l2Url: process.env.L2_URL || 'http://127.0.0.1:8545', // 'http://rinkeby.omgx.network'
  l1Url: process.env.L1_URL || 'http://127.0.0.1:9545',
  useL2: process.env.TARGET === 'OVM',
  privateKey: process.env.PRIVATE_KEY || '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'
}
```