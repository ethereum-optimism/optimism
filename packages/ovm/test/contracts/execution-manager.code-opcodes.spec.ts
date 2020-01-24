import '../setup'

/* External Imports */
import { Address } from '@pigi/rollup-core'
import {
  getLogger,
  add0x,
  BigNumber,
  hexStrToBuf,
  remove0x,
  keccak256,
  bufferUtils,
  bufToHexString,
} from '@pigi/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as DummyContract from '../../build/contracts/DummyContract.json'
import * as PurityChecker from '../../build/contracts/PurityChecker.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
} from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-code-opcodes', true)

/*********
 * TESTS *
 *********/

describe('Execution Manager -- Code-related opcodes', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let purityChecker: Contract
  let dummyContract: ContractFactory
  let dummyContractAddress: Address
  const dummyContractBytecode: Buffer = Buffer.from(
    DummyContract.evm.deployedBytecode.object,
    'hex'
  )
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)

  /* Link libraries before tests */
  before(async () => {
    purityChecker = await deployContract(
      wallet,
      PurityChecker,
      [ONE_FILLED_BYTES_32],
      { gasLimit: 6700000 }
    )
  })
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [purityChecker.address, '0x' + '00'.repeat(20)],
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

  describe('getContractCodeSize', async () => {
    it('properly gets contract code size for the contract we expect', async () => {
      const methodId: string = ethereumjsAbi
        .methodID('ovmEXTCODESIZE', [])
        .toString('hex')

      const encodedParams: string =
        '00'.repeat(12) + remove0x(dummyContractAddress)
      const data: string = `0x${methodId}${encodedParams}`

      const result: string = await executionManager.provider.call({
        to: add0x(executionManager.address),
        data,
        gasLimit: 6_700_000,
      })
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
      const methodId: string = ethereumjsAbi
        .methodID('ovmEXTCODEHASH', [])
        .toString('hex')

      const encodedParams: string =
        '00'.repeat(12) + remove0x(dummyContractAddress)
      const data: string = `0x${methodId}${encodedParams}`

      const codeHash: string = await executionManager.provider.call({
        to: add0x(executionManager.address),
        data,
        gasLimit: 6_700_000,
      })
      log.debug(`Resulting hash: [${codeHash}]`)

      const hash: string = keccak256(dummyContractBytecode.toString('hex'))

      remove0x(codeHash).should.equal(hash, 'Incorrect code hash!')
    })
  })

  describe('ovmEXTCODECOPY', async () => {
    it('properly gets all contract code via EXTCODECOPY', async () => {
      const methodId: string = ethereumjsAbi
        .methodID('ovmEXTCODECOPY', [])
        .toString('hex')

      const address: string = '00'.repeat(12) + remove0x(dummyContractAddress)
      const index: string = '00'.repeat(32)
      const length: string = bufferUtils
        .numberToBuffer(dummyContractBytecode.length)
        .toString('hex')
      const encodedParams: string = `${address}${index}${length}`

      const data: string = `0x${methodId}${remove0x(encodedParams)}`

      const code: string = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000_000,
      })
      log.debug(`Resulting code: [${code}]`)

      const codeBuff: Buffer = hexStrToBuf(code)
      codeBuff.should.eql(dummyContractBytecode, 'Incorrect code!')
    })
  })
})
