import './setup'

/* External Imports */
import {
  runFullnode,
  FullnodeContext
} from '@eth-optimism/rollup-full-node'
import {Address} from '@eth-optimism/rollup-core'
import {add0x, sleep} from '@eth-optimism/core-utils'

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

