# Optimistic Ethereum Deployments
  ## LAYER 2

  ### Chain IDs
  - Mainnet: 10
  - Kovan: 69
  - Goerli: 420

  ### Pre-deployed Contracts

  **NOTE**: Pre-deployed contract addresses are the same on every Optimistic Ethereum network.

  | Contract | Address |
  | -------- | ------- |
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

  ## LAYER 1
  ## MAINNET-TRIAL

Network : __mainnet-trial (chain id: 42069)__

| Contract | Address |
| -------- | ------- |
|AddressDictator|[0x020756FD65Ac40690b33B4ef3019d91db16c769e](https://mainnet-trial.etherscan.io/address/0x020756FD65Ac40690b33B4ef3019d91db16c769e)|
|BondManager|[0xe0bd909A7AA766427277869fB8e531b467Aa29d0](https://mainnet-trial.etherscan.io/address/0xe0bd909A7AA766427277869fB8e531b467Aa29d0)|
|CanonicalTransactionChain|[0xfF5723c4Ef6EA25aB09A0a11CEF64cf23ef8c417](https://mainnet-trial.etherscan.io/address/0xfF5723c4Ef6EA25aB09A0a11CEF64cf23ef8c417)|
|ChainStorageContainer-CTC-batches|[0xe6F1852f733BbF417eAD2D7B30686b66EA586629](https://mainnet-trial.etherscan.io/address/0xe6F1852f733BbF417eAD2D7B30686b66EA586629)|
|ChainStorageContainer-SCC-batches|[0xF617f6DAB862736686fD659d7589e2927b99983A](https://mainnet-trial.etherscan.io/address/0xF617f6DAB862736686fD659d7589e2927b99983A)|
|ChugSplashDictator|[0xa71cbD7A95610e3aa1AC7ba61ceaf67C17709990](https://mainnet-trial.etherscan.io/address/0xa71cbD7A95610e3aa1AC7ba61ceaf67C17709990)|
|Lib_AddressManager|[0xdE1FCfB0851916CA5101820A69b13a4E276bd81F](https://mainnet-trial.etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)|
|Proxy__OVM_L1CrossDomainMessenger|[0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1](https://mainnet-trial.etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)|
|Proxy__OVM_L1StandardBridge|[0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1](https://mainnet-trial.etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)|
|StateCommitmentChain|[0x982f743B1F164a07768fc02C4CEb7F576F3cb101](https://mainnet-trial.etherscan.io/address/0x982f743B1F164a07768fc02C4CEb7F576F3cb101)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

L1StandardBridge_for_verification_only: 
 - 0x0b3c3d11cFd47327d2B0d2F01b39b535B59eE051
 - https://mainnet-trial.etherscan.io/address/0x0b3c3d11cFd47327d2B0d2F01b39b535B59eE051)
OVM_L1CrossDomainMessenger: 
 - 0xfa416acA8988CAF7997824e98840860652d9Bb6A
 - https://mainnet-trial.etherscan.io/address/0xfa416acA8988CAF7997824e98840860652d9Bb6A)
-->
## MAINNET

Network : __mainnet (chain id: 1)__

| Contract | Address |
| -------- | ------- |
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
## KOVAN

Network : __kovan (chain id: 42)__

| Contract | Address |
| -------- | ------- |
|AddressDictator|[0x8676275c08626263c60282A26550464DFa19ABd6](https://kovan.etherscan.io/address/0x8676275c08626263c60282A26550464DFa19ABd6)|
|BondManager|[0xc5a603d273E28185c18Ba4d26A0024B2d2F42740](https://kovan.etherscan.io/address/0xc5a603d273E28185c18Ba4d26A0024B2d2F42740)|
|CanonicalTransactionChain|[0xf7B88A133202d41Fe5E2Ab22e6309a1A4D50AF74](https://kovan.etherscan.io/address/0xf7B88A133202d41Fe5E2Ab22e6309a1A4D50AF74)|
|ChainStorageContainer-CTC-batches|[0x1d6d23989ba6a6e915F0e35BBc574E914d4ed092](https://kovan.etherscan.io/address/0x1d6d23989ba6a6e915F0e35BBc574E914d4ed092)|
|ChainStorageContainer-SCC-batches|[0x122208Aa20237FB4c655a9eF02685F7255DF33E8](https://kovan.etherscan.io/address/0x122208Aa20237FB4c655a9eF02685F7255DF33E8)|
|ChugSplashDictator|[0x23d87F2792C2ca58E5C1b7BD831B0fbDDEEe0ED9](https://kovan.etherscan.io/address/0x23d87F2792C2ca58E5C1b7BD831B0fbDDEEe0ED9)|
|Lib_AddressManager|[0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a](https://kovan.etherscan.io/address/0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a)|
|Proxy__OVM_L1CrossDomainMessenger|[0x4361d0F75A0186C05f971c566dC6bEa5957483fD](https://kovan.etherscan.io/address/0x4361d0F75A0186C05f971c566dC6bEa5957483fD)|
|Proxy__OVM_L1StandardBridge|[0x22F24361D548e5FaAfb36d1437839f080363982B](https://kovan.etherscan.io/address/0x22F24361D548e5FaAfb36d1437839f080363982B)|
|StateCommitmentChain|[0xD7754711773489F31A0602635f3F167826ce53C5](https://kovan.etherscan.io/address/0xD7754711773489F31A0602635f3F167826ce53C5)|
<!--
Implementation addresses. DO NOT use these addresses directly.
Use their proxied counterparts seen above.

L1StandardBridge_for_verification_only: 
 - 0x51bB1dc7Ebb531539f6F8349D4177255A9994d1C
 - https://kovan.etherscan.io/address/0x51bB1dc7Ebb531539f6F8349D4177255A9994d1C)
OVM_L1CrossDomainMessenger: 
 - 0xaF91349fdf3B206E079A8FcaB7b8dFaFB96A654D
 - https://kovan.etherscan.io/address/0xaF91349fdf3B206E079A8FcaB7b8dFaFB96A654D)
-->
