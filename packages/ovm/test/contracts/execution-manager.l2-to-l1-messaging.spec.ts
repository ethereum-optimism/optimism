import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  add0x,
  bufToHexString,
} from '@eth-optimism/core-utils'

import { Contract, ContractFactory, ethers } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleCall from '../../build/contracts/SimpleCall.json'

/* Internal Imports */
import {
  manuallyDeployOvmContract,
  addressToBytes32Address,
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { GAS_LIMIT, DEFAULT_OPCODE_WHITELIST_MASK, L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS } from '../../src/app'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('l2-to-l1-messaging', true)

/*********
 * TESTS *
 *********/

describe('OVM L2 -> L1 message passer', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }) // debug: true, logger: console })
  const [wallet] = getWallets(provider)
  let executionManager: Contract
  let callContractAddress: Address

  beforeEach(async () => {
    // Deploy ExecutionManager the normal way
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      {
        gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
      }
    )

    // Deploy SimpleL2ToL1Sender with the ExecutionManager
    callContractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      SimpleCall,
      [executionManager.address]
    )
  })

  it(`Should emit the right msg.sender and calldata when an L2->L1 call is made`, async () => {
    const bytesToSendToL1 = '0x123412341234deadbeef'
    const passMessageToL1MethodId = bufToHexString(
      ethereumjsAbi.methodID('passMessageToL1', ['bytes'])
    )
    const txData: string =
      encodeMethodId('executeCall') +
      encodeRawArguments([
        0,
        0,
        addressToBytes32Address(callContractAddress),
        encodeMethodId('makeCall'),
        addressToBytes32Address(L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS),
        passMessageToL1MethodId,
        abi.encode(['bytes'], [bytesToSendToL1]),
      ])

    const txResult = await wallet.sendTransaction({
      to: executionManager.address,
      data: add0x(txData),
      gasLimit,
    })
    const receipt = await provider.getTransactionReceipt(txResult.hash)
    const txLogs = receipt.logs

    const l2ToL1EventTopic = ethers.utils.id(
      'L2ToL1Message(uint256,address,bytes)'
    )
    const crossChainMessageEvent = txLogs.find((logged) => {
      return logged.topics.includes(l2ToL1EventTopic)
    })

    crossChainMessageEvent.data.should.equal(
      abi.encode(
        ['uint', 'address', 'bytes'],
        [0, callContractAddress, bytesToSendToL1]
      )
    )
  })
})
