# The Verify Command

Once you have deployed contracts via [bootstrap][bootstrap], you can use the `verify` command to verify the source code on Etherscan. Constructor args used in the verification request are extracted automatically from contract initcode via the tx that created the contract.

[bootstrap]: bootstrap.md

You can call the `verify` command like this:

```shell
op-deployer verify \
  --l1-rpc-url <l1 rpc url> \
  --input-file <filepath to input .json file> \
  --etherscan-api-key <your free etherscan api key> \
  --artifacts-locator <l1 forge-artifacts locator>
```

## CLI Args

### `--l1-rpc-url`

Defines the RPC URL of the L1 chain to deploy to (currently only supports mainnet and sepolia).

### `--input-file`

The full filepath to the input .json file. This file should be a key/value store where the key is a contract name and the value is the contract address. The output of the `bootstrap superchain|implementations` commands is a good example of this format, and those output files can be fed directly into `verify`. Unless the `--contract-name` flag is passed, all contracts in the input file will be verified.

{
  "Opcm": "0x3a1f523a4bc09cd344a2745a108bb0398288094f",
  "OpcmContractsContainer": "0x660aeaac7508258f622cfdc489c16c864b4d8629",
  "OpcmGameTypeAdder": "0xc9060f6283b78e1feebfd1993cb6350b5626f115",
  "OpcmDeployer": "0x88e39ea5cfe6c4d450305eec5fd90dd1fba87f45",
  "OpcmUpgrader": "0xbf098a12edcf99f8e6db258b7ac567a1fd020f4b",
  "DelayedWETHImpl": "0x5e40b9231b86984b5150507046e354dbfbed3d9e",
  "OptimismPortalImpl": "0xb443da3e07052204a02d630a8933dac05a0d6fb4",
  "PreimageOracleSingleton": "0x1fb8cdfc6831fc866ed9c51af8817da5c287add3",
  "MipsSingleton": "0xf027f4a985560fb13324e943edf55ad6f1d15dc1",
  "SystemConfigImpl": "0x340f923e5c7cbb2171146f64169ec9d5a9ffe647",
  "L1CrossDomainMessengerImpl": "0x5d5a095665886119693f0b41d8dfee78da033e8b",
  "L1ERC721BridgeImpl": "0x7ae1d3bd877a4c5ca257404ce26be93a02c98013",
  "L1StandardBridgeImpl": "0x0b09ba359a106c9ea3b181cbc5f394570c7d2a7a",
  "OptimismMintableERC20FactoryImpl": "0x5493f4677a186f64805fe7317d6993ba4863988f",
  "DisputeGameFactoryImpl": "0x4bba758f006ef09402ef31724203f316ab74e4a0",
  "AnchorStateRegistryImpl": "0x7b465370bb7a333f99edd19599eb7fb1c2d3f8d2",
  "SuperchainConfigImpl": "0x4da82a327773965b8d4d85fa3db8249b387458e7",
  "ProtocolVersionsImpl": "0x37e15e4d6dffa9e5e320ee1ec036922e563cb76c"
}

### `--contract-name` (optional)

Specifies a single contract name, matching a contract key within the input file, to verify. If not provided, all contracts in the input file will be verified.

### `--artifacts-locator`

The locator to forge-artifacts containing the output of the `forge build` command (i.e. compiled bytecode and solidity source code). This can be a local path (with a `file://` prefix), remote URL (with a `http://` or `https://` prefix), or standard contracts tag (with a `tag://op-contracts/v` prefix).

## Output

Output logs will be printed to the console and look something like the following. If the final results show `numFailed=0`, all contracts were verified successfully.
```sh
INFO [03-05|15:56:55.900] Formatting etherscan verify request      name=superchainConfigProxyAddress            address=0x805fc6750ec23bdD58f7BBd6ce073649134C638A
INFO [03-05|15:56:55.900] Opening artifact                         path=Proxy.sol/Proxy.json           name=superchainConfigProxyAddress
INFO [03-05|15:56:55.905] contractName                             name=src/universal/Proxy.sol:Proxy
INFO [03-05|15:56:55.905] Extracting constructor args from initcode address=0x805fc6750ec23bdD58f7BBd6ce073649134C638A argSlots=1
INFO [03-05|15:56:56.087] Contract creation tx hash                txHash=0x71b377ccc11304afc32e1016c4828a34010a0d3d81701c7164fb19525ba4fbc4
INFO [03-05|15:56:56.494] Successfully extracted constructor args  address=0x805fc6750ec23bdD58f7BBd6ce073649134C638A
INFO [03-05|15:56:56.683] Verification request submitted           name=superchainConfigProxyAddress            address=0x805fc6750ec23bdD58f7BBd6ce073649134C638A
INFO [03-05|15:57:02.035] Verification complete                    name=superchainConfigProxyAddress            address=0x805fc6750ec23bdD58f7BBd6ce073649134C638A
INFO [03-05|15:57:02.208] Formatting etherscan verify request      name=protocolVersionsImplAddress             address=0x658812BEb9bF6286D03fBF1B5B936e1af490b768
INFO [03-05|15:57:02.208] Opening artifact                         path=ProtocolVersions.sol/ProtocolVersions.json name=protocolVersionsImplAddress
INFO [03-05|15:57:02.215] contractName                             name=src/L1/ProtocolVersions.sol:ProtocolVersions
INFO [03-05|15:57:02.418] Verification request submitted           name=protocolVersionsImplAddress             address=0x658812BEb9bF6286D03fBF1B5B936e1af490b768
INFO [03-05|15:57:07.789] Verification complete                    name=protocolVersionsImplAddress             address=0x658812BEb9bF6286D03fBF1B5B936e1af490b768
INFO [03-05|15:57:07.971] Contract is already verified             name=protocolVersionsProxyAddress            address=0x17C64430Fa08475D41801Dfe36bAFeE9667c6fA7
INFO [03-05|15:57:07.971] --- COMPLETE ---
INFO [03-05|15:57:07.971] final results                            numVerified=4 numSkipped=1 numFailed=0
```

## Known Limitations

- Does not currently work for contracts in the `opchain` bundle (deployed via `op-deployer apply`) that have constructor args. Those constructors args cannot be extracted from the deployment `tx.Data()` since `OPContractsManager.deploy()` uses factory pattern with CREATE2 to deploy those contracts.

- Currently only supports etherscan block explorers. Blockscout support is planned but not yet implemented.
