import { ethers } from 'ethers'
import { describe, it, expect } from 'vitest'

import { getEvents } from './getEvents'

describe(getEvents.name, () => {
  it('should get events on goerli', async () => {
    const key = 'animalfarm.school.attended'
    const creator = '0xBCf86Fd70a0183433763ab0c14E7a760194f3a9F'
    expect(
      await getEvents({
        creator,
        about: '0x00000000000000000000000000000000000060A7',
        key,
        provider: new ethers.providers.JsonRpcProvider(
          'https://goerli.optimism.io'
        ),
      })
    ).toMatchInlineSnapshot(`
      [
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0xBCf86Fd70a0183433763ab0c14E7a760194f3a9F",
            "0x00000000000000000000000000000000000060A7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
            "0x01",
          ],
          "blockHash": "0x75feb3572d4b7d682cf632bf64df72c8d9c336dedcf8df1c88f755d529ec1b85",
          "blockNumber": 3463240,
          "data": "0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010100000000000000000000000000000000000000000000000000000000000000",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 0,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x000000000000000000000000bcf86fd70a0183433763ab0c14e7a760194f3a9f",
            "0x00000000000000000000000000000000000000000000000000000000000060a7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
          ],
          "transactionHash": "0x0e77a32b2558f39e60c3e81bd6efd811cf4b3bd80a4f666d042a221ea63c93ab",
          "transactionIndex": 0,
        },
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0xBCf86Fd70a0183433763ab0c14E7a760194f3a9F",
            "0x00000000000000000000000000000000000060A7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
            "0x01",
          ],
          "blockHash": "0xdb11b4b06e5866be931667b8c62dca182240b9256a3d8c64c1c247107aa33752",
          "blockNumber": 4105095,
          "data": "0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010100000000000000000000000000000000000000000000000000000000000000",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 0,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x000000000000000000000000bcf86fd70a0183433763ab0c14e7a760194f3a9f",
            "0x00000000000000000000000000000000000000000000000000000000000060a7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
          ],
          "transactionHash": "0x61f59bd4dfe54272d9369effe3ae57a0ef2584161fcf2bbd55f5596002e759bd",
          "transactionIndex": 1,
        },
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0xBCf86Fd70a0183433763ab0c14E7a760194f3a9F",
            "0x00000000000000000000000000000000000060A7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
            "0x01",
          ],
          "blockHash": "0x4870baaac6d7195952dc25e5dc0109ea324f819f8152d2889c7b4ad64040a9bf",
          "blockNumber": 6278428,
          "data": "0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010100000000000000000000000000000000000000000000000000000000000000",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 0,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x000000000000000000000000bcf86fd70a0183433763ab0c14e7a760194f3a9f",
            "0x00000000000000000000000000000000000000000000000000000000000060a7",
            "0x616e696d616c6661726d2e7363686f6f6c2e617474656e646564000000000000",
          ],
          "transactionHash": "0x4e836b74c51a370375efa374297524d9b0f6eacdd699c30556680ae7dc9a14ea",
          "transactionIndex": 1,
        },
      ]
    `)
  })
  it('should get events on mainnet', async () => {
    const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
    const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
    const key = 'optimist.base-uri'
    expect(
      await getEvents({
        creator,
        about,
        key,
        provider: new ethers.providers.JsonRpcProvider('http://localhost:8545'),
      })
    ).toMatchInlineSnapshot(`
      [
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3",
            "0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
            "0x68747470733a2f2f73746f726167656170692e666c65656b2e636f2f33336630633965392d666437392d343634622d613431642d3634343238313961316230352d6275636b65742f6f7074696d6973742d6e66742f61747472696275746573",
          ],
          "blockHash": "0x5b5f34cb7a72eb6aaf6d8af873f210278738573386c88f85c605067c10d67ee3",
          "blockNumber": 50135778,
          "data": "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000005f68747470733a2f2f73746f726167656170692e666c65656b2e636f2f33336630633965392d666437392d343634622d613431642d3634343238313961316230352d6275636b65742f6f7074696d6973742d6e66742f6174747269627574657300",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 1,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x00000000000000000000000060c5c9c98bcbd0b0f2fd89b24c16e533baa8cda3",
            "0x0000000000000000000000002335022c740d17c2837f9c884bfe4ffdbf0a95d5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
          ],
          "transactionHash": "0x265c98ce12e0836616efd3ea2130df9647729574feb40d5607e4031ff9aace01",
          "transactionIndex": 0,
        },
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3",
            "0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
            "0x68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f61747472696275746573",
          ],
          "blockHash": "0x889ad6bb2eb7aee0c095c1f6cc11f5a7a65917d7bc06500dad3213fb031f1e9c",
          "blockNumber": 50141511,
          "data": "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000005e68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f617474726962757465730000",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 1,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x00000000000000000000000060c5c9c98bcbd0b0f2fd89b24c16e533baa8cda3",
            "0x0000000000000000000000002335022c740d17c2837f9c884bfe4ffdbf0a95d5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
          ],
          "transactionHash": "0xf4c0fc1ceec42831252c90b7d5c1e7a5bd6d9642d07c80afc8b525211852ee03",
          "transactionIndex": 0,
        },
        {
          "address": "0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77",
          "args": [
            "0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3",
            "0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
            "0x68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f61747472696275746573",
          ],
          "blockHash": "0x120931c24234d03af66b9b21fcaf3b97242ed8f0c0418a9b16fc5cc1a804e917",
          "blockNumber": 50141837,
          "data": "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000005e68747470733a2f2f6173736574732e6f7074696d69736d2e696f2f34613630393636312d363737342d343431662d396664622d3435336664626238393933312d6275636b65742f6f7074696d6973742d6e66742f617474726962757465730000",
          "decode": [Function],
          "event": "AttestationCreated",
          "eventSignature": "AttestationCreated(address,address,bytes32,bytes)",
          "getBlock": [Function],
          "getTransaction": [Function],
          "getTransactionReceipt": [Function],
          "logIndex": 1,
          "removeListener": [Function],
          "removed": false,
          "topics": [
            "0x28710dfecab43d1e29e02aa56b2e1e610c0bae19135c9cf7a83a1adb6df96d85",
            "0x00000000000000000000000060c5c9c98bcbd0b0f2fd89b24c16e533baa8cda3",
            "0x0000000000000000000000002335022c740d17c2837f9c884bfe4ffdbf0a95d5",
            "0x6f7074696d6973742e626173652d757269000000000000000000000000000000",
          ],
          "transactionHash": "0xfaf727afe431a920448636b80864dfeeef690903756f9c3041eb625ffcc82f11",
          "transactionIndex": 0,
        },
      ]
    `)
  })
})
