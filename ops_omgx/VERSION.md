# Integration

## V1

Accounts:

```javascript
{
  "Deployer": "0x2A2D5e9D1A0f3485f3D5c3fd983028E0f226FeD6",
  "Sequencer": "0xE50faB5E5F46BB3E3e412d6DFbA73491a2D97695",
  "Proposer": "0xfA20335C1Dbb08F67B0362CC07F707187CF378f7",
  "Relayer": "0x3C8b7FdbF1e5B2519B00A8c9317C4BA51d6a4f9d",
  "FastRelayer": "0xFA8077f292976ecB6B407c92C048983bDfDff428"
}
```

Regenesis:

```javascript
{
  "AddressManager": "0x227eE01C1daeF105a11F9440e33eE813Ff27d40d",
  "OVM_CanonicalTransactionChain": "0x1B17521841Cc3Bb339762a67cD7F519e933BBBa0",
  "OVM_ChainStorageContainer:CTC:batches": "0x039582CcCF07bDb32534f719C97BF77218f17589",
  "OVM_ChainStorageContainer:CTC:queue": "0xD362FcEeeBF3c5FFe9bC58E86Ba2B2deD97eB547",
  "OVM_ChainStorageContainer:SCC:batches": "0x693b253D1B652510C65042895927cF74edA1300c",
  "OVM_ExecutionManager": "0x1469BC949E397B31604B320bd27673cec0df8217",
  "OVM_FraudVerifier": "0xC28ea274bC685d3eAb33Ae9a7c42049Df0FBe1A8",
  "OVM_L1CrossDomainMessenger": "0x16e1e20d16bE53b1f10bDf800DBA2Da163B5Fd19",
  "OVM_L1CrossDomainMessengerFast": "0x24663c1d9e4030f0e9B690a5dcCC665e28DB89fA",
  "OVM_L1ETHGateway": "0x7A82Fe30F687175CF69C445469dCbE399E8a74f2",
  "OVM_L1MultiMessageRelayer": "0x853f93F3737E138c50eA2A650cdC95354D5580e2",
  "OVM_SafetyChecker": "0xe50f2F7Eb86aa582DfA393d6A23651D2EDA4247c",
  "OVM_StateCommitmentChain": "0x314480e5C9568453E1f43C31f720d9655Edb9236",
  "OVM_StateManagerFactory": "0x910030a7CCEbe6Fd403F6Ec5c543f72Ed07623fD",
  "OVM_StateTransitionerFactory": "0x02902447BBDf6eBF7007a548EDAFE638B6c67Dd2",
  "Proxy__OVM_L1CrossDomainMessenger": "0x3122b5DC8d2d9B2Ca2e0582D8a1c111E1217CBB7",
  "Proxy__OVM_L1ETHGateway": "0x481795417E6E5B4b8803193B210AF67D081B028E",
  "OVM_BondManager": "0x5399E74Cd94EC695e2174d3867f9FEE476971c72",
  "OVM_Sequencer": "0xE50faB5E5F46BB3E3e412d6DFbA73491a2D97695",
  "Deployer": "0x2A2D5e9D1A0f3485f3D5c3fd983028E0f226FeD6"
}
```

Images:

```javascript
{
  "deployer": "omgx/deployer-rinkeby:integration-v1",
  "data_transport_layer": "omgx/data-transport-layer:integration-v1",
  "geth_l2": "omgx/l2geth:integration-v1",
  "batch_submitter": "omgx/batch-submitter:integration-v1",
  "message_relayer": "omgx/message-relayer:integration-v1",
  "message_relayer_fast": "omgx/message-relayer-fast:integration-v1"
}
```

## V2

Accounts:

```javascript
{
  "Deployer": "0x2A2D5e9D1A0f3485f3D5c3fd983028E0f226FeD6",
  "Sequencer": "0xE50faB5E5F46BB3E3e412d6DFbA73491a2D97695",
  "Proposer": "0xfA20335C1Dbb08F67B0362CC07F707187CF378f7",
  "Relayer": "0x3C8b7FdbF1e5B2519B00A8c9317C4BA51d6a4f9d",
  "FastRelayer": "0xFA8077f292976ecB6B407c92C048983bDfDff428"
}
```

Regenesis:

