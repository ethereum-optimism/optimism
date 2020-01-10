import '../setup'

/* External Imports */
import { getLogger } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

const log = getLogger('state-manager', true)

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Begin tests */
describe.skip('ExecutionManager', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager

  /* Link libraries before tests */
  before(async () => {
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )
  })

  /*
   * Test hello world!
   */
  describe('Hello World!', async () => {
    it('Hello World!', async () => {
      log.info('Hello World!')
    })
  })
})
