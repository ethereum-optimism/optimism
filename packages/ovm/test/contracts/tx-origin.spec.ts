import '../setup'

/* External Imports */
import { getLogger, add0x, getCurrentTime } from '@eth-optimism/core-utils'
import {
  Address,
  CHAIN_ID,
  GAS_LIMIT,
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_ETHNODE_GAS_LIMIT,
  getUnsignedTransactionCalldata,
} from '@eth-optimism/rollup-core'
import {
  ExecutionManagerContractDefinition as ExecutionManager,
  FullStateManagerContractDefinition as StateManager,
  TestSimpleTxOriginContractDefinition as SimpleTxOrigin,
} from '@eth-optimism/rollup-contracts'

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract, ContractFactory } from 'ethers'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import {
  addressToBytes32Address,
  manuallyDeployOvmContract,
  signTransation,
} from '../helpers'

const log = getLogger('simple-storage', true)

/*********
 * TESTS *
 *********/

describe('SimpleTxOrigin', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  let executionManager: Contract
  let simpleTxOrigin: ContractFactory
  let simpleTxOriginOvmAddress: Address

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and SimpleTxOrigin
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )

    // Deploy SimpleTxOrigin with the ExecutionManager
    simpleTxOriginOvmAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleTxOrigin,
      [executionManager.address]
    )
    // Also set our simple storage ethers contract so we can generate unsigned transactions
    simpleTxOrigin = new ContractFactory(
      SimpleTxOrigin.abi as any, // For some reason the ABI type definition is not accepted
      SimpleTxOrigin.bytecode
    )
  })

  describe('getOrigin', async () => {
    it('correctly gets the origin address', async () => {
      const getStorageMethodId: string = ethereumjsAbi
        .methodID('getOrigin', [])
        .toString('hex')

      const innerCallData: string = add0x(`${getStorageMethodId}`)
      const stateManagerAddress = await executionManager.getStateManagerAddress()
      const stateManager = new Contract(
        stateManagerAddress,
        StateManager.abi,
        wallet
      )
      const nonce = await stateManager.getOvmContractNonce(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: simpleTxOriginOvmAddress,
        value: 0,
        data: innerCallData,
        chainId: CHAIN_ID,
      }
      const [v, r, s] = await signTransation(wallet, transaction)
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
        gasLimit: 6_700_000,
      })

      result.should.equal(addressToBytes32Address(wallet.address))
    })
  })
})
