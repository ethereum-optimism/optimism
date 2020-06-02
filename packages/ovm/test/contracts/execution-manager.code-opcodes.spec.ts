import '../setup'

/* External Imports */
import {
  Address,
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '@eth-optimism/rollup-core'
import {
  getLogger,
  BigNumber,
  hexStrToBuf,
  remove0x,
  keccak256,
} from '@eth-optimism/core-utils'

import {
  ExecutionManagerContractDefinition as ExecutionManager,
  TestDummyContractDefinition as DummyContract,
} from '@eth-optimism/rollup-contracts'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  executeOVMCall,
  addressToBytes32Address,
} from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-code-opcodes', true)

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Code-related opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let dummyContract: ContractFactory
  let dummyContractAddress: Address
  const dummyContractBytecode: Buffer = Buffer.from(
    DummyContract.evm.deployedBytecode.object,
    'hex'
  )

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
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

  describe('getContractCodeSize', async () => {
    it('properly gets contract code size for the contract we expect', async () => {
      const result: string = await executeOVMCall(
        executionManager,
        'ovmEXTCODESIZE',
        [addressToBytes32Address(dummyContractAddress)]
      )
      log.debug(`Resulting size: [${result}]`)

      const codeSize: number = new BigNumber(remove0x(result), 'hex').toNumber()
      codeSize.should.equal(
        dummyContractBytecode.length,
        'Incorrect bytecode length!'
      )
    })
  })

  describe('getContractCodeHash', async () => {
    it('properly gets contract code hash for the contract we expect', async () => {
      const codeHash: string = await executeOVMCall(
        executionManager,
        'ovmEXTCODEHASH',
        [addressToBytes32Address(dummyContractAddress)]
      )
      log.debug(`Resulting hash: [${codeHash}]`)

      const hash: string = keccak256(dummyContractBytecode.toString('hex'))

      remove0x(codeHash).should.equal(hash, 'Incorrect code hash!')
    })
  })

  describe('ovmEXTCODECOPY', async () => {
    it('properly gets all contract code via EXTCODECOPY', async () => {
      const code: string = await executeOVMCall(
        executionManager,
        'ovmEXTCODECOPY',
        [
          addressToBytes32Address(dummyContractAddress),
          0,
          dummyContractBytecode.length,
        ]
      )
      log.debug(`Resulting code: [${code}]`)

      const codeBuff: Buffer = hexStrToBuf(code)
      codeBuff.should.eql(dummyContractBytecode, 'Incorrect code!')
    })

    it('returns zeroed bytes if the range is out of bounds', async () => {
      const code: string = await executeOVMCall(
        executionManager,
        'ovmEXTCODECOPY',
        [
          addressToBytes32Address(dummyContractAddress),
          0,
          dummyContractBytecode.length + 3,
        ]
      )
      log.debug(`Resulting code: [${code}]`)

      const codeBuff: Buffer = hexStrToBuf(code)
      const bytecodeWithZeroedBytes = Buffer.concat([
        dummyContractBytecode,
        Buffer.alloc(3),
      ])
      codeBuff.should.eql(bytecodeWithZeroedBytes, 'Incorrect code!')
    })

    it('returns zeroed bytes if the provided address is invalid', async () => {
      const offset = 0
      const length = 99
      const code: string = await executeOVMCall(
        executionManager,
        'ovmEXTCODECOPY',
        [addressToBytes32Address('11'.repeat(20)), 0, length]
      )
      log.debug(`Resulting code: [${code}]`)

      const codeBuff: Buffer = hexStrToBuf(code)
      codeBuff.should.eql(Buffer.alloc(length), 'Incorrect code!')
    })
  })
})
