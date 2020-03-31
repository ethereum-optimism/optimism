import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger, add0x, getCurrentTime } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, ethers } from 'ethers'
import { TransactionReceipt } from 'ethers/providers'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
  executeTransaction,
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
} from '../helpers'
import {
  CHAIN_ID,
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
} from '../../src/app'

const log = getLogger('simple-storage', true)

/*********
 * TESTS *
 *********/

describe('SimpleStorage', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple storage contract
  let executionManager: Contract
  let simpleStorage: ContractFactory
  let simpleStorageOvmAddress: Address

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and SimpleStorage
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    // Deploy SimpleStorage with the ExecutionManager
    simpleStorageOvmAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleStorage,
      [executionManager.address]
    )
    // Also set our simple storage ethers contract so we can generate unsigned transactions
    simpleStorage = new ContractFactory(
      SimpleStorage.abi as any, // For some reason the ABI type definition is not accepted
      SimpleStorage.bytecode
    )
  })

  const setStorage = async (slot, value): Promise<TransactionReceipt> => {
    const setStorageMethodId: string = ethereumjsAbi
      .methodID('setStorage', [])
      .toString('hex')

    const innerCallData: string = add0x(`${setStorageMethodId}${slot}${value}`)
    return executeTransaction(
      executionManager,
      wallet,
      simpleStorageOvmAddress,
      innerCallData,
      true
    )
  }

  describe('setStorage', async () => {
    it('properly sets storage for the contract we expect', async () => {
      // create calldata vars
      const slot: string = '99'.repeat(32)
      const value: string = '01'.repeat(32)

      const reciept = await setStorage(slot, value)
    })
  })

  describe('getStorage', async () => {
    it('correctly loads a value after we store it', async () => {
      // Create the variables we will use for set & get storage
      const slot = '99'.repeat(32)
      const value = '01'.repeat(32)
      const reciept = await setStorage(slot, value)

      const getStorageMethodId: string = ethereumjsAbi
        .methodID('getStorage', [])
        .toString('hex')

      const innerCallData: string = add0x(`${getStorageMethodId}${slot}`)
      const nonce = await executionManager.getOvmContractNonce(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: simpleStorageOvmAddress,
        value: 0,
        data: innerCallData,
        chainId: CHAIN_ID,
      }
      const signedMessage = await wallet.sign(transaction)
      const [v, r, s] = ethers.utils.RLP.decode(signedMessage).slice(-3)
      const callData = getUnsignedTransactionCalldata(
        executionManager,
        'executeEOACall',
        [
          getCurrentTime(),
          0,
          transaction.nonce,
          transaction.to,
          transaction.data,
          v,
          r,
          s,
        ]
      )

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data: add0x(callData),
        gasLimit,
      })
      result.should.equal(add0x(value))
    })
  })
})
