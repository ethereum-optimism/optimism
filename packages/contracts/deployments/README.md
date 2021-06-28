# Optimism Regenesis Deployments
## LAYER 2

### Chain IDs:
- Mainnet: 10
- Kovan: 69
- Goerli: 420
*The contracts relevant for the majority of developers are `OVM_ETH` and the cross-domain messengers. The L2 addresses don't change.*

### Predeploy contracts:
| Contract                    | Address                                      |
| --------------------------- | -------------------------------------------- |
| OVM_ETH:                    | `0x4200000000000000000000000000000000000006` |
| OVM_L2StandardBridge:       | `0x4200000000000000000000000000000000000010` |
| OVM_L2CrossDomainMessenger: | `0x4200000000000000000000000000000000000007` |
| OVM_L2ToL1MessagePasser:    | `0x4200000000000000000000000000000000000000` |
| OVM_L1MessageSender:        | `0x4200000000000000000000000000000000000001` |
| OVM_DeployerWhitelist:      | `0x4200000000000000000000000000000000000002` |
| OVM_ECDSAContractAccount:   | `0x4200000000000000000000000000000000000003` |
| OVM_SequencerEntrypoint:    | `0x4200000000000000000000000000000000000005` |
| Lib_AddressManager:         | `0x4200000000000000000000000000000000000008` |
| OVM_SequencerFeeVault:      | `0x4200000000000000000000000000000000000011` |
| ERC1820Registry:            | `0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24` |

---
---

## LAYER 1

## OPTIMISTIC-KOVAN

Network : __undefined (chain id: 69)__

