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
|BondManager|[0xE5AE60bD6F8DEe4D0c2BC9268e23B92F1cacC58F](https://goerli.etherscan.io/address/0xE5AE60bD6F8DEe4D0c2BC9268e23B92F1cacC58F)|
|CanonicalTransactionChain|[0x2ebA8c4EfDB39A8Cd8f9eD65c50ec079f7CEBD81](https://goerli.etherscan.io/address/0x2ebA8c4EfDB39A8Cd8f9eD65c50ec079f7CEBD81)|
|ChainStorageContainer-CTC-batches|[0x0821Ff73FD88bb73E90F2Ea459B57430dff731Dd](https://goerli.etherscan.io/address/0x0821Ff73FD88bb73E90F2Ea459B57430dff731Dd)|
|ChainStorageContainer-CTC-queue|[0xf96dc01589969B85e27017F1bC449CB981eED9C8](https://goerli.etherscan.io/address/0xf96dc01589969B85e27017F1bC449CB981eED9C8)|
|ChainStorageContainer-SCC-batches|[0x829863Ce01B475B7d030539d2181d49E7A4b8aD9](https://goerli.etherscan.io/address/0x829863Ce01B475B7d030539d2181d49E7A4b8aD9)|
|Lib_AddressManager|[0x2F7E3cAC91b5148d336BbffB224B4dC79F09f01D](https://goerli.etherscan.io/address/0x2F7E3cAC91b5148d336BbffB224B4dC79F09f01D)|
|Proxy__L1CrossDomainMessenger|[0xEcC89b9EDD804850C4F343A278Be902be11AaF42](https://goerli.etherscan.io/address/0xEcC89b9EDD804850C4F343A278Be902be11AaF42)|
|StateCommitmentChain|[0x1afcA918eff169eE20fF8AB6Be75f3E872eE1C1A](https://goerli.etherscan.io/address/0x1afcA918eff169eE20fF8AB6Be75f3E872eE1C1A)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

L1CrossDomainMessenger: 
 - 0xd32718Fdb54e482C5Aa8eb7007cC898d798B3185
 - https://goerli.etherscan.io/address/0xd32718Fdb54e482C5Aa8eb7007cC898d798B3185)
Proxy__L1StandardBridge: 
 - 0x73298186A143a54c20ae98EEE5a025bD5979De02
 - https://goerli.etherscan.io/address/0x73298186A143a54c20ae98EEE5a025bD5979De02)
-->
---
