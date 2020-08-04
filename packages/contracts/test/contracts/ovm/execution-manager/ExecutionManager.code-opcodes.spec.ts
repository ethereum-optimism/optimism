import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  BigNumber,
  hexStrToBuf,
  remove0x,
  keccak256,
  NULL_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'
import { DEFAULT_GAS_METER_PARAMS } from '@eth-optimism/rollup-core'

/* Internal Imports */
import {
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  Address,
  manuallyDeployOvmContract,
  executeOVMCall,
  addressToBytes32Address,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
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

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
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
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          DEFAULT_GAS_METER_PARAMS,
        ],
      }
    )
  })

  let dummyContractAddress: Address
  let dummyContractBytecode: Buffer
  beforeEach(async () => {
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
