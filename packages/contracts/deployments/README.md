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
|BondManager|[0x6618d1A81E7E984018c987AAbDcc35ad3b0aC728](https://kovan.etherscan.io/address/0x6618d1A81E7E984018c987AAbDcc35ad3b0aC728)|
|CanonicalTransactionChain|[0xa4569529548BFae8C874E029D44CD1251F479f69](https://kovan.etherscan.io/address/0xa4569529548BFae8C874E029D44CD1251F479f69)|
|ChainStorageContainer-CTC-batches|[0x0D07F64f132AcD51032975dc997520CfaaB2353e](https://kovan.etherscan.io/address/0x0D07F64f132AcD51032975dc997520CfaaB2353e)|
|ChainStorageContainer-CTC-queue|[0x5590B2800A0c7629eef5fD931c635168e68c0891](https://kovan.etherscan.io/address/0x5590B2800A0c7629eef5fD931c635168e68c0891)|
|ChainStorageContainer-SCC-batches|[0x2BC582f85F5061cae1F1cDD4D8987B881D7709d4](https://kovan.etherscan.io/address/0x2BC582f85F5061cae1F1cDD4D8987B881D7709d4)|
|Lib_AddressManager|[0x3AD1eeD551d26335caD030911C15d008abBe9825](https://kovan.etherscan.io/address/0x3AD1eeD551d26335caD030911C15d008abBe9825)|
|OVM_L1CrossDomainMessenger|[0xBCCe0cCEF373C2342D62C064B1Cc195Eed420905](https://kovan.etherscan.io/address/0xBCCe0cCEF373C2342D62C064B1Cc195Eed420905)|
|Proxy__L1CrossDomainMessenger|[0xD73bc0F558fADfff3ae6845eB418F221F4698fda](https://kovan.etherscan.io/address/0xD73bc0F558fADfff3ae6845eB418F221F4698fda)|
|StateCommitmentChain|[0xF2e0bBf87AF49fdf38a37d183d901C809A4b12Dd](https://kovan.etherscan.io/address/0xF2e0bBf87AF49fdf38a37d183d901C809A4b12Dd)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

Proxy__L1StandardBridge: 
 - 0x54616734b03e0D59963a929ECa1F093Bc4C7b410
 - https://kovan.etherscan.io/address/0x54616734b03e0D59963a929ECa1F093Bc4C7b410)
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