| Contract           | Address                                                                                                                         |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| OVM_GasPriceOracle | [0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76](https://undefined.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76) |
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

-->
---
## MAINNET

Network : __mainnet (chain id: 1)__

| Contract                              | Address                                                                                                               |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Lib_AddressManager                    | [0xdE1FCfB0851916CA5101820A69b13a4E276bd81F](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F) |
| OVM_CanonicalTransactionChain         | [0x4BF681894abEc828B212C906082B444Ceb2f6cf6](https://etherscan.io/address/0x4BF681894abEc828B212C906082B444Ceb2f6cf6) |
| OVM_ChainStorageContainer-CTC-batches | [0x3EA1a3839D8ca9a7ff3c567a9F36f4C4DbECc3eE](https://etherscan.io/address/0x3EA1a3839D8ca9a7ff3c567a9F36f4C4DbECc3eE) |
| OVM_ChainStorageContainer-CTC-queue   | [0xA0b912b3Ea71A04065Ff82d3936D518ED6E38039](https://etherscan.io/address/0xA0b912b3Ea71A04065Ff82d3936D518ED6E38039) |
| OVM_ChainStorageContainer-SCC-batches | [0x77eBfdFcC906DDcDa0C42B866f26A8D5A2bb0572](https://etherscan.io/address/0x77eBfdFcC906DDcDa0C42B866f26A8D5A2bb0572) |
| OVM_ExecutionManager                  | [0x2745C24822f542BbfFB41c6cB20EdF766b5619f5](https://etherscan.io/address/0x2745C24822f542BbfFB41c6cB20EdF766b5619f5) |
| OVM_FraudVerifier                     | [0x042065416C5c665dc196076745326Af3Cd840D15](https://etherscan.io/address/0x042065416C5c665dc196076745326Af3Cd840D15) |
| OVM_L1MultiMessageRelayer             | [0xF26391FBB1f77481f80a7d646AC08ba3817eA891](https://etherscan.io/address/0xF26391FBB1f77481f80a7d646AC08ba3817eA891) |
| OVM_SafetyChecker                     | [0xfe1F9Cf28ecDb12110aa8086e6FD343EA06035cC](https://etherscan.io/address/0xfe1F9Cf28ecDb12110aa8086e6FD343EA06035cC) |
| OVM_StateCommitmentChain              | [0xE969C2724d2448F1d1A6189d3e2aA1F37d5998c1](https://etherscan.io/address/0xE969C2724d2448F1d1A6189d3e2aA1F37d5998c1) |
| OVM_StateManagerFactory               | [0xd0e3e318154716BD9d007E1E6B021Eab246ff98d](https://etherscan.io/address/0xd0e3e318154716BD9d007E1E6B021Eab246ff98d) |
| OVM_StateTransitionerFactory          | [0x38A6ed6fd76035684caDef38cF49a2FffA782B67](https://etherscan.io/address/0x38A6ed6fd76035684caDef38cF49a2FffA782B67) |
| Proxy__OVM_L1CrossDomainMessenger     | [0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1) |
| Proxy__OVM_L1StandardBridge           | [0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1) |
| mockOVM_BondManager                   | [0xCd76de5C57004d47d0216ec7dAbd3c72D8c49057](https://etherscan.io/address/0xCd76de5C57004d47d0216ec7dAbd3c72D8c49057) |
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger:
 - 0x16393737D09d2722AD13DcA3cA8C3DB957699F1D
 - https://etherscan.io/address/0x16393737D09d2722AD13DcA3cA8C3DB957699F1D

OVM_L1ETHGateway:
 - 0x40c9067ec8087EcF101FC10d2673636955b81A32
 - https://etherscan.io/address/0x40c9067ec8087EcF101FC10d2673636955b81A32
-->
---
## KOVAN

Network : __kovan (chain id: 42)__

| Contract                              | Address                                                                                                                     |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Lib_AddressManager                    | [0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a](https://kovan.etherscan.io/address/0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a) |
| OVM_CanonicalTransactionChain         | [0xe28c499EB8c36C0C18d1bdCdC47a51585698cb93](https://kovan.etherscan.io/address/0xe28c499EB8c36C0C18d1bdCdC47a51585698cb93) |
| OVM_ChainStorageContainer-CTC-batches | [0xF95D79298FD12e5ED778CCf717aA30f638b060E1](https://kovan.etherscan.io/address/0xF95D79298FD12e5ED778CCf717aA30f638b060E1) |
| OVM_ChainStorageContainer-CTC-queue   | [0x2BE00E5F043a0f62c3e4d775F3235E28A0239395](https://kovan.etherscan.io/address/0x2BE00E5F043a0f62c3e4d775F3235E28A0239395) |
| OVM_ChainStorageContainer-SCC-batches | [0x50DA41A2A185fb917aecEFfa1CB4534dC5C264b4](https://kovan.etherscan.io/address/0x50DA41A2A185fb917aecEFfa1CB4534dC5C264b4) |
| OVM_ExecutionManager                  | [0xC68795aC9d96374eaE746DAcC1334ba54798e17D](https://kovan.etherscan.io/address/0xC68795aC9d96374eaE746DAcC1334ba54798e17D) |
| OVM_FraudVerifier                     | [0xaeEd60e029Eb435f960d78C355786060589738B3](https://kovan.etherscan.io/address/0xaeEd60e029Eb435f960d78C355786060589738B3) |
| OVM_L1MultiMessageRelayer             | [0x5818840763Ee28ff0A3E3e8CB9eDeDd07Fb1Cd3f](https://kovan.etherscan.io/address/0x5818840763Ee28ff0A3E3e8CB9eDeDd07Fb1Cd3f) |
| OVM_SafetyChecker                     | [0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a](https://kovan.etherscan.io/address/0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a) |
| OVM_StateCommitmentChain              | [0xa2487713665AC596b0b3E4881417f276834473d2](https://kovan.etherscan.io/address/0xa2487713665AC596b0b3E4881417f276834473d2) |
| OVM_StateManagerFactory               | [0xBcca22E9F5579193E27dD39aD821A03778C44EFA](https://kovan.etherscan.io/address/0xBcca22E9F5579193E27dD39aD821A03778C44EFA) |
| OVM_StateTransitionerFactory          | [0xFD7B9268e790837d393Fd371Ddeb42FE5EC45B54](https://kovan.etherscan.io/address/0xFD7B9268e790837d393Fd371Ddeb42FE5EC45B54) |
| Proxy__OVM_L1CrossDomainMessenger     | [0x4361d0F75A0186C05f971c566dC6bEa5957483fD](https://kovan.etherscan.io/address/0x4361d0F75A0186C05f971c566dC6bEa5957483fD) |
| Proxy__OVM_L1StandardBridge           | [0x22F24361D548e5FaAfb36d1437839f080363982B](https://kovan.etherscan.io/address/0x22F24361D548e5FaAfb36d1437839f080363982B) |
| mockOVM_BondManager                   | [0xD6143943447DFf503d948Fba3D8af3d4Df28f45c](https://kovan.etherscan.io/address/0xD6143943447DFf503d948Fba3D8af3d4Df28f45c) |
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger:
 - 0xa172330C2E6Ec2bF04662Bb9b67ae857910b7f7f
 - https://kovan.etherscan.io/address/0xa172330C2E6Ec2bF04662Bb9b67ae857910b7f7f)
OVM_L1StandardBridge:
 - 0x8293ab0dc701a1387031a13786276f1baa3fcd4e
 - https://kovan.etherscan.io/address/0x8293ab0dc701a1387031a13786276f1baa3fcd4e)
-->
---
## GOERLI

Network : __goerli (chain id: 5)__

| Contract                              | Address                                                                                                                      |
| ------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Lib_AddressManager                    | [0xA4346c8c120DdCE2c5447e68790625F10Bb4d47A](https://goerli.etherscan.io/address/0xA4346c8c120DdCE2c5447e68790625F10Bb4d47A) |
| OVM_CanonicalTransactionChain         | [0x4781674AAe242bbDf6C58b81Cf4F06F1534cd37d](https://goerli.etherscan.io/address/0x4781674AAe242bbDf6C58b81Cf4F06F1534cd37d) |
| OVM_ChainStorageContainer-CTC-batches | [0xd5F2B9f6Ee80065b2Ce18bF1e629c5aC1C98c7F6](https://goerli.etherscan.io/address/0xd5F2B9f6Ee80065b2Ce18bF1e629c5aC1C98c7F6) |
| OVM_ChainStorageContainer-CTC-queue   | [0x3EA657c5aA0E4Bce1D8919dC7f248724d7B0987a](https://goerli.etherscan.io/address/0x3EA657c5aA0E4Bce1D8919dC7f248724d7B0987a) |
| OVM_ChainStorageContainer-SCC-batches | [0x777adA49d40DAC02AE5b4FdC292feDf9066435A3](https://goerli.etherscan.io/address/0x777adA49d40DAC02AE5b4FdC292feDf9066435A3) |
| OVM_ExecutionManager                  | [0x838a74bAdfD28Fd0e32E4A88BddDa502D56ae7F7](https://goerli.etherscan.io/address/0x838a74bAdfD28Fd0e32E4A88BddDa502D56ae7F7) |
| OVM_FraudVerifier                     | [0x916f75037b87Bf4Fe0Dc7719815bd972F0618669](https://goerli.etherscan.io/address/0x916f75037b87Bf4Fe0Dc7719815bd972F0618669) |
| OVM_L1MultiMessageRelayer             | [0x2545fa928d5d278cA75Fd47306e4a89096ff6403](https://goerli.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403) |
| OVM_SafetyChecker                     | [0x71D4ea896C9a2D4a973CC5c7E347B6707691ECa0](https://goerli.etherscan.io/address/0x71D4ea896C9a2D4a973CC5c7E347B6707691ECa0) |
| OVM_StateCommitmentChain              | [0x9bA5E286934F0A29fb2f8421f60d3eE8A853447C](https://goerli.etherscan.io/address/0x9bA5E286934F0A29fb2f8421f60d3eE8A853447C) |
| OVM_StateManagerFactory               | [0x24C7F0a4a2B926613B31c4cDDA4c0f90c0772f2b](https://goerli.etherscan.io/address/0x24C7F0a4a2B926613B31c4cDDA4c0f90c0772f2b) |
| OVM_StateTransitionerFactory          | [0x703303Ce2d92Ef95F17a622E3d538390251165E8](https://goerli.etherscan.io/address/0x703303Ce2d92Ef95F17a622E3d538390251165E8) |
| Proxy__OVM_L1CrossDomainMessenger     | [0xa85716330ff84Ab312D5B43F3BfDcC7E650fd88A](https://goerli.etherscan.io/address/0xa85716330ff84Ab312D5B43F3BfDcC7E650fd88A) |
| Proxy__OVM_L1StandardBridge           | [0x74B6CC2F377fB769cEd6c697bC4C58a9c342E424](https://goerli.etherscan.io/address/0x74B6CC2F377fB769cEd6c697bC4C58a9c342E424) |
| mockOVM_BondManager                   | [0x795F355F75f9B28AEC6cC6A887704191e630065b](https://goerli.etherscan.io/address/0x795F355F75f9B28AEC6cC6A887704191e630065b) |
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger:
 - 0x3B1D4DE5F7Fe8487980Ee7608BE302dC60a9caE9
 - https://goerli.etherscan.io/address/0x3B1D4DE5F7Fe8487980Ee7608BE302dC60a9caE9)
-->
---
