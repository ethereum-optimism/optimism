import '../setup'

/* External Imports */
import { add0x, getLogger } from '@pigi/core-utils'

import { Contract, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as L2ExecutionManager from '../../build/contracts/L2ExecutionManager.json'
import * as PurityChecker from '../../build/contracts/PurityChecker.json'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('l2-execution-manager-calls', true)

/*********
 * TESTS *
 *********/

const zero32: string = add0x('00'.repeat(32))
const key: string = add0x('01'.repeat(32))
const value: string = add0x('02'.repeat(32))

describe('L2 Execution Manager', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Useful constant
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  // Create pointers to our execution manager & simple copier contract
  let l2ExecutionManager: Contract
  let purityChecker: Contract

  /* Link libraries before tests */
  before(async () => {
    purityChecker = await deployContract(
      wallet,
      PurityChecker,
      [ONE_FILLED_BYTES_32],
      { gasLimit: 6700000 }
    )
  })
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    l2ExecutionManager = await deployContract(
      wallet,
      L2ExecutionManager,
      [purityChecker.address, '0x' + '00'.repeat(20)],
      { gasLimit: 6700000 }
    )
  })

  describe('Store external-to-internal tx hash map', async () => {
    it('properly maps OVM tx hash to internal tx hash', async () => {
      await l2ExecutionManager.mapOvmTransactionHashToInternalTransactionHash(
        key,
        value
      )
    })

    it('properly reads non-existent mapping', async () => {
      const result = await l2ExecutionManager.getInternalTransactionHash(key)
      result.should.equal(zero32, 'Incorrect unpopulated result!')
    })

    it('properly reads existing mapping', async () => {
      await l2ExecutionManager.mapOvmTransactionHashToInternalTransactionHash(
        key,
        value
      )
      const result = await l2ExecutionManager.getInternalTransactionHash(key)
      result.should.equal(value, 'Incorrect populated result!')
    })
  })
})
