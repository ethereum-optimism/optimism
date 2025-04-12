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

Example:
```json
{
  "opcmAddress": "0x437d303c20ea12e0edba02478127b12cbad54626",
  "opcmContractsContainerAddress": "0xf89d7ce62fc3a18354b37b045017d585f7e332ab",
  "opcmGameTypeAdderAddress": "0x9aa4b6c0575e978dbe6d6bc31b7e4403ea8bd81d",
  "opcmDeployerAddress": "0x535388c15294dc77a287430926aba5ba5fe6016a",
  "opcmUpgraderAddress": "0x68a7a93750eb56dd043f5baa41022306e6cd50fa",
  "delayedWETHImplAddress": "0x33ddc90167c923651e5aef8b14bc197f3e8e7b56",
  "optimismPortalImplAddress": "0x54b75cb6f44e36768912e070cd9cb995fc887e6c",
  "ethLockboxImplAddress": "0x05484deeb3067a5332960ca77a5f5603df878ced",
  "preimageOracleSingletonAddress": "0xfbcd4b365f97cb020208b5875ceaf6de76ec068b",
  "mipsSingletonAddress": "0xcc50288ad0d79278397785607ed675292dce37b1",
  "systemConfigImplAddress": "0xfb24aa6d99824b2c526768e97b23694aa3fe31d6",
  "l1CrossDomainMessengerImplAddress": "0x957c0bf84fe541efe46b020a6797fb1fb2eaa6ac",
  "l1ERC721BridgeImplAddress": "0x62786d16978436f5d85404735a28b9eb237e63d0",
  "l1StandardBridgeImplAddress": "0x6c9b377c00ec7e6755aec402cd1cfff34fa75728",
  "optimismMintableERC20FactoryImplAddress": "0x3842175f3af499c27593c772c0765f862b909b93",
  "disputeGameFactoryImplAddress": "0x70ed1725abb48e96be9f610811e33ed8a0fa97f9",
  "anchorStateRegistryImplAddress": "0xce2206af314e5ed99b48239559bdf8a47b7524d4",
  "superchainConfigImplAddress": "0x77008cdc99fb1cf559ac33ca3a67a4a2f04cc5ef",
  "protocolVersionsImplAddress": "0x32e07ddb36833cae3ca1ec5f73ca348a7e9467f4"
}
```

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
