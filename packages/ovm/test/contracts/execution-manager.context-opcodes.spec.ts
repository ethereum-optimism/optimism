import { should } from '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  bufferUtils,
  bufToHexString,
  getLogger,
  hexStrToNumber,
  remove0x,
  TestUtils,
} from '@eth-optimism/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as ContextContract from '../../build/contracts/ContextContract.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
  bytes32AddressToAddress,
  addressToBytes32Address,
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
  executeOVMCall,
  encodeRawArguments,
  encodeMethodId,
} from '../helpers'
import { GAS_LIMIT, OPCODE_WHITELIST_MASK } from '../../src/app'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-context', true)

const executeCallMethodId: string = ethereumjsAbi
  .methodID('executeCall', [])
  .toString('hex')

const callThroughEMMethodId: string = ethereumjsAbi
  .methodID('callThroughExecutionManager', [])
  .toString('hex')

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Context opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  const defaultTimestampAndQueueOrigin: string = '00'.repeat(64)

  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let contract: ContractFactory
  let contract2: ContractFactory
  let contractAddress: Address
  let contract2Address: Address
  let contractAddress32: string
  let contract2Address32: string

  beforeEach(async () => {
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
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
    contract = new ContractFactory(
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

    log.debug(`Contract address: [${contractAddress}]`)

    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    contract2 = new ContractFactory(
      ContextContract.abi as any,
      ContextContract.bytecode
    )

    contractAddress32 = addressToBytes32Address(contractAddress)
    contract2Address32 = addressToBytes32Address(contract2Address)
  })

  describe('ovmCALLER', async () => {
    it('reverts when CALLER is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeCall([contractAddress32, encodeMethodId('ovmCALLER')])
      })
    })

    it('properly retrieves CALLER when caller is set', async () => {
      const result = await executeCall([
        contractAddress32,
        encodeMethodId('callThroughExecutionManager'),
        contract2Address32,
        encodeMethodId('getCALLER'),
      ])
      log.debug(`CALLER result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contractAddress32, 'Addresses do not match.')
    })
  })

  describe('ovmADDRESS', async () => {
    it('reverts when ADDRESS is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeCall([contractAddress32, encodeMethodId('ovmADDRESS')])
      })
    })

    it('properly retrieves ADDRESS when address is set', async () => {
      const result = await executeCall([
        contractAddress32,
        encodeMethodId('callThroughExecutionManager'),
        contract2Address32,
        encodeMethodId('getADDRESS'),
      ])

      log.debug(`ADDRESS result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contract2Address32, 'Addresses do not match.')
    })
  })

  describe('ovmTIMESTAMP', async () => {
    it('reverts when TIMESTAMP is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeCall([
          contractAddress32,
          encodeMethodId('callThroughExecutionManager'),
          contract2Address32,
          encodeMethodId('getTIMESTAMP'),
        ])
      })
    })

    it('properly retrieves TIMESTAMP when timestamp is set', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getTIMESTAMP', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${timestampMethodId}`

      const timestamp: string = '00'.repeat(30) + '1111'
      const queueOrigin: string = '00'.repeat(32)

      const data = `0x${executeCallMethodId}${timestamp}${queueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      // const result = await executionManager.provider.call({
      //   to: executionManager.address,
      //   data,
      //   gasLimit,
      // })
      const result = await executeOVMCall(executionManager, 'executeCall', [
        99,
        0,
        contractAddress32,
        encodeMethodId('callThroughExecutionManager'),
        contract2Address32,
        encodeMethodId('getTIMESTAMP'),
      ])

      log.debug(`TIMESTAMP result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.equal(99, 'Timestamps do not match.')
    })
  })

  describe('ovmGASLIMIT', async () => {
    it('properly retrieves GASLIMIT', async () => {
      const gasLimitMethodId: string = ethereumjsAbi
        .methodID('getGASLIMIT', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contractAddress32
      )}${gasLimitMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contract2Address32
      )}${internalCall}`

      const result = await executeCall([
        contractAddress32,
        encodeMethodId('callThroughExecutionManager'),
        contract2Address32,
        encodeMethodId('getGASLIMIT'),
      ])

      log.debug(`GASLIMIT result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.equal(GAS_LIMIT, 'Gas limits do not match.')
    })
  })

  describe('ovmQueueOrigin', async () => {
    it('gets Queue Origin when it is 0', async () => {
      const timestampMethodId: string = ethereumjsAbi
        .methodID('getQueueOrigin', [])
        .toString('hex')

      const internalCall: string = `${callThroughEMMethodId}${remove0x(
        contract2Address32
      )}${timestampMethodId}`

      const data = `0x${executeCallMethodId}${defaultTimestampAndQueueOrigin}${remove0x(
        contractAddress32
      )}${internalCall}`

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(
        defaultTimestampAndQueueOrigin.substr(64),
        'Queue origins do not match.'
      )
    })

    it('properly retrieves Queue Origin when queue origin is set', async () => {
      const queueOrigin: string = '00'.repeat(30) + '1111'
      const result = await executeOVMCall(executionManager, 'executeCall', [
        0,
        queueOrigin,
        contractAddress32,
        encodeMethodId('callThroughExecutionManager'),
        contract2Address32,
        encodeMethodId('getQueueOrigin'),
      ])

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })
  })

  const executeCall = (args: any[]): Promise<string> => {
    return executeOVMCall(executionManager, 'executeCall', [
      encodeRawArguments([0, 0, ...args]),
    ])
  }
})
