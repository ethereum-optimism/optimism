import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  BigNumber,
  hexStrToBuf,
  remove0x,
  keccak256,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  Address,
  manuallyDeployOvmContract,
  executeOVMCall,
  addressToBytes32Address,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-code-opcodes', true)

export const abi = new ethers.utils.AbiCoder()

/* Tests */
describe('Execution Manager -- Code-related opcodes', () => {
  const provider = ethers.provider

  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let ExecutionManager: ContractFactory
  let DummyContract: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    DummyContract = await ethers.getContractFactory('DummyContract')

    const dummyContract = await DummyContract.deploy()
    dummyContractBytecode = hexStrToBuf(
      await provider.getCode(dummyContract.address)
    )
  })

  let executionManager: Contract
  let dummyContractAddress: Address
  let dummyContractBytecode: Buffer
  beforeEach(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
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
