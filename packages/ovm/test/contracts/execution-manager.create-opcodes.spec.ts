import '../setup'

/* External Imports */
import { getLogger, remove0x, add0x } from '@eth-optimism/core-utils'
import {
  ExecutionManagerContractDefinition as ExecutionManager,
  TestSimpleStorageArgsFromCalldataDefinition as SimpleStorage,
  TestInvalidOpcodesContractDefinition as InvalidOpcodes,
} from '@eth-optimism/rollup-contracts'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  DEFAULT_CHAIN_PARAMS,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '@eth-optimism/rollup-core'

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract, ContractFactory } from 'ethers'

const log = getLogger('execution-manager-create', true)

/* Internal Imports */
import {
  gasLimit,
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { fromPairs } from 'lodash'

const methodIds = fromPairs(
  ['ovmCREATE', 'ovmCREATE2'].map((methodId) => [
    methodId,
    encodeMethodId(methodId),
  ])
)

/*********
 * TESTS *
 *********/

describe('ExecutionManager -- Create opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  let executionManager: Contract
  let safetyCheckedExecutionManager: Contract
  let deployTx
  let deployInvalidTx

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), DEFAULT_CHAIN_PARAMS, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    deployTx = new ContractFactory(
      SimpleStorage.abi,
      SimpleStorage.bytecode
    ).getDeployTransaction(executionManager.address)

    safetyCheckedExecutionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), DEFAULT_CHAIN_PARAMS, false], // Note: this is false, so it's safety checked.
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    deployInvalidTx = new ContractFactory(
      InvalidOpcodes.abi,
      InvalidOpcodes.bytecode
    ).getDeployTransaction()
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
        gasLimit,
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
        gasLimit,
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
        gasLimit,
      })

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.equal('00'.repeat(32), 'Should be 0 address')
    })
  })
})
