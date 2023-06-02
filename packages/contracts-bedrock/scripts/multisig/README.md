## Multisig Operation Scripts

A collection of scripts used by multisig signers to verify the
integrity of the transactions to be signed.


### Contract Verification for Bedrock Migration

[CheckForBedrockMigration.s.sol](./CheckForBedrockMigration.s.sol) is
a script used by the Bedrock migration signers before the migration,
to verify the contracts affected by the migration are always under the
control of the multisig, and security critical configurations are
correctly initialized.

Example usage:

``` bash
git clone git@github.com:ethereum-optimism/optimism.git
cd optimism/
git pull
git checkout develop
nvm use
yarn install
yarn clean
yarn build
cd packages/contracts-bedrock
export L1_UPGRADE_KEY=0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
export BEDROCK_JSON_DIR=deployments/mainnet
forge script scripts/multisig/CheckForBedrockMigration.s.sol --rpc-url <TRUSTWORTHY_L1_RPC_URL>
```

Expected output:

``` bash
Script ran successfully.

  BEDROCK_JSON_DIR = deployments/mainnet
  Checking AddressManager 0xdE1FCfB0851916CA5101820A69b13a4E276bd81F
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0xdE1FCfB0851916CA5101820A69b13a4E276bd81F.owner().
  Checking L1CrossDomainMessenger 0x2150Bc3c64cbfDDbaC9815EF615D6AB8671bfe43
    -- Success: 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed == 0x2150Bc3c64cbfDDbaC9815EF615D6AB8671bfe43.PORTAL().
  Checking L1CrossDomainMessengerProxy 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1.owner().
    -- Success: 0xdE1FCfB0851916CA5101820A69b13a4E276bd81F == 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1.libAddressManager().
  Checking L1ERC721Bridge 0x4afDD3A48E13B305e98D9EEad67B1b5867E370DF
    -- Success: 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1 == 0x4afDD3A48E13B305e98D9EEad67B1b5867E370DF.messenger().
  Checking L1ERC721BridgeProxy 0x5a7749f83b81B301cAb5f48EB8516B986DAef23D
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0x5a7749f83b81B301cAb5f48EB8516B986DAef23D.admin().
    -- Success: 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1 == 0x5a7749f83b81B301cAb5f48EB8516B986DAef23D.messenger().
  Checking L1ProxyAdmin 0x543bA4AADBAb8f9025686Bd03993043599c6fB04
    -- Success: 0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB == 0x543bA4AADBAb8f9025686Bd03993043599c6fB04.owner().
  Checking L1StandardBridge 0xBFB731Cd36D26c2a7287716DE857E4380C73A64a
    -- Success: 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1 == 0xBFB731Cd36D26c2a7287716DE857E4380C73A64a.messenger().
  Checking L1StandardBridgeProxy 0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1.getOwner().
    -- Success: 0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1 == 0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1.messenger().
  Checking L1UpgradeKeyAddress 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
  Checking L2OutputOracle 0xd2E67B6a032F0A9B1f569E63ad6C38f7342c2e00
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0xd2E67B6a032F0A9B1f569E63ad6C38f7342c2e00.CHALLENGER().
    -- Success: 0x0000000000000000000000000000000000093A80 == 0xd2E67B6a032F0A9B1f569E63ad6C38f7342c2e00.FINALIZATION_PERIOD_SECONDS().
  Checking L2OutputOracleProxy 0xdfe97868233d1aa22e815a266982f2cf17685a27
    -- Success: 0x543bA4AADBAb8f9025686Bd03993043599c6fB04 == 0xdfe97868233d1aa22e815a266982f2cf17685a27.admin().
  Checking OptimismMintableERC20Factory 0xaE849EFA4BcFc419593420e14707996936E365E2
    -- Success: 0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1 == 0xaE849EFA4BcFc419593420e14707996936E365E2.BRIDGE().
  Checking OptimismMintableERC20FactoryProxy 0x75505a97BD334E7BD3C476893285569C4136Fa0F
    -- Success: 0x543bA4AADBAb8f9025686Bd03993043599c6fB04 == 0x75505a97BD334E7BD3C476893285569C4136Fa0F.admin().
  Checking OptimismPortal 0x28a55488fef40005309e2DA0040DbE9D300a64AB
    -- Success: 0xdfe97868233d1aa22e815a266982f2cf17685a27 == 0x28a55488fef40005309e2DA0040DbE9D300a64AB.L2_ORACLE().
  Checking OptimismPortalProxy 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed
    -- Success: 0x543bA4AADBAb8f9025686Bd03993043599c6fB04 == 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed.admin().
  Checking PortalSender 0x0A893d9576b9cFD9EF78595963dc973238E78210
    -- Success: 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed == 0x0A893d9576b9cFD9EF78595963dc973238E78210.PORTAL().
  Checking SystemConfigProxy 0x229047fed2591dbec1eF1118d64F7aF3dB9EB290
    -- Success: 0x543bA4AADBAb8f9025686Bd03993043599c6fB04 == 0x229047fed2591dbec1eF1118d64F7aF3dB9EB290.admin().
  Checking SystemDictator 0x09E040a72FD3492355C5aEEdbC3154075f83488a
    -- Success: 0x0000000000000000000000000000000000000000 == 0x09E040a72FD3492355C5aEEdbC3154075f83488a.owner().
  Checking SystemDictatorProxy 0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB
    -- Success: 0x09E040a72FD3492355C5aEEdbC3154075f83488a == 0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB.implementation().
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB.owner().
    -- Success: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A == 0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB.admin().
```
