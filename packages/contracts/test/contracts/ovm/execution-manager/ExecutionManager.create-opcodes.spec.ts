import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, remove0x, add0x } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'
import { fromPairs } from 'lodash'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-create', true)

const methodIds = fromPairs(
  ['ovmCREATE', 'ovmCREATE2'].map((methodId) => [
    methodId,
    encodeMethodId(methodId),
  ])
)

/* Tests */
describe('ExecutionManager -- Create opcodes', () => {
  let ExecutionManager: ContractFactory
  let SimpleStorage: ContractFactory
  let InvalidOpcodes: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    SimpleStorage = await ethers.getContractFactory('SimpleStorage')
    InvalidOpcodes = await ethers.getContractFactory('InvalidOpcodes')
  })

  let executionManager: Contract
  let safetyCheckedExecutionManager: Contract
  let deployTx: any
  let deployInvalidTx: any
  beforeEach(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
    )

    deployTx = SimpleStorage.getDeployTransaction()

    safetyCheckedExecutionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      false
    )

    deployInvalidTx = InvalidOpcodes.getDeployTransaction()
  })

  describe('ovmCREATE', async () => {
    it('returns created address when passed valid bytecode', async () => {
      const result = await executeOVMCall(executionManager, 'ovmCREATE', [
        deployTx.data,
      ])

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('returns 0 address when passed invalid bytecode', async () => {
      const data = add0x(
        methodIds.ovmCREATE + encodeRawArguments([deployInvalidTx.data])
      )

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: safetyCheckedExecutionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.equal('00'.repeat(32), 'Should be 0 address')
    })
  })

  describe('ovmCREATE2', async () => {
    it('returns created address when passed salt and bytecode', async () => {
      const data = add0x(
        methodIds.ovmCREATE2 + encodeRawArguments([0, deployTx.data])
      )

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('returns 0 address when passed salt and invalid bytecode', async () => {
      const data = add0x(
        methodIds.ovmCREATE2 + encodeRawArguments([0, deployInvalidTx.data])
      )

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: safetyCheckedExecutionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.equal('00'.repeat(32), 'Should be 0 address')
    })
  })
})
