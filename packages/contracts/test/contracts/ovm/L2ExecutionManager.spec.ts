import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { add0x, getLogger } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { DEFAULT_OPCODE_WHITELIST_MASK, GAS_LIMIT } from '../../test-helpers'

/* Logging */
const log = getLogger('l2-execution-manager-calls', true)

export const abi = new ethers.utils.AbiCoder()
const zero32: string = add0x('00'.repeat(32))
const key: string = add0x('01'.repeat(32))
const value: string = add0x('02'.repeat(32))

describe('L2 Execution Manager', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let L2ExecutionManager: ContractFactory
  let l2ExecutionManager: Contract
  beforeEach(async () => {
    L2ExecutionManager = await ethers.getContractFactory('L2ExecutionManager')
    l2ExecutionManager = await L2ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
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
