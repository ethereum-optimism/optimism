import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger, remove0x, add0x } from '@eth-optimism/core-utils'
import * as ethereumjsAbi from 'ethereumjs-abi'
import { Contract, ContractFactory } from 'ethers'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import * as InvalidOpcodes from '../../build/contracts/InvalidOpcodes.json'

const log = getLogger('execution-manager-create', true)

/* Internal Imports */
import { OPCODE_WHITELIST_MASK, GAS_LIMIT } from '../../src/app'
import {
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { fromPairs } from "lodash";

const methodIds = fromPairs([
  'ovmCREATE',
  'ovmCREATE2',
].map((methodId) =>
  [methodId, encodeMethodId(methodId)])
)

/*********
 * TESTS *
 *********/

describe('ExecutionManager -- Create opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  let executionManager: Contract
  let purityCheckedExecutioManager: Contract
  let deployTx
  let deployInvalidTx

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    deployTx = new ContractFactory(
      SimpleStorage.abi,
      SimpleStorage.bytecode
    ).getDeployTransaction(executionManager.address)

    purityCheckedExecutioManager = await deployContract(
      wallet,
      ExecutionManager, // Note: this is false, so it's purity checked.
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, false],
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
        remove0x(deployTx.data)
      ])

      log.debug(`Result: [${result}]`)

      const address: string = remove0x(result)
      address.length.should.equal(64, 'Should be a full word for the address')
      address.should.not.equal('00'.repeat(32), 'Should not be 0 address')
    })

    it('returns 0 address when passed invalid bytecode', async () => {
      const data = add0x(methodIds.ovmCREATE + encodeRawArguments([deployInvalidTx.data]))

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: purityCheckedExecutioManager.address,
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
      const data = add0x(methodIds.ovmCREATE2 + encodeRawArguments([0, deployTx.data]))

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
      const data = add0x(methodIds.ovmCREATE2 + encodeRawArguments([0, deployInvalidTx.data]))

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: purityCheckedExecutioManager.address,
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
