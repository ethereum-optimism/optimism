import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  add0x,
  BigNumber,
  hexStrToBuf,
  remove0x,
  keccak256,
  bufferUtils,
  bufToHexString,
} from '@eth-optimism/core-utils'

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
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
  executeOVMCall,
  addressToBytes32Address,
} from '../helpers'
import { GAS_LIMIT, OPCODE_WHITELIST_MASK } from '../../src/app'

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
      [OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
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
    it.only('properly gets contract code size for the contract we expect', async () => {
      const result: string = await executeOVMCall(
        executionManager,
        wallet,
        "EXTCODESIZE",
        [
          addressToBytes32Address(dummyContractAddress),
        ]
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
        wallet,
        "EXTCODEHASH",
        [
          addressToBytes32Address(dummyContractAddress),
        ]
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
        wallet,
        "EXTCODECOPY",
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
        wallet,
        "EXTCODECOPY",
        [
          addressToBytes32Address(dummyContractAddress),
          0,
          dummyContractBytecode.length + 3,
        ]
      )
      log.debug(`Resulting code: [${code}]`)

      const codeBuff: Buffer = hexStrToBuf(code)
      const bytecodeWithZeroedBytes = Buffer.concat([dummyContractBytecode, Buffer.alloc(3)])
      codeBuff.should.eql(bytecodeWithZeroedBytes, 'Incorrect code!')
    })
  })
})