```javascript
{
 "AddressManager": "0xd58781Cdb5FC05CB94c579D9a84A0e0F5242b5ad",
  "OVM_CanonicalTransactionChain": "0xb7945b1C99Ed3D5093a2cA4ee6454B8911e4861A",
  "OVM_ChainStorageContainer-CTC-batches": "0x1889Adb3678E41b47496c5a7882337039C6ebBe1",
  "OVM_ChainStorageContainer-CTC-queue": "0xd016AE4Ca2B482fC83817345A32dD60F5E9DFdb8",
  "OVM_ChainStorageContainer-SCC-batches": "0x1D8EEc9c2157B6fB0b28201185475d091CD4Cb89",
  "OVM_ExecutionManager": "0x9970eF0D48bFf67846f487554762A81Cb6D65ADa",
  "OVM_FraudVerifier": "0x2384494f19CF08442B37aCD63A46947118C5d5bd",
  "OVM_L1CrossDomainMessenger": "0x4B669b500f39B5746D5E5293Bbc2Ac739C430aF9",
  "OVM_L1CrossDomainMessengerFast": "0x704b3410533EEe40EcE4242Cf0d480DBb0225896",
  "OVM_L1MultiMessageRelayer": "0x5C621BE82C4E9a73d8428AA6fF01ec48FFf48174",
  "OVM_SafetyChecker": "0xEb6C6071C518e44251aC76E8CcE0A57fCA672675",
  "OVM_StateCommitmentChain": "0x59A5662186928742C6F37f25BCf057D387C33408",
  "OVM_StateManagerFactory": "0x8c6652F82E114C8D3FaA7113B1408ae6364f1D11",
  "OVM_StateTransitionerFactory": "0xb6046496DeDAFb0E416c8C816Fa25Ffaf25c309f",
  "Proxy__OVM_L1CrossDomainMessenger": "0x0C1E0c73A48e7624DB86bc5234E7E3188cb7b47e",
  "Proxy__OVM_L1StandardBridge": "0x95c3b9448A9B5F563e7DC47Ac3e4D6fF0F9Fad93",
  "OVM_BondManager": "0xF66591BD3f660b39407AC2A0343b593F651dd0A2",
  "OVM_Sequencer": "0xE50faB5E5F46BB3E3e412d6DFbA73491a2D97695",
  "Deployer": "0x2A2D5e9D1A0f3485f3D5c3fd983028E0f226FeD6"
}
```

Images:

```javascript
{
  "deployer": "omgx/deployer-rinkeby:integration-v2",
  "data_transport_layer": "omgx/data-transport-layer:integration-v2",
  "geth_l2": "omgx/l2geth:integration-v2",
  "batch_submitter": "omgx/batch-submitter:integration-v2",
  "message_relayer": "omgx/message-relayer:integration-v2",
  "message_relayer_fast": "omgx/message-relayer-fast:integration-v2"
}
```

## 

# Production

## V1

Accounts:

```javascript
{
  "Deployer": "0x122816e7A7AeB40601d0aC0DCAA8402F7aa4cDfA",
  "Sequencer": "0xE48E5b731FAAb955d147FA954cba19d93Dc03529",
  "Proposer": "0x7f3cDbe9906Fd57373e8d18AaA159Fc713f379b0",
  "Relayer": "0x494Ae1fCd178e0DBA5a3B32D9324C90e47D88AA8",
  "FastRelayer": "0x81922840527936c3453c99a81dBd4b13d7363722"
}
```

Regenesis:

```javascript
{
  "AddressManager": "0x93A96D6A5beb1F661cf052722A1424CDDA3e9418",
  "OVM_CanonicalTransactionChain": "0xdc8A2730E167bFe5A96E0d713D95D399D070dF60",
  "OVM_ChainStorageContainer-CTC-batches": "0xF4A6Bb0744fb75D009AB184184856d5f6edcB6ba",
  "OVM_ChainStorageContainer-CTC-queue": "0x46FC9c5301A4FB5DaE830Aca7BD98Ef328c96c4a",
  "OVM_ChainStorageContainer-SCC-batches": "0x8B7D233E9cD4a2f950dd82A4F71D2C833d710b52",
  "OVM_ExecutionManager": "0xf431c82fA505A6B081A5f80FCD6c018972D60D8B",
  "OVM_FraudVerifier": "0xFEFf7EfcbF79dD688A616BCb1F511B1b8cE0068A",
  "OVM_L1CrossDomainMessenger": "0x8109f1Af0e8A74e393703Ca5447C5414E1946500",
  "OVM_L1CrossDomainMessengerFast": "0x4238A43A1B03a5284438342AeB742a81894DAbac",
  "OVM_L1MultiMessageRelayer": "0x5881EE5ef1c0BC1d9bB78788e1Bb8737398545D7",
  "OVM_SafetyChecker": "0xa10eAe6538C515e82F16D2C95c0936A4452BB117",
  "OVM_StateCommitmentChain": "0x1ba99640444B81f3928e4F174CFB4FF426B4FFAE",
  "OVM_StateManagerFactory": "0xc4E3E4F9631220f2B1Ada9ee1164E30640c56c94",
  "OVM_StateTransitionerFactory": "0xAC82f9F03f51c8fFef9Ff0362973e89C0dA4aa40",
  "Proxy__OVM_L1CrossDomainMessenger": "0xF10EEfC14eB5b7885Ea9F7A631a21c7a82cf5D76",
  "Proxy__OVM_L1StandardBridge": "0xDe085C82536A06b40D20654c2AbA342F2abD7077",
  "OVM_BondManager": "0x2Ba9F9a6D6D7F604E9e2ca2Ea5f8C9Fa75E13835",
  "OVM_Sequencer": "0xE48E5b731FAAb955d147FA954cba19d93Dc03529",
  "Deployer": "0x122816e7A7AeB40601d0aC0DCAA8402F7aa4cDfA"
}
```

Images:

```javascript
{
  "deployer": "omgx/deployer-rinkeby:production-v1",
  "data_transport_layer": "omgx/data-transport-layer:production-v1",
  "geth_l2": "omgx/l2geth:production-v1",
  "batch_submitter": "omgx/batch-submitter:production-v1",
  "message_relayer": "omgx/message-relayer:production-v1",
  "message_relayer_fast": "omgx/message-relayer-fast:production-v1"
}
```

# 
