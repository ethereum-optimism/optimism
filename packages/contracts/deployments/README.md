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

## MAINNET

Network : __mainnet (chain id: 1)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xd3EeD86464Ff13B4BFD81a3bB1e753b7ceBA3A39](https://etherscan.io/address/0xd3EeD86464Ff13B4BFD81a3bB1e753b7ceBA3A39)|
|OVM_CanonicalTransactionChain|[0x405B4008Da75C48F4E54AA39607378967Ae62338](https://etherscan.io/address/0x405B4008Da75C48F4E54AA39607378967Ae62338)|
|OVM_ChainStorageContainer-CTC-batches|[0x65E921eE201E4a0881FF84ea462baB744bB2fbf0](https://etherscan.io/address/0x65E921eE201E4a0881FF84ea462baB744bB2fbf0)|
|OVM_ChainStorageContainer-CTC-queue|[0x03004C447722d207B0355529A6d0dA0696BF6ec6](https://etherscan.io/address/0x03004C447722d207B0355529A6d0dA0696BF6ec6)|
|OVM_ChainStorageContainer-SCC-batches|[0x6B7Fce2C4FD1934a2d251F8b0930ac82DdDAD804](https://etherscan.io/address/0x6B7Fce2C4FD1934a2d251F8b0930ac82DdDAD804)|
|OVM_ExecutionManager|[0xEd93C5c21c502bB52b4D77240fA9a5d38472304d](https://etherscan.io/address/0xEd93C5c21c502bB52b4D77240fA9a5d38472304d)|
|OVM_FraudVerifier|[0xF7C64A47A557D2944798801C08771e15455c56c4](https://etherscan.io/address/0xF7C64A47A557D2944798801C08771e15455c56c4)|
|OVM_L1CrossDomainMessenger|[0xeec700E5a793e28B068537c7dd95d632B603440A](https://etherscan.io/address/0xeec700E5a793e28B068537c7dd95d632B603440A)|
|OVM_L1ETHGateway|[0x384bC62a4bb9aE617c8dD0eC351d7780444EFDc0](https://etherscan.io/address/0x384bC62a4bb9aE617c8dD0eC351d7780444EFDc0)|
|OVM_L1MultiMessageRelayer|[0x22adc8A1152B090721E253Ee88CC12a15bcF9222](https://etherscan.io/address/0x22adc8A1152B090721E253Ee88CC12a15bcF9222)|
|OVM_SafetyChecker|[0x4667c625b36Df62e393a9483BCfB2F00cA0708D1](https://etherscan.io/address/0x4667c625b36Df62e393a9483BCfB2F00cA0708D1)|
|OVM_StateCommitmentChain|[0x1D0C46671E0696a4Ba800032D5195d5b0f8c60A3](https://etherscan.io/address/0x1D0C46671E0696a4Ba800032D5195d5b0f8c60A3)|
|OVM_StateManagerFactory|[0xc43AB03567A18CC75CD4B75ABDBEb6DfC2192fF3](https://etherscan.io/address/0xc43AB03567A18CC75CD4B75ABDBEb6DfC2192fF3)|
|OVM_StateTransitionerFactory|[0x8FA5bfeeb7786D2a241527E8aE8cA1d7511A0E10](https://etherscan.io/address/0x8FA5bfeeb7786D2a241527E8aE8cA1d7511A0E10)|
|Proxy__OVM_L1CrossDomainMessenger|[0xD1EC7d40CCd01EB7A305b94cBa8AB6D17f6a9eFE](https://etherscan.io/address/0xD1EC7d40CCd01EB7A305b94cBa8AB6D17f6a9eFE)|
|Proxy__OVM_L1ETHGateway|[0xF20C38fCdDF0C790319Fd7431d17ea0c2bC9959c](https://etherscan.io/address/0xF20C38fCdDF0C790319Fd7431d17ea0c2bC9959c)|
|mockOVM_BondManager|[0x99EDa8472E93Aa28E5470eEDEc6e32081E14DaFC](https://etherscan.io/address/0x99EDa8472E93Aa28E5470eEDEc6e32081E14DaFC)|
---

## KOVAN

Network : __kovan (chain id: 42)__

|Contract|Address|
|--|--|
|Lib_AddressManager|[0xFaf27b24ba54C6910C12CFF5C9453C0e8D634e05](https://kovan.etherscan.io/address/0xFaf27b24ba54C6910C12CFF5C9453C0e8D634e05)|
|OVM_CanonicalTransactionChain|[0xeBD8F6ACF629f27AC7dDDD0603df3359a4f063E3](https://kovan.etherscan.io/address/0xeBD8F6ACF629f27AC7dDDD0603df3359a4f063E3)|
|OVM_ChainStorageContainer-CTC-batches|[0x18bA855471f10B74851C0e133db597075Dff128d](https://kovan.etherscan.io/address/0x18bA855471f10B74851C0e133db597075Dff128d)|
|OVM_ChainStorageContainer-CTC-queue|[0xf388A98F640baB14e5Cd343B1c27817811aDd682](https://kovan.etherscan.io/address/0xf388A98F640baB14e5Cd343B1c27817811aDd682)|
|OVM_ChainStorageContainer-SCC-batches|[0xDC1f37ec1eeBF9fe5087c24f889E15AB228FDD22](https://kovan.etherscan.io/address/0xDC1f37ec1eeBF9fe5087c24f889E15AB228FDD22)|
|OVM_ExecutionManager|[0x1e9d3f68422b50d3Fc413cb6a79C4144089cf64A](https://kovan.etherscan.io/address/0x1e9d3f68422b50d3Fc413cb6a79C4144089cf64A)|
|OVM_FraudVerifier|[0x139D12963897129D48C99402Cc481e8C0E8FD0BC](https://kovan.etherscan.io/address/0x139D12963897129D48C99402Cc481e8C0E8FD0BC)|
|OVM_L1CrossDomainMessenger|[0xDBafb4AB19eafE27aF30Dd9C811a1BF4F64b603b](https://kovan.etherscan.io/address/0xDBafb4AB19eafE27aF30Dd9C811a1BF4F64b603b)|
|OVM_L1ETHGateway|[0x0E8917aF9eB7812c7819EF4e80D2217679d11324](https://kovan.etherscan.io/address/0x0E8917aF9eB7812c7819EF4e80D2217679d11324)|
|OVM_L1MultiMessageRelayer|[0xf56d4FAeD6F52c4ce14e44885084dAFc5c440138](https://kovan.etherscan.io/address/0xf56d4FAeD6F52c4ce14e44885084dAFc5c440138)|
|OVM_SafetyChecker|[0xeb91D9059761aFa197deD7b1FB4228F7ea921d3e](https://kovan.etherscan.io/address/0xeb91D9059761aFa197deD7b1FB4228F7ea921d3e)|
|OVM_StateCommitmentChain|[0x41f707A213FB83010586860f81A4BF2F0FEbe56D](https://kovan.etherscan.io/address/0x41f707A213FB83010586860f81A4BF2F0FEbe56D)|
|OVM_StateManagerFactory|[0xda9Da06A7b7D902A746649cA1304665C83a465F8](https://kovan.etherscan.io/address/0xda9Da06A7b7D902A746649cA1304665C83a465F8)|
|OVM_StateTransitionerFactory|[0xE77250c2663d4E81a0Cd7B321f0BB270694A4851](https://kovan.etherscan.io/address/0xE77250c2663d4E81a0Cd7B321f0BB270694A4851)|
|Proxy__OVM_L1CrossDomainMessenger|[0x48062eD9b6488EC41c4CfbF2f568D7773819d8C9](https://kovan.etherscan.io/address/0x48062eD9b6488EC41c4CfbF2f568D7773819d8C9)|
|Proxy__OVM_L1ETHGateway|[0xf3902e50dA095bD2e954AB320E8eafDA6152dFDa](https://kovan.etherscan.io/address/0xf3902e50dA095bD2e954AB320E8eafDA6152dFDa)|
|mockOVM_BondManager|[0x77e244ec49014cFb9c4572453568eCC3AbB70A2d](https://kovan.etherscan.io/address/0x77e244ec49014cFb9c4572453568eCC3AbB70A2d)|
