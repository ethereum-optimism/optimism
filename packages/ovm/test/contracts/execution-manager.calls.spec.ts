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
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
} from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-calls', true)

/*********
 * TESTS *
 *********/

const methodId: string = ethereumjsAbi
  .methodID('executeCall', [])
  .toString('hex')

describe('Execution Manager -- Calls', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let contractAddressGenerator: Contract
  let rlpEncode: Contract
  let dummyContract: ContractFactory
  let dummyContractAddress: Address

  /* Link libraries before tests */
  before(async () => {
    rlpEncode = await deployContract(wallet, RLPEncode, [], {
      gasLimit: 6700000,
    })
    contractAddressGenerator = await deployContract(
      wallet,
      ContractAddressGenerator,
      [rlpEncode.address],
      {
        gasLimit: 6700000,
      }
    )
  })
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [
        '0x' + '00'.repeat(20),
        contractAddressGenerator.address,
        '0x' + '00'.repeat(20),
      ],
      {
        gasLimit: 6700000,
      }
    )

    // Deploy SimpleCopier with the ExecutionManager
    dummyContractAddress = await manuallyDeployOvmContract(
      wallet,
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

  describe('executeCall', async () => {
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
