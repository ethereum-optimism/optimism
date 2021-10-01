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
|OVM_L2ToL1MessagePasser|0x4200000000000000000000000000000000000000|
|OVM_DeployerWhitelist|0x4200000000000000000000000000000000000002|
|L2CrossDomainMessenger|0x4200000000000000000000000000000000000007|
|OVM_GasPriceOracle|0x420000000000000000000000000000000000000F|
|L2StandardBridge|0x4200000000000000000000000000000000000010|
|OVM_SequencerFeeVault|0x4200000000000000000000000000000000000011|
|L2StandardTokenFactory|0x4200000000000000000000000000000000000012|
|OVM_L1BlockNumber|0x4200000000000000000000000000000000000013|
|OVM_ETH|0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000|
|WETH9|0x4200000000000000000000000000000000000006|

---
---

## LAYER 1

## OPTIMISTIC-KOVAN

Network : __undefined (chain id: 69)__

|Contract|Address|
|--|--|
|OVM_GasPriceOracle|[0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76](https://undefined.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)|
|OVM_L2StandardTokenFactory|[undefined](https://undefined.etherscan.io/address/undefined)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

-->
---
## MAINNET

Network : __mainnet (chain id: 1)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xdE1FCfB0851916CA5101820A69b13a4E276bd81F](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)|
|OVM_CanonicalTransactionChain|[0x4BF681894abEc828B212C906082B444Ceb2f6cf6](https://etherscan.io/address/0x4BF681894abEc828B212C906082B444Ceb2f6cf6)|
|OVM_ChainStorageContainer-CTC-batches|[0x3EA1a3839D8ca9a7ff3c567a9F36f4C4DbECc3eE](https://etherscan.io/address/0x3EA1a3839D8ca9a7ff3c567a9F36f4C4DbECc3eE)|
|OVM_ChainStorageContainer-CTC-queue|[0xA0b912b3Ea71A04065Ff82d3936D518ED6E38039](https://etherscan.io/address/0xA0b912b3Ea71A04065Ff82d3936D518ED6E38039)|
|OVM_ChainStorageContainer-SCC-batches|[0x77eBfdFcC906DDcDa0C42B866f26A8D5A2bb0572](https://etherscan.io/address/0x77eBfdFcC906DDcDa0C42B866f26A8D5A2bb0572)|
|OVM_ExecutionManager|[0x2745C24822f542BbfFB41c6cB20EdF766b5619f5](https://etherscan.io/address/0x2745C24822f542BbfFB41c6cB20EdF766b5619f5)|
|OVM_FraudVerifier|[0x042065416C5c665dc196076745326Af3Cd840D15](https://etherscan.io/address/0x042065416C5c665dc196076745326Af3Cd840D15)|
|OVM_L1CrossDomainMessenger|[0xbfba066b5cA610Fe70AdCE45FcB622F945891bb0](https://etherscan.io/address/0xbfba066b5cA610Fe70AdCE45FcB622F945891bb0)|
|OVM_L1MultiMessageRelayer|[0xF26391FBB1f77481f80a7d646AC08ba3817eA891](https://etherscan.io/address/0xF26391FBB1f77481f80a7d646AC08ba3817eA891)|
|OVM_SafetyChecker|[0xfe1F9Cf28ecDb12110aa8086e6FD343EA06035cC](https://etherscan.io/address/0xfe1F9Cf28ecDb12110aa8086e6FD343EA06035cC)|
|OVM_StateCommitmentChain|[0xE969C2724d2448F1d1A6189d3e2aA1F37d5998c1](https://etherscan.io/address/0xE969C2724d2448F1d1A6189d3e2aA1F37d5998c1)|
|OVM_StateManagerFactory|[0xd0e3e318154716BD9d007E1E6B021Eab246ff98d](https://etherscan.io/address/0xd0e3e318154716BD9d007E1E6B021Eab246ff98d)|
|OVM_StateTransitionerFactory|[0x38A6ed6fd76035684caDef38cF49a2FffA782B67](https://etherscan.io/address/0x38A6ed6fd76035684caDef38cF49a2FffA782B67)|
|Proxy__OVM_L1CrossDomainMessenger|[0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)|
|Proxy__OVM_L1StandardBridge|[0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)|
|mockOVM_BondManager|[0xCd76de5C57004d47d0216ec7dAbd3c72D8c49057](https://etherscan.io/address/0xCd76de5C57004d47d0216ec7dAbd3c72D8c49057)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

-->
---
## KOVAN

Network : __kovan (chain id: 42)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a](https://kovan.etherscan.io/address/0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a)|
|OVM_CanonicalTransactionChain|[0xe28c499EB8c36C0C18d1bdCdC47a51585698cb93](https://kovan.etherscan.io/address/0xe28c499EB8c36C0C18d1bdCdC47a51585698cb93)|
|OVM_ChainStorageContainer-CTC-batches|[0xF95D79298FD12e5ED778CCf717aA30f638b060E1](https://kovan.etherscan.io/address/0xF95D79298FD12e5ED778CCf717aA30f638b060E1)|
|OVM_ChainStorageContainer-CTC-queue|[0x2BE00E5F043a0f62c3e4d775F3235E28A0239395](https://kovan.etherscan.io/address/0x2BE00E5F043a0f62c3e4d775F3235E28A0239395)|
|OVM_ChainStorageContainer-SCC-batches|[0x50DA41A2A185fb917aecEFfa1CB4534dC5C264b4](https://kovan.etherscan.io/address/0x50DA41A2A185fb917aecEFfa1CB4534dC5C264b4)|
|OVM_ExecutionManager|[0xC68795aC9d96374eaE746DAcC1334ba54798e17D](https://kovan.etherscan.io/address/0xC68795aC9d96374eaE746DAcC1334ba54798e17D)|
|OVM_FraudVerifier|[0xaeEd60e029Eb435f960d78C355786060589738B3](https://kovan.etherscan.io/address/0xaeEd60e029Eb435f960d78C355786060589738B3)|
|OVM_L1CrossDomainMessenger|[0x333d2674E2D7e1e7327dc076030ce9615183709C](https://kovan.etherscan.io/address/0x333d2674E2D7e1e7327dc076030ce9615183709C)|
|OVM_L1MultiMessageRelayer|[0x5818840763Ee28ff0A3E3e8CB9eDeDd07Fb1Cd3f](https://kovan.etherscan.io/address/0x5818840763Ee28ff0A3E3e8CB9eDeDd07Fb1Cd3f)|
|OVM_SafetyChecker|[0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a](https://kovan.etherscan.io/address/0xf0FaB0ce35a6d3F82b0B42f09A2734065908dB6a)|
|OVM_StateCommitmentChain|[0xa2487713665AC596b0b3E4881417f276834473d2](https://kovan.etherscan.io/address/0xa2487713665AC596b0b3E4881417f276834473d2)|
|OVM_StateManagerFactory|[0xBcca22E9F5579193E27dD39aD821A03778C44EFA](https://kovan.etherscan.io/address/0xBcca22E9F5579193E27dD39aD821A03778C44EFA)|
|OVM_StateTransitionerFactory|[0xFD7B9268e790837d393Fd371Ddeb42FE5EC45B54](https://kovan.etherscan.io/address/0xFD7B9268e790837d393Fd371Ddeb42FE5EC45B54)|
|Proxy__OVM_L1CrossDomainMessenger|[0x4361d0F75A0186C05f971c566dC6bEa5957483fD](https://kovan.etherscan.io/address/0x4361d0F75A0186C05f971c566dC6bEa5957483fD)|
|Proxy__OVM_L1StandardBridge|[0x22F24361D548e5FaAfb36d1437839f080363982B](https://kovan.etherscan.io/address/0x22F24361D548e5FaAfb36d1437839f080363982B)|
|mockOVM_BondManager|[0xD6143943447DFf503d948Fba3D8af3d4Df28f45c](https://kovan.etherscan.io/address/0xD6143943447DFf503d948Fba3D8af3d4Df28f45c)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

-->
---
## GOERLI

Network : __goerli (chain id: 5)__

|Contract|Address|
|--|--|
|BondManager|[0x9a55980339eCEd11106D6b8F2b6F981a63F1Ed5a](https://goerli.etherscan.io/address/0x9a55980339eCEd11106D6b8F2b6F981a63F1Ed5a)|
|CanonicalTransactionChain|[0xD58ee2Fc654042E7a325988F1e169D5c1dB7c891](https://goerli.etherscan.io/address/0xD58ee2Fc654042E7a325988F1e169D5c1dB7c891)|
|ChainStorageContainer-CTC-batches|[0xB8691D69Fe6fEd985BC74AF0d47dd3FBf90089C6](https://goerli.etherscan.io/address/0xB8691D69Fe6fEd985BC74AF0d47dd3FBf90089C6)|
|ChainStorageContainer-CTC-queue|[0x6bB7983D6913aB3984C6096fCa60B9806007851c](https://goerli.etherscan.io/address/0x6bB7983D6913aB3984C6096fCa60B9806007851c)|
|ChainStorageContainer-SCC-batches|[0x87DC0598F2bA9781fB3b016cDf68292C814981D2](https://goerli.etherscan.io/address/0x87DC0598F2bA9781fB3b016cDf68292C814981D2)|
|Lib_AddressManager|[0xf8FFc434b18D59Cf65524aDdbBdE65c5ce6D2f6F](https://goerli.etherscan.io/address/0xf8FFc434b18D59Cf65524aDdbBdE65c5ce6D2f6F)|
|Proxy__L1CrossDomainMessenger|[0xE4e015EC8EfAC0d505d8a656C5F93c7FA30c6C5e](https://goerli.etherscan.io/address/0xE4e015EC8EfAC0d505d8a656C5F93c7FA30c6C5e)|
|StateCommitmentChain|[0x4b0e2936f555bC0A34BF6a694A47345fd1e8b8BA](https://goerli.etherscan.io/address/0x4b0e2936f555bC0A34BF6a694A47345fd1e8b8BA)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

L1CrossDomainMessenger: 
 - 0xc879152CABddefD9DaF99293Ea971c64a5bBCa1C
 - https://goerli.etherscan.io/address/0xc879152CABddefD9DaF99293Ea971c64a5bBCa1C)
Proxy__L1StandardBridge: 
 - 0xBB5fd80abeAd143B59D1B92a75be8bb0F44D8ECc
 - https://goerli.etherscan.io/address/0xBB5fd80abeAd143B59D1B92a75be8bb0F44D8ECc)
-->
---
