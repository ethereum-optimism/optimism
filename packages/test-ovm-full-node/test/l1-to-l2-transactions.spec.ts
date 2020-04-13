import './setup'

/* External Imports */
import {
  runFullnode,
  FullnodeContext
} from '@eth-optimism/rollup-full-node'
import {Address} from '@eth-optimism/rollup-core'
import {add0x, hexStrToBuf, keccak256, sleep} from '@eth-optimism/core-utils'

import {Contract, Wallet} from 'ethers'
import {JsonRpcProvider} from 'ethers/providers'
import {deployContract} from 'ethereum-waffle'

/* Internal Imports */
import {getUnsignedTransactionCalldata} from '../src'

/* Contract Imports */
import * as SimpleStorage from '../build/SimpleStorage.json'


const storageKey: string = '0x' + '01'.repeat(32)
const storageValue: string = '0x' + '22'.repeat(32)

describe('L1 To L2 Transaction Passing', () => {
  let wallet: Wallet
  let simpleStorage: Contract
  let provider: JsonRpcProvider
  let rollupFullnodeContext: FullnodeContext

  describe('Local tests', () => {
    before(async () => {
      rollupFullnodeContext = await runFullnode(true)
    })

    after(async () => {
      try {
        await rollupFullnodeContext.fullnodeRpcServer.close()
      } catch (e) {
        // don't do anything
      }
    })

    beforeEach(async () => {
      provider = new JsonRpcProvider('http://0.0.0.0:8545')
      wallet = new Wallet(Wallet.createRandom().privateKey, provider)

      simpleStorage = await deployContract(wallet, SimpleStorage, [])
    })

    it('should process l1-to-l2-transaction properly', async () => {
      const ovmEntrypoint: Address = simpleStorage.address
      const ovmCalldata: string = getUnsignedTransactionCalldata(simpleStorage, 'setStorage', [storageKey, storageValue])

      await rollupFullnodeContext.l1NodeContext.l1ToL2TransactionPasser.passTransactionToL2(add0x(ovmEntrypoint), ovmCalldata)

      await sleep(8_000)

      const res = await simpleStorage.getStorage(storageKey)
      res.should.equal(storageValue, `L1 Transaction did not flow through!`)
    }).timeout(20_000)
  })

  // describe('Rinkeby tests!', () => {
  //   before(async () => {
  //     // TODO: Update earliest block here
  //     process.env.L1_EARLIEST_BLOCK = ''
  //     // TODO: Update PK here:
  //     process.env.L1_SEQUENCER_PRIVATE_KEY = ''
  //     process.env.L1_NODE_INFURA_NETWORK = 'rinkeby'
  //     // TODO: Update Infura project ID here:
  //     process.env.L1_NODE_INFURA_PROJECT_ID = ''
         // These are the addresses of the contracts on rinkeby
  //     process.env.L1_TO_L2_TRANSACTION_PASSER_ADDRESS = '0xcF8aF92c52245C6595A2de7375F405B24c3a05BD'
  //     process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS = '0x3cD9393742c656c5E33A1a6ee73ef4B27fd54951'
  //
  //     rollupFullnodeContext = await runFullnode(true)
  //   })
  //
  //   after(async () => {
  //     try {
  //       await rollupFullnodeContext.fullnodeRpcServer.close()
  //     } catch (e) {
  //       // don't do anything
  //     }
  //
  //     delete process.env.L1_EARLIEST_BLOCK
  //     delete process.env.L1_SEQUENCER_PRIVATE_KEY
  //     delete process.env.L1_NODE_INFURA_NETWORK
  //     delete process.env.L1_NODE_INFURA_PROJECT_ID
  //     delete process.env.L1_TO_L2_TRANSACTION_PASSER_ADDRESS
  //     delete process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS
  //   })
  //
  //   beforeEach(async () => {
  //     provider = new JsonRpcProvider('http://0.0.0.0:8545')
  //     wallet = new Wallet(Wallet.createRandom().privateKey, provider)
  //
  //     simpleStorage = await deployContract(wallet, SimpleStorage, [])
  //   })
  //
  //   it.only('should process l1-to-l2-transaction properly', async () => {
  //     const k: number = Math.floor(Math.random() * Math.floor(9007199254740991))
  //     const v: number = Math.floor(Math.random() * Math.floor(9007199254740991))
  //
  //     const randomKey: string = add0x(keccak256(k.toString(16)))
  //     const randomValue: string = add0x(keccak256(v.toString(16)))
  //
  //     console.log(`Sending L1 message passer contract key ${randomKey} and value: ${randomValue}`)
  //
  //     const ovmEntrypoint: Address = simpleStorage.address
  //     const ovmCalldata: string = getUnsignedTransactionCalldata(simpleStorage, 'setStorage', [randomKey, randomValue])
  //
  //     await rollupFullnodeContext.l1NodeContext.l1ToL2TransactionPasser.passTransactionToL2(add0x(ovmEntrypoint), ovmCalldata)
  //
  //     console.log(`Waiting 60s for tx to be mined.`)
  //     await sleep(60_000)
  //
  //     console.log(`Fetching key ${randomKey} from L2 contract.`)
  //     const res = await simpleStorage.getStorage(randomKey)
  //     console.log(`Received value ${res} from L2 contract.`)
  //     res.should.equal(randomValue, `L1 Transaction did not flow through!`)
  //   }).timeout(65_000)
  // })

})

