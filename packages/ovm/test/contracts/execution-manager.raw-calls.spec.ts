import '../setup'

/* External Imports */
import { Address } from '@pigi/rollup-core'
import { getLogger, BigNumber, remove0x, add0x } from '@pigi/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as DummyContract from '../../build/contracts/DummyContract.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
} from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('raw-calls', true)

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Raw Calls', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let dummyContract: ContractFactory
  let dummyContractAddress: Address

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )

    // Deploy SimpleCopier with the ExecutionManager
    dummyContractAddress = await manuallyDeployOvmContract(
      provider,
      executionManager,
      DummyContract,
      []
    )

    log.debug(`Contract address: [${dummyContractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    dummyContract = new ContractFactory(
      DummyContract.abi as any,
      DummyContract.bytecode
    )
  })

  describe('executeRawCall', async () => {
    it('properly executes a raw call -- 0 param', async () => {
      // Create the variables we will use for setStorage
      const intParam = 0
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = remove0x(
        getUnsignedTransactionCalldata(dummyContract, 'dummyFunction', [
          intParam,
          bytesParam,
        ])
      )

      const methodId: string = ethereumjsAbi
        .methodID('executeRawCall', [])
        .toString('hex')

      const timestamp: string = '00'.repeat(32)
      const queueOrigin: string = timestamp
      const contractAddress: string =
        '00'.repeat(12) + remove0x(dummyContractAddress)
      const encodedParams: string = `${timestamp}${queueOrigin}${contractAddress}${calldata}`
      const data = `0x${methodId}${encodedParams}`

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`Result: [${result}]`)

      remove0x(result).length.should.be.gt(0, 'No result when expected!')
      const [success, byteData] = abi.decode(['bool', 'bytes'], result)

      success.should.equal(
        false,
        'Success should be false since intParam is 0!'
      )
      byteData.should.equal(bytesParam, 'Returned bytes not as expected!')
    })

    it('properly executes a raw call -- 1 param', async () => {
      // Create the variables we will use for setStorage
      const intParam = 1
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = remove0x(
        getUnsignedTransactionCalldata(dummyContract, 'dummyFunction', [
          intParam,
          bytesParam,
        ])
      )

      const methodId: string = ethereumjsAbi
        .methodID('executeRawCall', [])
        .toString('hex')

      const timestamp: string = '00'.repeat(32)
      const queueOrigin: string = timestamp
      const contractAddress: string =
        '00'.repeat(12) + remove0x(dummyContractAddress)
      const encodedParams: string = `${timestamp}${queueOrigin}${contractAddress}${calldata}`
      const data = `0x${methodId}${encodedParams}`

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`Result: [${result}]`)

      remove0x(result).length.should.be.gt(0, 'No result when expected!')
      const [success, byteData] = abi.decode(['bool', 'bytes'], result)

      success.should.equal(
        true,
        'Success should be false since intParam is not 0!'
      )
      byteData.should.equal(bytesParam, 'Returned bytes not as expected!')
    })

    it('returns failure when inner call fails', async () => {
      // Create the variables we will use for setStorage

      const methodId: string = ethereumjsAbi
        .methodID('executeRawCall', [])
        .toString('hex')

      const timestamp: string = '00'.repeat(32)
      const queueOrigin: string = timestamp
      const contractAddress: string =
        '00'.repeat(12) + remove0x(dummyContractAddress)
      const encodedParams: string = `${timestamp}${queueOrigin}${contractAddress}00`
      const data = `0x${methodId}${encodedParams}`

      let failed = false
      try {
        // Now actually apply it to our execution manager
        await executionManager.provider.call({
          to: executionManager.address,
          data,
          gasLimit: 6_700_000,
        })
      } catch (e) {
        if (e.message.indexOf('revert') >= 0) {
          failed = true
        }
      }

      failed.should.equal(true, 'Execution should have failed and reverted.')
    })
  })
})
