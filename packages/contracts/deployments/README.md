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
|OVM_L1MessageSender|0x4200000000000000000000000000000000000001|
|OVM_DeployerWhitelist|0x4200000000000000000000000000000000000002|
|OVM_ECDSAContractAccount|0x4200000000000000000000000000000000000003|
|OVM_SequencerEntrypoint|0x4200000000000000000000000000000000000005|
|OVM_ETH|0x4200000000000000000000000000000000000006|
|OVM_L2CrossDomainMessenger|0x4200000000000000000000000000000000000007|
|Lib_AddressManager|0x4200000000000000000000000000000000000008|
|OVM_ProxyEOA|0x4200000000000000000000000000000000000009|
|OVM_ExecutionManagerWrapper|0x420000000000000000000000000000000000000B|
|OVM_GasPriceOracle|0x420000000000000000000000000000000000000F|
|OVM_SequencerFeeVault|0x4200000000000000000000000000000000000011|
|OVM_L2StandardBridge|0x4200000000000000000000000000000000000010|
|ERC1820Registry|0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24|

---
---

## LAYER 1

## RINKEBY-PRODUCTION

Network : __rinkeby (chain id: 4)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0x93A96D6A5beb1F661cf052722A1424CDDA3e9418](https://rinkeby.etherscan.io/address/0x93A96D6A5beb1F661cf052722A1424CDDA3e9418)|
|OVM_CanonicalTransactionChain|[0xdc8A2730E167bFe5A96E0d713D95D399D070dF60](https://rinkeby.etherscan.io/address/0xdc8A2730E167bFe5A96E0d713D95D399D070dF60)|
|OVM_ChainStorageContainer-CTC-batches|[0xF4A6Bb0744fb75D009AB184184856d5f6edcB6ba](https://rinkeby.etherscan.io/address/0xF4A6Bb0744fb75D009AB184184856d5f6edcB6ba)|
|OVM_ChainStorageContainer-CTC-queue|[0x46FC9c5301A4FB5DaE830Aca7BD98Ef328c96c4a](https://rinkeby.etherscan.io/address/0x46FC9c5301A4FB5DaE830Aca7BD98Ef328c96c4a)|
|OVM_ChainStorageContainer-SCC-batches|[0x8B7D233E9cD4a2f950dd82A4F71D2C833d710b52](https://rinkeby.etherscan.io/address/0x8B7D233E9cD4a2f950dd82A4F71D2C833d710b52)|
|OVM_ExecutionManager|[0xf431c82fA505A6B081A5f80FCD6c018972D60D8B](https://rinkeby.etherscan.io/address/0xf431c82fA505A6B081A5f80FCD6c018972D60D8B)|
|OVM_FraudVerifier|[0xFEFf7EfcbF79dD688A616BCb1F511B1b8cE0068A](https://rinkeby.etherscan.io/address/0xFEFf7EfcbF79dD688A616BCb1F511B1b8cE0068A)|
|OVM_L1MultiMessageRelayer|[0x5881EE5ef1c0BC1d9bB78788e1Bb8737398545D7](https://rinkeby.etherscan.io/address/0x5881EE5ef1c0BC1d9bB78788e1Bb8737398545D7)|
|OVM_SafetyChecker|[0xa10eAe6538C515e82F16D2C95c0936A4452BB117](https://rinkeby.etherscan.io/address/0xa10eAe6538C515e82F16D2C95c0936A4452BB117)|
|OVM_StateCommitmentChain|[0x1ba99640444B81f3928e4F174CFB4FF426B4FFAE](https://rinkeby.etherscan.io/address/0x1ba99640444B81f3928e4F174CFB4FF426B4FFAE)|
|OVM_StateManagerFactory|[0xc4E3E4F9631220f2B1Ada9ee1164E30640c56c94](https://rinkeby.etherscan.io/address/0xc4E3E4F9631220f2B1Ada9ee1164E30640c56c94)|
|OVM_StateTransitionerFactory|[0xAC82f9F03f51c8fFef9Ff0362973e89C0dA4aa40](https://rinkeby.etherscan.io/address/0xAC82f9F03f51c8fFef9Ff0362973e89C0dA4aa40)|
|Proxy__OVM_L1CrossDomainMessenger|[0xF10EEfC14eB5b7885Ea9F7A631a21c7a82cf5D76](https://rinkeby.etherscan.io/address/0xF10EEfC14eB5b7885Ea9F7A631a21c7a82cf5D76)|
|Proxy__OVM_L1StandardBridge|[0xDe085C82536A06b40D20654c2AbA342F2abD7077](https://rinkeby.etherscan.io/address/0xDe085C82536A06b40D20654c2AbA342F2abD7077)|
|mockOVM_BondManager|[0x2Ba9F9a6D6D7F604E9e2ca2Ea5f8C9Fa75E13835](https://rinkeby.etherscan.io/address/0x2Ba9F9a6D6D7F604E9e2ca2Ea5f8C9Fa75E13835)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0x8109f1Af0e8A74e393703Ca5447C5414E1946500
 - https://rinkeby.etherscan.io/address/0x8109f1Af0e8A74e393703Ca5447C5414E1946500)
-->
---
## RINKEBY-INTEGRATION

Network : __rinkeby (chain id: 4)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xd58781Cdb5FC05CB94c579D9a84A0e0F5242b5ad](https://rinkeby.etherscan.io/address/0xd58781Cdb5FC05CB94c579D9a84A0e0F5242b5ad)|
|OVM_CanonicalTransactionChain|[0xb7945b1C99Ed3D5093a2cA4ee6454B8911e4861A](https://rinkeby.etherscan.io/address/0xb7945b1C99Ed3D5093a2cA4ee6454B8911e4861A)|
|OVM_ChainStorageContainer-CTC-batches|[0x1889Adb3678E41b47496c5a7882337039C6ebBe1](https://rinkeby.etherscan.io/address/0x1889Adb3678E41b47496c5a7882337039C6ebBe1)|
|OVM_ChainStorageContainer-CTC-queue|[0xd016AE4Ca2B482fC83817345A32dD60F5E9DFdb8](https://rinkeby.etherscan.io/address/0xd016AE4Ca2B482fC83817345A32dD60F5E9DFdb8)|
|OVM_ChainStorageContainer-SCC-batches|[0x1D8EEc9c2157B6fB0b28201185475d091CD4Cb89](https://rinkeby.etherscan.io/address/0x1D8EEc9c2157B6fB0b28201185475d091CD4Cb89)|
|OVM_ExecutionManager|[0x9970eF0D48bFf67846f487554762A81Cb6D65ADa](https://rinkeby.etherscan.io/address/0x9970eF0D48bFf67846f487554762A81Cb6D65ADa)|
|OVM_FraudVerifier|[0x2384494f19CF08442B37aCD63A46947118C5d5bd](https://rinkeby.etherscan.io/address/0x2384494f19CF08442B37aCD63A46947118C5d5bd)|
|OVM_L1MultiMessageRelayer|[0x5C621BE82C4E9a73d8428AA6fF01ec48FFf48174](https://rinkeby.etherscan.io/address/0x5C621BE82C4E9a73d8428AA6fF01ec48FFf48174)|
|OVM_SafetyChecker|[0xEb6C6071C518e44251aC76E8CcE0A57fCA672675](https://rinkeby.etherscan.io/address/0xEb6C6071C518e44251aC76E8CcE0A57fCA672675)|
|OVM_StateCommitmentChain|[0x59A5662186928742C6F37f25BCf057D387C33408](https://rinkeby.etherscan.io/address/0x59A5662186928742C6F37f25BCf057D387C33408)|
|OVM_StateManagerFactory|[0x8c6652F82E114C8D3FaA7113B1408ae6364f1D11](https://rinkeby.etherscan.io/address/0x8c6652F82E114C8D3FaA7113B1408ae6364f1D11)|
|OVM_StateTransitionerFactory|[0xb6046496DeDAFb0E416c8C816Fa25Ffaf25c309f](https://rinkeby.etherscan.io/address/0xb6046496DeDAFb0E416c8C816Fa25Ffaf25c309f)|
|Proxy__OVM_L1CrossDomainMessenger|[0x0C1E0c73A48e7624DB86bc5234E7E3188cb7b47e](https://rinkeby.etherscan.io/address/0x0C1E0c73A48e7624DB86bc5234E7E3188cb7b47e)|
|Proxy__OVM_L1StandardBridge|[0x95c3b9448A9B5F563e7DC47Ac3e4D6fF0F9Fad93](https://rinkeby.etherscan.io/address/0x95c3b9448A9B5F563e7DC47Ac3e4D6fF0F9Fad93)|
|mockOVM_BondManager|[0xF66591BD3f660b39407AC2A0343b593F651dd0A2](https://rinkeby.etherscan.io/address/0xF66591BD3f660b39407AC2A0343b593F651dd0A2)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0x4B669b500f39B5746D5E5293Bbc2Ac739C430aF9
 - https://rinkeby.etherscan.io/address/0x4B669b500f39B5746D5E5293Bbc2Ac739C430aF9)
-->
---
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
## MAINNET-PRODUCTION

Network : __mainnet (chain id: 1)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0x8376ac6C3f73a25Dd994E0b0669ca7ee0C02F089](https://etherscan.io/address/0x8376ac6C3f73a25Dd994E0b0669ca7ee0C02F089)|
|OVM_CanonicalTransactionChain|[0x4B5D9E5A6B1a514eba15A2f949531DcCd7c272F2](https://etherscan.io/address/0x4B5D9E5A6B1a514eba15A2f949531DcCd7c272F2)|
|OVM_ChainStorageContainer-CTC-batches|[0xA7557b676EA0D9406459409B5ad01c14b5522c46](https://etherscan.io/address/0xA7557b676EA0D9406459409B5ad01c14b5522c46)|
|OVM_ChainStorageContainer-CTC-queue|[0x33938f8E5F2c36e3Ca2B01E878b3322E280d4c50](https://etherscan.io/address/0x33938f8E5F2c36e3Ca2B01E878b3322E280d4c50)|
|OVM_ChainStorageContainer-SCC-batches|[0x318d4dAb7D3793E40139b496c3B89422Ae5372D1](https://etherscan.io/address/0x318d4dAb7D3793E40139b496c3B89422Ae5372D1)|
|OVM_ExecutionManager|[0xa230D4b11F66A3DEEE0bEAf8D04551F236C8B646](https://etherscan.io/address/0xa230D4b11F66A3DEEE0bEAf8D04551F236C8B646)|
|OVM_FraudVerifier|[0x872c65c835deB2CFB3493f2C3dD353633Ae4f4B8](https://etherscan.io/address/0x872c65c835deB2CFB3493f2C3dD353633Ae4f4B8)|
|OVM_L1MultiMessageRelayer|[0xAb2AF3A98D229b7dAeD7305Bb88aD0BA2c42f9cA](https://etherscan.io/address/0xAb2AF3A98D229b7dAeD7305Bb88aD0BA2c42f9cA)|
|OVM_SafetyChecker|[0x85c0Cebfe3b81d64D256b38fDf65DD05887e5884](https://etherscan.io/address/0x85c0Cebfe3b81d64D256b38fDf65DD05887e5884)|
|OVM_StateCommitmentChain|[0x17834b754e2f09946CE48D7B5beB4D7D94D98aB6](https://etherscan.io/address/0x17834b754e2f09946CE48D7B5beB4D7D94D98aB6)|
|OVM_StateManagerFactory|[0x0c4935b421Af8F86698Fb77233e90AbC5f146846](https://etherscan.io/address/0x0c4935b421Af8F86698Fb77233e90AbC5f146846)|
|OVM_StateTransitionerFactory|[0xc6dd73D427Bf784dd1e2f9F64029a79533ffAb40](https://etherscan.io/address/0xc6dd73D427Bf784dd1e2f9F64029a79533ffAb40)|
|Proxy__OVM_L1CrossDomainMessenger|[0x6D4528d192dB72E282265D6092F4B872f9Dff69e](https://etherscan.io/address/0x6D4528d192dB72E282265D6092F4B872f9Dff69e)|
|Proxy__OVM_L1StandardBridge|[0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00](https://etherscan.io/address/0xdc1664458d2f0B6090bEa60A8793A4E66c2F1c00)|
|mockOVM_BondManager|[0xa4F8CD56c14fCEc655CfdDb2ceBd9f1e9329Ec27](https://etherscan.io/address/0xa4F8CD56c14fCEc655CfdDb2ceBd9f1e9329Ec27)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

OVM_L1CrossDomainMessenger: 
 - 0x25109139f8C4F9f7b4E4d5452A067feaE3a537F3
 - https://etherscan.io/address/0x25109139f8C4F9f7b4E4d5452A067feaE3a537F3)
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

OVM_L1CrossDomainMessenger: 
 - 0xbfba066b5cA610Fe70AdCE45FcB622F945891bb0
 - https://etherscan.io/address/0xbfba066b5cA610Fe70AdCE45FcB622F945891bb0)
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

OVM_L1CrossDomainMessenger: 
 - 0x333d2674E2D7e1e7327dc076030ce9615183709C
 - https://kovan.etherscan.io/address/0x333d2674E2D7e1e7327dc076030ce9615183709C)
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
