# The Verify Command

Once you have deployed contracts via [bootstrap][bootstrap] or [apply][apply], you can use the `verify` command to verify the source code on block explorers like Etherscan or Blockscout. The command uses the `forge verify-contract` binary, which automatically handles constructor argument detection and source code verification.

[bootstrap]: bootstrap.md
[apply]: apply.md

You can call the `verify` command like this:

```shell
op-deployer verify \
  --l1-rpc-url <l1 rpc url> \
  --input-file <filepath to input .json or state.json file> \
  --verifier-api-key <your api key> \
  --artifacts-locator <l1 forge-artifacts locator> \
  --verifier etherscan
```

For Blockscout verification (uses default URLs for mainnet/sepolia, no API key required):

```shell
op-deployer verify \
  --l1-rpc-url <l1 rpc url> \
  --input-file <filepath to input .json or state.json file> \
  --artifacts-locator <l1 forge-artifacts locator> \
  --verifier blockscout
```

For custom block explorer verification (Etherscan v2-compatible, API key may be required):

```shell
op-deployer verify \
  --l1-rpc-url <l1 rpc url> \
  --input-file <filepath to input .json or state.json file> \
  --artifacts-locator <l1 forge-artifacts locator> \
  --verifier custom \
  --verifier-url <custom etherscan v2 compatible api url>
```

## CLI Args

### `--l1-rpc-url`

Defines the RPC URL of the L1 chain to deploy to (currently only supports mainnet and sepolia).

### `--input-file`

The full filepath to the input file. This can be either:
- A simple JSON file with contract name/address pairs (output from `bootstrap superchain|implementations`)
- A complete `state.json` file (output from `apply`)

The verifier automatically detects the file format and extracts all contracts. Unless the `--contract-name` flag is passed, all contracts in the input file will be verified.

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

### `--verifier`

The block explorer(s) to use for verification. Supports multiple verifiers separated by commas.

Options:
- `etherscan` (default): Uses Etherscan for mainnet/sepolia
- `blockscout`: Uses default Blockscout URLs for mainnet/sepolia
- `custom`: For custom Etherscan v2-compatible instances (requires `--verifier-url`)

Examples:
- Single verifier: `--verifier etherscan`
- Multiple verifiers: `--verifier etherscan,blockscout` (verifies on both)

### `--verifier-url`

The verifier API URL. Usage varies by verifier type:
- `etherscan`: Ignored (automatically determined from chain ID)
- `blockscout`: Optional (defaults to standard Blockscout URLs for mainnet/sepolia)
- `custom`: Required. Example: `https://etherscanv2.compat-api.example.com/api`

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

## Automatic Verification

You can automatically verify contracts after deployment by using the `--verify` flag with `apply` or `bootstrap` commands:

```shell
op-deployer apply \
  --workdir ./.deployer \
  --l1-rpc-url <l1 rpc url> \
  --private-key <deployer private key> \
  --verify \
  --verifier-api-key <your api key>
```

This will verify all deployed contracts at the end of the deployment process.

### Multi-Verifier Deployment

You can verify on multiple block explorers simultaneously:

```shell
op-deployer bootstrap superchain \
  --l1-rpc-url <l1 rpc url> \
  --private-key <deployer private key> \
  --outfile ./superchain.json \
  --superchain-proxy-admin-owner <owner address> \
  --protocol-versions-owner <owner address> \
  --guardian <guardian address> \
  --verify \
  --verifier etherscan,blockscout \
  --verifier-api-key <etherscan api key>
```

This will:
1. Deploy the superchain contracts
2. Verify on Etherscan (using the API key)
3. Verify on Blockscout (no API key required)
4. Report combined results from both verifiers

## Supported Contract Bundles

The verify command now supports all contract bundles:
- **Superchain** contracts (from `bootstrap superchain`)
- **Implementations** contracts (from `bootstrap implementations`)
- **OpChain** contracts (from `apply` - including all chain-specific contracts)

When using a `state.json` file from `apply`, the verifier automatically extracts and verifies contracts from all deployment stages.

## Block Explorer Support

The verification command supports both Etherscan and Blockscout block explorers through the forge binary, alongside any Etherscan v2 compatible APIs.
