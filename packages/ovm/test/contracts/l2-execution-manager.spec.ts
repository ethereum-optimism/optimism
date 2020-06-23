import '../setup'

/* External Imports */
import { add0x, getLogger } from '@eth-optimism/core-utils'
import { L2ExecutionManagerContractDefinition as L2ExecutionManager } from '@eth-optimism/rollup-contracts'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  DEFAULT_CHAIN_PARAMS,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '@eth-optimism/rollup-core'

import { Contract, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */

const log = getLogger('l2-execution-manager-calls', true)

export const abi = new ethers.utils.AbiCoder()

/*********
 * TESTS *
 *********/

const zero32: string = add0x('00'.repeat(32))
const key: string = add0x('01'.repeat(32))
const value: string = add0x('02'.repeat(32))

describe('L2 Execution Manager', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let l2ExecutionManager: Contract

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    l2ExecutionManager = await deployContract(
      wallet,
      L2ExecutionManager,
      [
        DEFAULT_OPCODE_WHITELIST_MASK,
        '0x' + '00'.repeat(20),
        DEFAULT_CHAIN_PARAMS,
        true,
      ],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
  })

  describe('Store OVM transactions', async () => {
    const fakeSignedTx = add0x(
      Buffer.from('derp')
        .toString('hex')
        .repeat(20)
    )
    it('properly maps OVM tx hash to internal tx hash', async () => {
      await l2ExecutionManager.storeOvmTransaction(key, value, fakeSignedTx)
    })

    it('properly reads non-existent mapping', async () => {
      const result = await l2ExecutionManager.getInternalTransactionHash(key)
      result.should.equal(zero32, 'Incorrect unpopulated result!')
    })

    it('properly reads existing OVM tx hash -> internal tx hash mapping', async () => {
      await l2ExecutionManager.storeOvmTransaction(key, value, fakeSignedTx)
      const result = await l2ExecutionManager.getInternalTransactionHash(key)
      result.should.equal(value, 'Incorrect hash mapped!')
    })

    it('properly reads existing internal tx hash -> OVM tx hash mapping', async () => {
      await l2ExecutionManager.storeOvmTransaction(key, value, fakeSignedTx)
      const result = await l2ExecutionManager.getOvmTransactionHash(value)
      result.should.equal(key, 'Incorrect hash mapped!')
    })

    it('properly reads existing OVM tx hash -> OVM tx mapping', async () => {
      await l2ExecutionManager.storeOvmTransaction(key, value, fakeSignedTx)
      const result = await l2ExecutionManager.getOvmTransaction(key)
      result.should.equal(fakeSignedTx, 'Incorrect tx mapped!')
    })
  })
})
