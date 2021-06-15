# Optimism Regenesis Deployments
## LAYER 2

### Chain IDs:
- Mainnet: 10
- Kovan: 69
- Goerli: 420
*The contracts relevant for the majority of developers are `OVM_ETH` and the cross-domain messengers. The L2 addresses don't change.*

### Predeploy contracts:
|Contract|Address|
|--|--|
|OVM_ETH: | `0x4200000000000000000000000000000000000006`
|OVM_L2CrossDomainMessenger: | `0x4200000000000000000000000000000000000007`
|OVM_L2ToL1MessagePasser: | `0x4200000000000000000000000000000000000000`
|OVM_L1MessageSender: | `0x4200000000000000000000000000000000000001`
|OVM_DeployerWhitelist: | `0x4200000000000000000000000000000000000002`
|OVM_ECDSAContractAccount: | `0x4200000000000000000000000000000000000003`
|OVM_SequencerEntrypoint: | `0x4200000000000000000000000000000000000005`
|Lib_AddressManager: | `0x4200000000000000000000000000000000000008`
|ERC1820Registry: | `0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24`

---
---

## LAYER 1

## OPTIMISTIC-KOVAN

Network : __undefined (chain id: 69)__

|Contract|Address|
|--|--|
|OVM_GasPriceOracle|[0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76](https://undefined.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

-->
---
## MAINNET

Network : __mainnet (chain id: 1)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0x668E5b997b9aE88a56cd40409119d4Db9e6d752E](https://etherscan.io/address/0x668E5b997b9aE88a56cd40409119d4Db9e6d752E)|
|OVM_CanonicalTransactionChain|[0xa88e220c7FC7F0D845D2624a5dF1DfD6874B9a44](https://etherscan.io/address/0xa88e220c7FC7F0D845D2624a5dF1DfD6874B9a44)|
|OVM_ChainStorageContainer-CTC-batches|[0x28157e8a8E6d22A367c63Ad61dD56d9E6bDCE905](https://etherscan.io/address/0x28157e8a8E6d22A367c63Ad61dD56d9E6bDCE905)|
|OVM_ChainStorageContainer-CTC-queue|[0xD68670eD8800c856613FD3e4C55539A2Ff53cCb3](https://etherscan.io/address/0xD68670eD8800c856613FD3e4C55539A2Ff53cCb3)|
|OVM_ChainStorageContainer-SCC-batches|[0x7B8af5f008A7C5eFD319e68Fd5C9A68008519Caf](https://etherscan.io/address/0x7B8af5f008A7C5eFD319e68Fd5C9A68008519Caf)|
|OVM_ExecutionManager|[0x3f5FA555c434b49D946042955013966Fd108DaC3](https://etherscan.io/address/0x3f5FA555c434b49D946042955013966Fd108DaC3)|
|OVM_FraudVerifier|[0x169CC2f69Cc16da17B71Df2dce6161ef57991bB9](https://etherscan.io/address/0x169CC2f69Cc16da17B71Df2dce6161ef57991bB9)|
|OVM_L1MultiMessageRelayer|[0xc34F5B7279A9276A9D02491c59630fa725B7c36B](https://etherscan.io/address/0xc34F5B7279A9276A9D02491c59630fa725B7c36B)|
|OVM_SafetyChecker|[0xD87eFbBb82f1B7d25469641ee2E0E513f144394C](https://etherscan.io/address/0xD87eFbBb82f1B7d25469641ee2E0E513f144394C)|
|OVM_StateCommitmentChain|[0x6786EB419547a4902d285F70c6acDbC9AefAdB6F](https://etherscan.io/address/0x6786EB419547a4902d285F70c6acDbC9AefAdB6F)|
|OVM_StateManagerFactory|[0xA4C213C1E2bF5594baB0BCdF071ed5B0E946b19e](https://etherscan.io/address/0xA4C213C1E2bF5594baB0BCdF071ed5B0E946b19e)|
|OVM_StateTransitionerFactory|[0xAe4ef5e45C42dA513d2B48E184B64A05c18d8154](https://etherscan.io/address/0xAe4ef5e45C42dA513d2B48E184B64A05c18d8154)|
|Proxy__OVM_L1CrossDomainMessenger|[0x902e5fF5A99C4eC1C21bbab089fdabE32EF0A5DF](https://etherscan.io/address/0x902e5fF5A99C4eC1C21bbab089fdabE32EF0A5DF)|
|Proxy__OVM_L1ETHGateway|[0xe681F80966a8b1fFadECf8068bD6F99034791c95](https://etherscan.io/address/0xe681F80966a8b1fFadECf8068bD6F99034791c95)|
|mockOVM_BondManager|[0x90c5F8d045bBcCc99d907f30E8707F06D95d065b](https://etherscan.io/address/0x90c5F8d045bBcCc99d907f30E8707F06D95d065b)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0x598F2b19e983910529affAb7D219724a019339CC
 - https://etherscan.io/address/0x598F2b19e983910529affAb7D219724a019339CC)
OVM_L1ETHGateway: 
 - 0x40c9067ec8087EcF101FC10d2673636955b81A32
 - https://etherscan.io/address/0x40c9067ec8087EcF101FC10d2673636955b81A32)
-->
---
## KOVAN

Network : __kovan (chain id: 42)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xd56F695e73286ac252A37593DD4E7c14270eC1Df](https://kovan.etherscan.io/address/0xd56F695e73286ac252A37593DD4E7c14270eC1Df)|
|OVM_CanonicalTransactionChain|[0x895eabB95D684c15fa46Dc00a6b7557450083DEF](https://kovan.etherscan.io/address/0x895eabB95D684c15fa46Dc00a6b7557450083DEF)|
|OVM_ChainStorageContainer-CTC-batches|[0xeb335a8A5e8bA008cF7Cb02D5C3432f4fDB576da](https://kovan.etherscan.io/address/0xeb335a8A5e8bA008cF7Cb02D5C3432f4fDB576da)|
|OVM_ChainStorageContainer-CTC-queue|[0x207fa9Aa7Dee9AA790A8DF64778D3E3B6273BC90](https://kovan.etherscan.io/address/0x207fa9Aa7Dee9AA790A8DF64778D3E3B6273BC90)|
|OVM_ChainStorageContainer-SCC-batches|[0xFE1CE27173676A6850ECF4e0536D7C468A4dAfa0](https://kovan.etherscan.io/address/0xFE1CE27173676A6850ECF4e0536D7C468A4dAfa0)|
|OVM_ExecutionManager|[0xa2EB1961183a04157fF707Fa2Be2249e149c8FAB](https://kovan.etherscan.io/address/0xa2EB1961183a04157fF707Fa2Be2249e149c8FAB)|
|OVM_FraudVerifier|[0x4B2F74938Ddb8742C33b46aD1a402c85e9dABC44](https://kovan.etherscan.io/address/0x4B2F74938Ddb8742C33b46aD1a402c85e9dABC44)|
|OVM_L1MultiMessageRelayer|[0x942b1B1CaF9e7654318CbfCfD1bca6727C716638](https://kovan.etherscan.io/address/0x942b1B1CaF9e7654318CbfCfD1bca6727C716638)|
|OVM_SafetyChecker|[0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a](https://kovan.etherscan.io/address/0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a)|
|OVM_StateCommitmentChain|[0xdB1367bB36C34618778D492725C3eD11B508aC54](https://kovan.etherscan.io/address/0xdB1367bB36C34618778D492725C3eD11B508aC54)|
|OVM_StateManagerFactory|[0x3b96673C9e24D362501e87B239F60543e20beD50](https://kovan.etherscan.io/address/0x3b96673C9e24D362501e87B239F60543e20beD50)|
|OVM_StateTransitionerFactory|[0xd6eDb16a89A2EE4484fa8fdCDb11B8B5633c3687](https://kovan.etherscan.io/address/0xd6eDb16a89A2EE4484fa8fdCDb11B8B5633c3687)|
|Proxy__OVM_L1CrossDomainMessenger|[0x78b88FD62FBdBf67b9C5C6528CF84E9d30BB28e0](https://kovan.etherscan.io/address/0x78b88FD62FBdBf67b9C5C6528CF84E9d30BB28e0)|
|Proxy__OVM_L1ETHGateway|[0xB191d67F69e823445cD59e5A88953a82be73b9C6](https://kovan.etherscan.io/address/0xB191d67F69e823445cD59e5A88953a82be73b9C6)|
|mockOVM_BondManager|[0x8ECe272C9f83041bcb1Cd57AC49Ca6494776bE01](https://kovan.etherscan.io/address/0x8ECe272C9f83041bcb1Cd57AC49Ca6494776bE01)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0xa9D9045E4A753c856Fc0053369E780f23559E0A1
 - https://kovan.etherscan.io/address/0xa9D9045E4A753c856Fc0053369E780f23559E0A1)
OVM_L1ETHGateway: 
 - 0x25bb69ee5665536Ce6aeb51094F0bed9e4DACc30
 - https://kovan.etherscan.io/address/0x25bb69ee5665536Ce6aeb51094F0bed9e4DACc30)
-->
---
## GOERLI

Network : __goerli (chain id: 5)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xA4346c8c120DdCE2c5447e68790625F10Bb4d47A](https://goerli.etherscan.io/address/0xA4346c8c120DdCE2c5447e68790625F10Bb4d47A)|
|OVM_CanonicalTransactionChain|[0x4781674AAe242bbDf6C58b81Cf4F06F1534cd37d](https://goerli.etherscan.io/address/0x4781674AAe242bbDf6C58b81Cf4F06F1534cd37d)|
|OVM_ChainStorageContainer-CTC-batches|[0xd5F2B9f6Ee80065b2Ce18bF1e629c5aC1C98c7F6](https://goerli.etherscan.io/address/0xd5F2B9f6Ee80065b2Ce18bF1e629c5aC1C98c7F6)|
|OVM_ChainStorageContainer-CTC-queue|[0x3EA657c5aA0E4Bce1D8919dC7f248724d7B0987a](https://goerli.etherscan.io/address/0x3EA657c5aA0E4Bce1D8919dC7f248724d7B0987a)|
|OVM_ChainStorageContainer-SCC-batches|[0x777adA49d40DAC02AE5b4FdC292feDf9066435A3](https://goerli.etherscan.io/address/0x777adA49d40DAC02AE5b4FdC292feDf9066435A3)|
|OVM_ExecutionManager|[0x838a74bAdfD28Fd0e32E4A88BddDa502D56ae7F7](https://goerli.etherscan.io/address/0x838a74bAdfD28Fd0e32E4A88BddDa502D56ae7F7)|
|OVM_FraudVerifier|[0x916f75037b87Bf4Fe0Dc7719815bd972F0618669](https://goerli.etherscan.io/address/0x916f75037b87Bf4Fe0Dc7719815bd972F0618669)|
|OVM_L1MultiMessageRelayer|[0x2545fa928d5d278cA75Fd47306e4a89096ff6403](https://goerli.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)|
|OVM_SafetyChecker|[0x71D4ea896C9a2D4a973CC5c7E347B6707691ECa0](https://goerli.etherscan.io/address/0x71D4ea896C9a2D4a973CC5c7E347B6707691ECa0)|
|OVM_StateCommitmentChain|[0x9bA5E286934F0A29fb2f8421f60d3eE8A853447C](https://goerli.etherscan.io/address/0x9bA5E286934F0A29fb2f8421f60d3eE8A853447C)|
|OVM_StateManagerFactory|[0x24C7F0a4a2B926613B31c4cDDA4c0f90c0772f2b](https://goerli.etherscan.io/address/0x24C7F0a4a2B926613B31c4cDDA4c0f90c0772f2b)|
|OVM_StateTransitionerFactory|[0x703303Ce2d92Ef95F17a622E3d538390251165E8](https://goerli.etherscan.io/address/0x703303Ce2d92Ef95F17a622E3d538390251165E8)|
|Proxy__OVM_L1CrossDomainMessenger|[0xa85716330ff84Ab312D5B43F3BfDcC7E650fd88A](https://goerli.etherscan.io/address/0xa85716330ff84Ab312D5B43F3BfDcC7E650fd88A)|
|Proxy__OVM_L1ETHGateway|[0xA721CF3e39E5cB4CfEEc0e32EE05B3D05AA9aE39](https://goerli.etherscan.io/address/0xA721CF3e39E5cB4CfEEc0e32EE05B3D05AA9aE39)|
|Proxy__OVM_L1StandardBridge|[0x74B6CC2F377fB769cEd6c697bC4C58a9c342E424](https://goerli.etherscan.io/address/0x74B6CC2F377fB769cEd6c697bC4C58a9c342E424)|
|mockOVM_BondManager|[0x795F355F75f9B28AEC6cC6A887704191e630065b](https://goerli.etherscan.io/address/0x795F355F75f9B28AEC6cC6A887704191e630065b)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0x3B1D4DE5F7Fe8487980Ee7608BE302dC60a9caE9
 - https://goerli.etherscan.io/address/0x3B1D4DE5F7Fe8487980Ee7608BE302dC60a9caE9)
OVM_L1ETHGateway: 
 - 0x746E840b94cC75921D1cb620b83CFd0C658B2852
 - https://goerli.etherscan.io/address/0x746E840b94cC75921D1cb620b83CFd0C658B2852)
-->
---
