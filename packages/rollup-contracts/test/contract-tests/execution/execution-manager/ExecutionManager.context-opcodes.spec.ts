import { should } from '../../../setup'

/* External Imports */
import {
  getLogger,
  hexStrToNumber,
  remove0x,
  add0x,
  ZERO_ADDRESS,
  TestUtils,
  getCurrentTime,
} from '@eth-optimism/core-utils'
import {
  Address,
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '@eth-optimism/rollup-core'

import {
  ExecutionManagerContractDefinition as ExecutionManager,
  TestContextContractDefinition as ContextContract,
} from '@eth-optimism/rollup-contracts'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  addressToBytes32Address,
  encodeRawArguments,
  encodeMethodId,
  gasLimit,
} from '../../../test-helpers'
import { fromPairs } from 'lodash'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-context', true)

const methodIds = fromPairs(
  [
    'callThroughExecutionManager',
    'getADDRESS',
    'getCALLER',
    'getGASLIMIT',
    'getQueueOrigin',
    'getTIMESTAMP',
    'ovmADDRESS',
    'ovmCALLER',
  ].map((methodId) => [methodId, encodeMethodId(methodId)])
)

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Context opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)

  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let contractAddress: Address
  let contract2Address: Address
  let contractAddress32: string
  let contract2Address32: string

  beforeEach(async () => {
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    // Deploy SimpleCopier with the ExecutionManager
    contractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract address: [${contractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    const contract = new ContractFactory(
      ContextContract.abi as any,
      ContextContract.bytecode
    )

    // Deploy SimpleCopier with the ExecutionManager
    contract2Address = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract 2 address: [${contract2Address}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    const contract2 = new ContractFactory(
      ContextContract.abi as any,
      ContextContract.bytecode
    )

    contractAddress32 = addressToBytes32Address(contractAddress)
    contract2Address32 = addressToBytes32Address(contract2Address)
  })

  describe('ovmCALLER', async () => {
    it('reverts when CALLER is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeTransaction(contractAddress, methodIds.ovmCALLER, [])
      })
    })

    it('properly retrieves CALLER when caller is set', async () => {
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getCALLER]
      )
      log.debug(`CALLER result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contractAddress32, 'Addresses do not match.')
    })
  })

  describe('ovmADDRESS', async () => {
    it('reverts when ADDRESS is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeTransaction(contractAddress, methodIds.ovmADDRESS, [])
      })
    })

    it('properly retrieves ADDRESS when address is set', async () => {
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getADDRESS]
      )

      log.debug(`ADDRESS result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contract2Address32, 'Addresses do not match.')
    })
  })

  describe('ovmTIMESTAMP', async () => {
    it('properly retrieves TIMESTAMP', async () => {
      const timestamp: number = getCurrentTime()
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getTIMESTAMP]
      )

      log.debug(`TIMESTAMP result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.be.gte(
        timestamp,
        'Timestamps do not match.'
      )
    })
  })

  describe('ovmGASLIMIT', async () => {
    it('properly retrieves GASLIMIT', async () => {
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getGASLIMIT]
      )

      log.debug(`GASLIMIT result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.equal(GAS_LIMIT, 'Gas limits do not match.')
    })
  })

  describe('ovmQueueOrigin', async () => {
    it('gets Queue Origin when it is 0', async () => {
      const queueOrigin: string = '00'.repeat(32)
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getQueueOrigin]
      )

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })

    it('properly retrieves Queue Origin when queue origin is set', async () => {
      const queueOrigin: string = '00'.repeat(30) + '1111'
      const result = await executeTransaction(
        contractAddress,
        methodIds.callThroughExecutionManager,
        [contract2Address32, methodIds.getQueueOrigin],
        add0x(queueOrigin)
      )

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })
  })

  const executeTransaction = async (
    address: string,
    methodId: string,
    args: any[],
    queueOrigin = ZERO_ADDRESS
  ): Promise<string> => {
    const callBytes = add0x(methodId + encodeRawArguments(args))
    const data = executionManager.interface.functions[
      'executeTransaction'
    ].encode([
      getCurrentTime(),
      queueOrigin,
      address,
      callBytes,
      ZERO_ADDRESS,
      ZERO_ADDRESS,
      true,
    ])
    return executionManager.provider.call({
      to: executionManager.address,
      data,
      gasLimit,
    })
  }
})
