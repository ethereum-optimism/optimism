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
|OVM_ProxySequencerEntrypoint: | `0x4200000000000000000000000000000000000004`
|OVM_SequencerEntrypoint: | `0x4200000000000000000000000000000000000005`
|Lib_AddressManager: | `0x4200000000000000000000000000000000000008`
|ERC1820Registry: | `0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24`

---
---

## LAYER 1

## MAINNET-V2

Network : __mainnet (chain id: 1)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0xd3EeD86464Ff13B4BFD81a3bB1e753b7ceBA3A39|[Open](https://etherscan.io/address/0xd3EeD86464Ff13B4BFD81a3bB1e753b7ceBA3A39)|
|OVM_CanonicalTransactionChain|0x405B4008Da75C48F4E54AA39607378967Ae62338|[Open](https://etherscan.io/address/0x405B4008Da75C48F4E54AA39607378967Ae62338)|
|OVM_ChainStorageContainer:CTC:batches|0x65E921eE201E4a0881FF84ea462baB744bB2fbf0|[Open](https://etherscan.io/address/0x65E921eE201E4a0881FF84ea462baB744bB2fbf0)|
|OVM_ChainStorageContainer:CTC:queue|0x03004C447722d207B0355529A6d0dA0696BF6ec6|[Open](https://etherscan.io/address/0x03004C447722d207B0355529A6d0dA0696BF6ec6)|
|OVM_ChainStorageContainer:SCC:batches|0x6B7Fce2C4FD1934a2d251F8b0930ac82DdDAD804|[Open](https://etherscan.io/address/0x6B7Fce2C4FD1934a2d251F8b0930ac82DdDAD804)|
|OVM_ExecutionManager|0xEd93C5c21c502bB52b4D77240fA9a5d38472304d|[Open](https://etherscan.io/address/0xEd93C5c21c502bB52b4D77240fA9a5d38472304d)|
|OVM_FraudVerifier|0xF7C64A47A557D2944798801C08771e15455c56c4|[Open](https://etherscan.io/address/0xF7C64A47A557D2944798801C08771e15455c56c4)|
|OVM_L1CrossDomainMessenger|0xeec700E5a793e28B068537c7dd95d632B603440A|[Open](https://etherscan.io/address/0xeec700E5a793e28B068537c7dd95d632B603440A)|
|OVM_L1ETHGateway|0x384bC62a4bb9aE617c8dD0eC351d7780444EFDc0|[Open](https://etherscan.io/address/0x384bC62a4bb9aE617c8dD0eC351d7780444EFDc0)|
|OVM_L1MultiMessageRelayer|0x22adc8A1152B090721E253Ee88CC12a15bcF9222|[Open](https://etherscan.io/address/0x22adc8A1152B090721E253Ee88CC12a15bcF9222)|
|OVM_SafetyChecker|0x4667c625b36Df62e393a9483BCfB2F00cA0708D1|[Open](https://etherscan.io/address/0x4667c625b36Df62e393a9483BCfB2F00cA0708D1)|
|OVM_StateCommitmentChain|0x1D0C46671E0696a4Ba800032D5195d5b0f8c60A3|[Open](https://etherscan.io/address/0x1D0C46671E0696a4Ba800032D5195d5b0f8c60A3)|
|OVM_StateManagerFactory|0xc43AB03567A18CC75CD4B75ABDBEb6DfC2192fF3|[Open](https://etherscan.io/address/0xc43AB03567A18CC75CD4B75ABDBEb6DfC2192fF3)|
|OVM_StateTransitionerFactory|0x8FA5bfeeb7786D2a241527E8aE8cA1d7511A0E10|[Open](https://etherscan.io/address/0x8FA5bfeeb7786D2a241527E8aE8cA1d7511A0E10)|
|Proxy__OVM_L1CrossDomainMessenger|0xD1EC7d40CCd01EB7A305b94cBa8AB6D17f6a9eFE|[Open](https://etherscan.io/address/0xD1EC7d40CCd01EB7A305b94cBa8AB6D17f6a9eFE)|
|Proxy__OVM_L1ETHGateway|0xF20C38fCdDF0C790319Fd7431d17ea0c2bC9959c|[Open](https://etherscan.io/address/0xF20C38fCdDF0C790319Fd7431d17ea0c2bC9959c)|
|mockOVM_BondManager|0x99EDa8472E93Aa28E5470eEDEc6e32081E14DaFC|[Open](https://etherscan.io/address/0x99EDa8472E93Aa28E5470eEDEc6e32081E14DaFC)|
---
## MAINNET-V1

Network : __mainnet (chain id: 1)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0x1De8CFD4C1A486200286073aE91DE6e8099519f1|[Open](https://etherscan.io/address/0x1De8CFD4C1A486200286073aE91DE6e8099519f1)|
|OVM_CanonicalTransactionChain|0xed2701f7135eab0D7ca02e6Ab634AD6CbE159Ffb|[Open](https://etherscan.io/address/0xed2701f7135eab0D7ca02e6Ab634AD6CbE159Ffb)|
|OVM_ChainStorageContainer:CTC:batches|0x7Cb043e523F6B5D492E0d2221e45062d3878599c|[Open](https://etherscan.io/address/0x7Cb043e523F6B5D492E0d2221e45062d3878599c)|
|OVM_ChainStorageContainer:CTC:queue|0x62De49fe8215DFF88b9C1a2ea573E1471fF61f83|[Open](https://etherscan.io/address/0x62De49fe8215DFF88b9C1a2ea573E1471fF61f83)|
|OVM_ChainStorageContainer:SCC:batches|0x7C3e67e5E885556cEF01866CB7bdB5A254D35698|[Open](https://etherscan.io/address/0x7C3e67e5E885556cEF01866CB7bdB5A254D35698)|
|OVM_L1CrossDomainMessenger|0xE8F1bD5e5629F4adac6fd63A39F4b4cB76c5E7B2|[Open](https://etherscan.io/address/0xE8F1bD5e5629F4adac6fd63A39F4b4cB76c5E7B2)|
|OVM_StateCommitmentChain|0x901a629a72A5daF200fc359657f070b34bBfdd18|[Open](https://etherscan.io/address/0x901a629a72A5daF200fc359657f070b34bBfdd18)|
|Proxy__OVM_L1CrossDomainMessenger|0xfBE93ba0a2Df92A8e8D40cE00acCF9248a6Fc812|[Open](https://etherscan.io/address/0xfBE93ba0a2Df92A8e8D40cE00acCF9248a6Fc812)|
|mockOVM_BondManager|0x64cfd73BE445F6Aa4ee9F4f7B1d068008a9DAc06|[Open](https://etherscan.io/address/0x64cfd73BE445F6Aa4ee9F4f7B1d068008a9DAc06)|
---
## KOVAN-V2

Network : __kovan (chain id: 42)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0xFaf27b24ba54C6910C12CFF5C9453C0e8D634e05|[Open](https://kovan.etherscan.io/address/0xFaf27b24ba54C6910C12CFF5C9453C0e8D634e05)|
|OVM_CanonicalTransactionChain|0xeBD8F6ACF629f27AC7dDDD0603df3359a4f063E3|[Open](https://kovan.etherscan.io/address/0xeBD8F6ACF629f27AC7dDDD0603df3359a4f063E3)|
|OVM_ChainStorageContainer:CTC:batches|0x18bA855471f10B74851C0e133db597075Dff128d|[Open](https://kovan.etherscan.io/address/0x18bA855471f10B74851C0e133db597075Dff128d)|
|OVM_ChainStorageContainer:CTC:queue|0xf388A98F640baB14e5Cd343B1c27817811aDd682|[Open](https://kovan.etherscan.io/address/0xf388A98F640baB14e5Cd343B1c27817811aDd682)|
|OVM_ChainStorageContainer:SCC:batches|0xDC1f37ec1eeBF9fe5087c24f889E15AB228FDD22|[Open](https://kovan.etherscan.io/address/0xDC1f37ec1eeBF9fe5087c24f889E15AB228FDD22)|
|OVM_ExecutionManager|0x1e9d3f68422b50d3Fc413cb6a79C4144089cf64A|[Open](https://kovan.etherscan.io/address/0x1e9d3f68422b50d3Fc413cb6a79C4144089cf64A)|
|OVM_FraudVerifier|0x139D12963897129D48C99402Cc481e8C0E8FD0BC|[Open](https://kovan.etherscan.io/address/0x139D12963897129D48C99402Cc481e8C0E8FD0BC)|
|OVM_L1CrossDomainMessenger|0xDBafb4AB19eafE27aF30Dd9C811a1BF4F64b603b|[Open](https://kovan.etherscan.io/address/0xDBafb4AB19eafE27aF30Dd9C811a1BF4F64b603b)|
|OVM_L1ETHGateway|0x0E8917aF9eB7812c7819EF4e80D2217679d11324|[Open](https://kovan.etherscan.io/address/0x0E8917aF9eB7812c7819EF4e80D2217679d11324)|
|OVM_L1MultiMessageRelayer|0xf56d4FAeD6F52c4ce14e44885084dAFc5c440138|[Open](https://kovan.etherscan.io/address/0xf56d4FAeD6F52c4ce14e44885084dAFc5c440138)|
|OVM_SafetyChecker|0xeb91D9059761aFa197deD7b1FB4228F7ea921d3e|[Open](https://kovan.etherscan.io/address/0xeb91D9059761aFa197deD7b1FB4228F7ea921d3e)|
|OVM_StateCommitmentChain|0x41f707A213FB83010586860f81A4BF2F0FEbe56D|[Open](https://kovan.etherscan.io/address/0x41f707A213FB83010586860f81A4BF2F0FEbe56D)|
|OVM_StateManagerFactory|0xda9Da06A7b7D902A746649cA1304665C83a465F8|[Open](https://kovan.etherscan.io/address/0xda9Da06A7b7D902A746649cA1304665C83a465F8)|
|OVM_StateTransitionerFactory|0xE77250c2663d4E81a0Cd7B321f0BB270694A4851|[Open](https://kovan.etherscan.io/address/0xE77250c2663d4E81a0Cd7B321f0BB270694A4851)|
|Proxy__OVM_L1CrossDomainMessenger|0x48062eD9b6488EC41c4CfbF2f568D7773819d8C9|[Open](https://kovan.etherscan.io/address/0x48062eD9b6488EC41c4CfbF2f568D7773819d8C9)|
|Proxy__OVM_L1ETHGateway|0xf3902e50dA095bD2e954AB320E8eafDA6152dFDa|[Open](https://kovan.etherscan.io/address/0xf3902e50dA095bD2e954AB320E8eafDA6152dFDa)|
|mockOVM_BondManager|0x77e244ec49014cFb9c4572453568eCC3AbB70A2d|[Open](https://kovan.etherscan.io/address/0x77e244ec49014cFb9c4572453568eCC3AbB70A2d)|
---
## KOVAN-V1

Network : __kovan (chain id: 42)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0x72e6F5244828C10737cbC9659378B207246D26B2|[Open](https://kovan.etherscan.io/address/0x72e6F5244828C10737cbC9659378B207246D26B2)|
|OVM_CanonicalTransactionChain|0x0ecB7253Aef93dD936E2a9BCEb49bc2fA683Ee65|[Open](https://kovan.etherscan.io/address/0x0ecB7253Aef93dD936E2a9BCEb49bc2fA683Ee65)|
|OVM_ChainStorageContainer:CTC:batches|0x095744753D5353C1FC43EFb1ab81D06f3e2F4630|[Open](https://kovan.etherscan.io/address/0x095744753D5353C1FC43EFb1ab81D06f3e2F4630)|
|OVM_ChainStorageContainer:CTC:queue|0xFCE31EC2Bc82553FaA4A9a6DF36c9b0DFDAdD4B8|[Open](https://kovan.etherscan.io/address/0xFCE31EC2Bc82553FaA4A9a6DF36c9b0DFDAdD4B8)|
|OVM_ChainStorageContainer:SCC:batches|0xcFf7ed66bC3C1eA64c6394FEBb2408D16c6cBC5E|[Open](https://kovan.etherscan.io/address/0xcFf7ed66bC3C1eA64c6394FEBb2408D16c6cBC5E)|
|OVM_L1CrossDomainMessenger|0x19da6C4945f18F5E720054FECC50D6b5E015bd40|[Open](https://kovan.etherscan.io/address/0x19da6C4945f18F5E720054FECC50D6b5E015bd40)|
|OVM_StateCommitmentChain|0x2AAbAf6799822Efc77865401E05CE02897ecf520|[Open](https://kovan.etherscan.io/address/0x2AAbAf6799822Efc77865401E05CE02897ecf520)|
|Proxy__OVM_L1CrossDomainMessenger|0xb89065D5eB05Cac554FDB11fC764C679b4202322|[Open](https://kovan.etherscan.io/address/0xb89065D5eB05Cac554FDB11fC764C679b4202322)|
|mockOVM_BondManager|0x3Ff73EBc1d916a1A976521160ad92dFDF6a06d1f|[Open](https://kovan.etherscan.io/address/0x3Ff73EBc1d916a1A976521160ad92dFDF6a06d1f)|
---
## GOERLI-V2

Network : __goerli (chain id: 5)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0x9933d137bBF050Cf3D7555fE1beC91eF698814e5|[Open](https://goerli.etherscan.io/address/0x9933d137bBF050Cf3D7555fE1beC91eF698814e5)|
|OVM_CanonicalTransactionChain|0x557057458Ba57F03e3191ddA69118DFe42a7295d|[Open](https://goerli.etherscan.io/address/0x557057458Ba57F03e3191ddA69118DFe42a7295d)|
|OVM_ChainStorageContainer:CTC:batches|0x648D625eCa2A2491547d2D702e21070675518E4a|[Open](https://goerli.etherscan.io/address/0x648D625eCa2A2491547d2D702e21070675518E4a)|
|OVM_ChainStorageContainer:CTC:queue|0xe7C69bfEC244EC659871E5685fc17D86eaFB8305|[Open](https://goerli.etherscan.io/address/0xe7C69bfEC244EC659871E5685fc17D86eaFB8305)|
|OVM_ChainStorageContainer:SCC:batches|0x96bD3A792Cc288C51C55A33BC8089026c7009bfd|[Open](https://goerli.etherscan.io/address/0x96bD3A792Cc288C51C55A33BC8089026c7009bfd)|
|OVM_ExecutionManager|0x3212027673655d3047c13139e3233ccd4A78417c|[Open](https://goerli.etherscan.io/address/0x3212027673655d3047c13139e3233ccd4A78417c)|
|OVM_FraudVerifier|0x08BB26333Ed18CcF632e2d68DdC9B5aFfb2EE687|[Open](https://goerli.etherscan.io/address/0x08BB26333Ed18CcF632e2d68DdC9B5aFfb2EE687)|
|OVM_L1CrossDomainMessenger|0x7910D57c49fAE4F7c896A6cd185aB1e6196D8161|[Open](https://goerli.etherscan.io/address/0x7910D57c49fAE4F7c896A6cd185aB1e6196D8161)|
|OVM_L1ETHGateway|0x2C9573A5c0d94075601dB745255645FE5D2e5f7C|[Open](https://goerli.etherscan.io/address/0x2C9573A5c0d94075601dB745255645FE5D2e5f7C)|
|OVM_L1MultiMessageRelayer|0x120b44cC54e9b7E79b3583BE6B797D36DF9fD90a|[Open](https://goerli.etherscan.io/address/0x120b44cC54e9b7E79b3583BE6B797D36DF9fD90a)|
|OVM_SafetyChecker|0x97203a63AC85D811b75575bc5F7Ddc414548B287|[Open](https://goerli.etherscan.io/address/0x97203a63AC85D811b75575bc5F7Ddc414548B287)|
|OVM_StateCommitmentChain|0xc983d52292DCBBEE53a0730C6A3EEb61c6F19129|[Open](https://goerli.etherscan.io/address/0xc983d52292DCBBEE53a0730C6A3EEb61c6F19129)|
|OVM_StateManagerFactory|0x625Ee9D6a8486FDc0c70b1793F37d368f4698014|[Open](https://goerli.etherscan.io/address/0x625Ee9D6a8486FDc0c70b1793F37d368f4698014)|
|OVM_StateTransitionerFactory|0x28f8A0877c2DC85b3Aa269bD772CaCc6e92D7371|[Open](https://goerli.etherscan.io/address/0x28f8A0877c2DC85b3Aa269bD772CaCc6e92D7371)|
|Proxy__OVM_L1CrossDomainMessenger|0x03F6221792451CAD23dF17fF4D702bF93978a9b3|[Open](https://goerli.etherscan.io/address/0x03F6221792451CAD23dF17fF4D702bF93978a9b3)|
|Proxy__OVM_L1ETHGateway|0x499223f87451F2dcC638c506ff7549838A3ee00e|[Open](https://goerli.etherscan.io/address/0x499223f87451F2dcC638c506ff7549838A3ee00e)|
|mockOVM_BondManager|0x1e4f220d5CDD25e2C0E60e0B2f56a7CCC25719C1|[Open](https://goerli.etherscan.io/address/0x1e4f220d5CDD25e2C0E60e0B2f56a7CCC25719C1)|
---
## GOERLI-V1

Network : __goerli (chain id: 5)__

|Contract|Address|Etherscan|
|--|--|--|
|Lib_AddressManager|0x5011A092e66B2c89e2d09dfb9E418B4bCFb24C80|[Open](https://goerli.etherscan.io/address/0x5011A092e66B2c89e2d09dfb9E418B4bCFb24C80)|
|OVM_CanonicalTransactionChain|0x8468e3B58Cc7B34ab07ca5b80CB234e271435120|[Open](https://goerli.etherscan.io/address/0x8468e3B58Cc7B34ab07ca5b80CB234e271435120)|
|OVM_ChainStorageContainer:CTC:batches|0xe0992cB281cfb66cC53A98B7d32B0305d37F723D|[Open](https://goerli.etherscan.io/address/0xe0992cB281cfb66cC53A98B7d32B0305d37F723D)|
|OVM_ChainStorageContainer:CTC:queue|0x85f5bDc9C0269D32154fa1CCdbf697B46AF37273|[Open](https://goerli.etherscan.io/address/0x85f5bDc9C0269D32154fa1CCdbf697B46AF37273)|
|OVM_ChainStorageContainer:SCC:batches|0x2c68C92992c516b7Bdd816cc471938025672fd7a|[Open](https://goerli.etherscan.io/address/0x2c68C92992c516b7Bdd816cc471938025672fd7a)|
|OVM_L1CrossDomainMessenger|0x1c15fcA66a14eB2de5cDCf0BF1f45580b58ca5AC|[Open](https://goerli.etherscan.io/address/0x1c15fcA66a14eB2de5cDCf0BF1f45580b58ca5AC)|
|OVM_StateCommitmentChain|0xAE493AD1fFCD654E6e4b78a66be3c9780a6ca89d|[Open](https://goerli.etherscan.io/address/0xAE493AD1fFCD654E6e4b78a66be3c9780a6ca89d)|
|Proxy__OVM_L1CrossDomainMessenger|0x0f94dA8E27A6116E341c5C807aD32c62EBc90eB6|[Open](https://goerli.etherscan.io/address/0x0f94dA8E27A6116E341c5C807aD32c62EBc90eB6)|
|mockOVM_BondManager|0x1F5AbC065D4B3F3dc127CA8B0042bD4Fcaf79EFC|[Open](https://goerli.etherscan.io/address/0x1F5AbC065D4B3F3dc127CA8B0042bD4Fcaf79EFC)|
---
