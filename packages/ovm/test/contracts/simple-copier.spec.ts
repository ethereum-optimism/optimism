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

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleCopier from '../../build/contracts/SimpleCopier.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
} from '../helpers'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('simple-copier', true)

/*********
 * TESTS *
 *********/

describe('SimpleStorage', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let simpleCopier: ContractFactory
  let simpleCopierOvmAddress: Address
  const simpleCopierBytecode: Buffer = Buffer.from(
    SimpleCopier.evm.deployedBytecode.object,
    'hex'
  )

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and SimpleCopier

    // Set the ABI to consider `executeCall()` to be a "constant" function so that we can use web3.call(executeCall(...))
    // not just web3.applyTransaction(...)
    const executeCallAbi = ExecutionManager.abi.filter(
      (x) => x.name === 'executeCall'
    )[0]
    executeCallAbi.constant = true

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      new Array(2).fill('0x' + '00'.repeat(20)),
      {
        gasLimit: 6700000,
      }
    )

    // Deploy SimpleCopier with the ExecutionManager
    simpleCopierOvmAddress = await manuallyDeployOvmContract(
      provider,
      executionManager,
      SimpleCopier,
      [executionManager.address]
    )
    // Also set our simple copier Ethers contract so we can generate unsigned transactions
    simpleCopier = new ContractFactory(
      SimpleCopier.abi as any,
      SimpleCopier.bytecode
    )
  })

  describe('getContractCodeSize', async () => {
    it('properly gets contract code size for the contract we expect', async () => {
      // Generate our tx calldata
      const calldata = getUnsignedTransactionCalldata(
        simpleCopier,
        'getContractCodeSize',
        [add0x(simpleCopierOvmAddress)]
      )

      // Call through our ExecutionManager
      const result = await executionManager.executeCall(
        {
          ovmEntrypoint: simpleCopierOvmAddress,
          ovmCalldata: calldata,
        },
        0,
        0
      )
      const codeSize: number = new BigNumber(remove0x(result), 'hex').toNumber()
      codeSize.should.equal(
        simpleCopierBytecode.length,
        'Incorrect bytecode length!'
      )
    })
  })

  describe('getContractCodeHash', async () => {
    it('properly gets contract code hash for the contract we expect', async () => {
      // Generate our tx calldata
      const calldata = getUnsignedTransactionCalldata(
        simpleCopier,
        'getContractCodeHash',
        [add0x(simpleCopierOvmAddress)]
      )

      // Call through our ExecutionManager
      const codeHash = await executionManager.executeCall(
        {
          ovmEntrypoint: simpleCopierOvmAddress,
          ovmCalldata: calldata,
        },
        0,
        0
      )

      const hash: string = keccak256(simpleCopierBytecode.toString('hex'))

      remove0x(codeHash).should.equal(hash, 'Incorrect code hash!')
    })
  })

  describe('getContractCodeCopy', async () => {
    it('properly gets all contract code via CODECOPY', async () => {
      // Generate our tx calldata
      const calldata = getUnsignedTransactionCalldata(
        simpleCopier,
        'getContractCodeCopy',
        [add0x(simpleCopierOvmAddress), 0, simpleCopierBytecode.length]
      )

      // Call through our ExecutionManager
      const code = await executionManager.executeCall(
        {
          ovmEntrypoint: simpleCopierOvmAddress,
          ovmCalldata: calldata,
        },
        0,
        0
      )

      const decoded: string = abi.decode(['bytes'], code)[0]
      const codeBuff: Buffer = Buffer.from(remove0x(decoded), 'hex')

      codeBuff.should.eql(simpleCopierBytecode, 'Incorrect code!')
    })
  })
})
