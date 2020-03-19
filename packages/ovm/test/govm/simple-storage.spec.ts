import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import { getWallets } from 'ethereum-waffle'
import { getLogger, add0x } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, ethers } from 'ethers'
import { TransactionReceipt, JsonRpcProvider } from 'ethers/providers'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'

/* Internal Imports */
import {
  ensureGovmIsConnected,
  manuallyDeployOvmContract,
  getUnsignedTransactionCalldata,
  executeUnsignedEOACall,
} from '../helpers'
import { CHAIN_ID, GAS_LIMIT } from '../../src/app'

const log = getLogger('simple-storage', true)

/*********
 * TESTS *
 *********/

describe('SimpleStorage', () => {
  const provider = new JsonRpcProvider()
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple storage contract
  let executionManager: Contract
  let simpleStorage: ContractFactory
  let simpleStorageOvmAddress: Address
  const setStorageMethodId: string = ethereumjsAbi
    .methodID('setStorage', [])
    .toString('hex')
  const getStorageMethodId: string = ethereumjsAbi
    .methodID('getStorage', [])
    .toString('hex')

  /* Deploy contracts before each test */
  beforeEach(async () => {
    await ensureGovmIsConnected(provider)
    // Before each test let's deploy a fresh ExecutionManager and SimpleStorage
    // Deploy ExecutionManager the normal way
    executionManager = new ethers.Contract(
      process.env.EXECUTION_MANAGER_ADDRESS,
      ExecutionManager.abi,
      wallet
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
    const innerCallData: string = add0x(`${setStorageMethodId}${slot}${value}`)
    return executeUnsignedEOACall(
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
      const slot = '99'.repeat(32)
      const value = '01'.repeat(32)
      const reciept = await setStorage(slot, value)
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
        [0, 0, transaction.nonce, transaction.to, transaction.data, v, r, s]
      )

      const result = await executionManager.provider.call({
        to: executionManager.address,
        data: add0x(callData),
        gasLimit: 6_700_000,
      })
      result.should.equal(add0x(value))
    })
  })
})
