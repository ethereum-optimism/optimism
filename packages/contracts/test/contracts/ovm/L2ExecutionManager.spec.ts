import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { add0x, getLogger, NULL_ADDRESS } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  makeAddressResolver,
  AddressResolverMapping,
  fillHexBytes
} from '../../test-helpers'

/* Logging */
const log = getLogger('l2-execution-manager-calls', true)

const zero32: string = fillHexBytes('00')
const key: string = fillHexBytes('01')
const value: string = fillHexBytes('02')

describe('L2 Execution Manager', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let L2ExecutionManager: ContractFactory
  before(async () => {
    L2ExecutionManager = await ethers.getContractFactory('L2ExecutionManager')
  })

  let l2ExecutionManager: Contract
  beforeEach(async () => {
    l2ExecutionManager = await L2ExecutionManager.deploy(
      resolver.addressResolver.address,
      NULL_ADDRESS,
      GAS_LIMIT
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
