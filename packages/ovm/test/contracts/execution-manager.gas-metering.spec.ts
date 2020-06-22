import '../setup'

/* External Imports */
import {
  Address,
  GAS_LIMIT,
  CHAIN_ID,
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_ETHNODE_GAS_LIMIT,
  getUnsignedTransactionCalldata,
} from '@eth-optimism/rollup-core'
import {
  getLogger,
  padToLength,
  ZERO_ADDRESS,
  TestUtils,
  getCurrentTime,
} from '@eth-optimism/core-utils'

import {
  ExecutionManagerContractDefinition as ExecutionManager,
  FullStateManagerContractDefinition as StateManager,
  TestDummyContractDefinition as DummyContract,
} from '@eth-optimism/rollup-contracts'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import { manuallyDeployOvmContract, ZERO_UINT } from '../helpers'
import { exec } from 'child_process'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('execution-manager-gas-metering', true)

/*************
 * CONSTANTS *
 *************/

const OVM_TX_MAX_GAS = 2000000000
const GAS_RATE_LIMIT_EPOCH_LENGTH = 1000
const MAX_SEQUENCED_GAS_PER_EPOCH = 2000000000

/*********
 * TESTS *
 *********/

const unsignedCallMethodId: string = ethereumjsAbi
  .methodID('executeTransaction', [])
  .toString('hex')

describe.only('Execution Manager -- Gas Metering', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  // Create pointers to our execution manager & simple copier contract
  let executionManager: Contract
  let stateManager: Contract
  let dummyContract: ContractFactory
  let dummyContractAddress: Address

  beforeEach(async () => {
    // Before each test let's deploy a fresh ExecutionManager and DummyContract

    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
    // Set the state manager as well
    stateManager = new Contract(
      await executionManager.getStateManagerAddress(),
      StateManager.abi,
      wallet
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

  const assertEOACallRevertsWithMsg = async (call: () => Promise<any>,expectedEventMsg: string ) => {
    const tx = await call()
    const reciept = await provider.getTransactionReceipt(tx.hash)
        const revertTopic = ethers.utils.id(
          'EOACallRevert(bytes)'
        )
        const revertEvent = reciept.logs.find((logged) => {
          return logged.topics.includes(revertTopic)
        })
        revertEvent.data.should.equal(
          abi.encode(
            ['bytes'],
            [Buffer.from(expectedEventMsg)]
          )
        )
  }

  const dummyCalldata = '0x123412341234'
  describe('Per-transaction gas limit', async () => {
    it('Should emit EOACallRevert event if the gas limit is higher than the max allowed', async () =>{
      assertEOACallRevertsWithMsg(
        () => {
          return executionManager.executeTransaction(
            1,
            ZERO_UINT,
            dummyContractAddress,
            dummyCalldata,
            wallet.address,
            ZERO_ADDRESS,
            OVM_TX_MAX_GAS + 1,
            false
          )
        },
        'Transaction gas limit exceeds max OVM tx gas limit'
      )
    })
  })
  describe.only('Multi-transaction gas rate limiting', async () => {
    it('For two transactions with gas limits equalling the epoch limit, the second should fail', async () => {
      // first one should not revert
      const tx = await executionManager.executeTransaction(
        1,
        ZERO_UINT,
        dummyContractAddress,
        dummyCalldata,
        wallet.address,
        ZERO_ADDRESS,
        MAX_SEQUENCED_GAS_PER_EPOCH,
        false
      )
      const reciept = await provider.getTransactionReceipt(tx.hash)
      const revertTopic = ethers.utils.id(
        'EOACallRevert(bytes)'
      )
      const revertEvent = reciept.logs.find((logged) => {
        return logged.topics.includes(revertTopic)
      })
      console.log(revertEvent)
      revertEvent.should.equal(undefined) // should not be found
    })
  })
})

