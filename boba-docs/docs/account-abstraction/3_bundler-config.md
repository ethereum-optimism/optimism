# Bundler Config

The Bundler can be run with custom configuration as per the networks requirement. The configuration is through options (or env vars) provided to the bundler while spining it up - and the ways to specify these are - either through a) (reccommended) the env variables when running the script , see [bundler.sh](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/bundler/bundler.sh) b) or through a file `workdir/bundler.config.json`

The Bundler also has defaults set for certain parameters, the current defaults can be seen/set [here](https://github.com/bobanetwork/boba\_legacy/blob/develop/packages/boba/bundler/src/BundlerConfig.ts#L53)

But, its important to remember the following order of precedence while specifying configurations:

```
Shell vars > config file > defaults
```

The complete list of configuration variables that can be customized

```
  beneficiary: ow.string, // account that will receive fees, if any
  entryPoint: ow.string, // entryPoint contract
  entryPointWrapper: ow.optional.string, // entryPoint wrapper contract
  gasFactor: ow.string, 
  minBalance: ow.string, 
  mnemonic: ow.string, //mnemonic file or private key
  network: ow.string, // l2 network
  port: ow.string, // port to run on
  unsafe: ow.boolean, // flag to enable no storage or opcode checks
  conditionalRpc: ow.boolean, // flag to use eth_sendRawTransactionConditional RPC)
  whitelist: ow.optional.array.ofType(ow.string),
  blacklist: ow.optional.array.ofType(ow.string),
  maxBundleGas: ow.number, // max Bundle Gas available to use
  minStake: ow.string, // min stake an account needs to have multiple pending requests
  minUnstakeDelay: ow.number, // unstake delay to withdrawa stake
  autoBundleInterval: ow.number, // time to wait before sending a bundle
  autoBundleMempoolSize: ow.number, // bundle size to wait for before sending a bundle
  addressManager: ow.string, // address manager contract address
  l1NodeWeb3Url: ow.string, // l1 rpc
  enableDebugMethods: ow.boolean, // flag to enable debug methods on bundler
  l2Offset: ow.optional.number, // l2 block the bundler watches from, defaults to 0
  logsChunkSize: ow.optional.number, // the maximum permissble eth_getLogs range supported by the network, defaults to 5000
```

Note- EntryPointWrapper is a requirement for the bundler when it is run against Boba Network - since the sdk also supports v2 of the Boba Network which did not support custom errors.

The EntryPointWrapper routes the following calls, which the bundler needs in order to operate:

* simulateValidation()
* getSenderAddress() and includes the following helper methods-
* getCodeHashes()
* getUserOpHashes()
