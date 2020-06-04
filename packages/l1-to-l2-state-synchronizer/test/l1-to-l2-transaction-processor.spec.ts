import './setup'

/* External Imports */
import {
  deployContract,
  L1ToL2TransactionProcessor,
  getUnsignedTransactionCalldata,
  Environment,
} from '@eth-optimism/rollup-core'
import { TestSimpleStorageContractDefinition } from '@eth-optimism/rollup-contracts'
import { add0x, sleep } from '@eth-optimism/core-utils'
import { FullnodeContext, runFullnode } from '@eth-optimism/rollup-full-node'

import { Contract, Wallet } from 'ethers'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import { runTest } from '../exec'

const storageKey: string = '0x' + '01'.repeat(32)
const storageValue: string = '0x' + '22'.repeat(32)

describe('L1 To L2 Tx Processor Integration Tests', () => {
  let fullnodeContext: FullnodeContext
  let l2SimpleStorage: Contract
  let txProcessor: L1ToL2TransactionProcessor

  beforeEach(async () => {
    process.env.NO_L1_TO_L2_TX_PROCESSOR = '1'
    fullnodeContext = await runFullnode(true)
    const fullNodeProvider = new JsonRpcProvider(
      `http://${Environment.l2RpcServerHost()}:${Environment.l2RpcServerPort()}`
    )

    const wallet = Wallet.createRandom().connect(fullNodeProvider)
    l2SimpleStorage = await deployContract(
      wallet,
      TestSimpleStorageContractDefinition,
      []
    )

    process.env.L1_TO_L2_TX_PROCESSOR_PRIVATE_KEY = Wallet.createRandom().privateKey
    txProcessor = await runTest(
      fullnodeContext.l1NodeContext.provider,
      fullNodeProvider
    )
  })

  afterEach(async () => {
    delete process.env.NO_L1_TO_L2_TX_PROCESSOR
    delete process.env.L1_TO_L2_TX_PROCESSOR_PRIVATE_KEY
  })

  it('Receives and executes L1 to L2 Tx', async () => {
    const ovmEntrypoint = l2SimpleStorage.address
    const ovmCalldata: string = getUnsignedTransactionCalldata(
      l2SimpleStorage,
      'setStorage',
      [storageKey, storageValue]
    )

    await fullnodeContext.l1NodeContext.l1ToL2TransactionPasser.passTransactionToL2(
      add0x(ovmEntrypoint),
      ovmCalldata
    )

    await sleep(8_000)

    const res = await l2SimpleStorage.getStorage(storageKey)
    res.should.equal(storageValue, `L1 Transaction did not flow through!`)
  })
})
