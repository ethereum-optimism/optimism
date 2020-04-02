import '../setup'

/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  getCurrentTime,
  remove0x,
  add0x,
  TestUtils,
  bufToHexString,
  ZERO_ADDRESS,
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
  didCreateSucceed,
  gasLimit,
  executeOVMCall,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { GAS_LIMIT, DEFAULT_OPCODE_WHITELIST_MASK } from '../../src/app'
import { cloneDeep, fromPairs } from 'lodash'
import { L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS } from '../../src/app/constants'

export const abi = new ethers.utils.AbiCoder()

const log = getLogger('l2-to-l1-messaging', true)

/***********
 * HELPERS *
 **********/

/**
 * Override the ABI description of a particular function, changing it's `constant` & `outputs` values.
 * @param {Array} an abi object.
 * @param {string} the name of the function we would like to change.
 * @param {Object} an object containing the new `constant` & `outputs` values.
 */
function overrideAbiFunctionData(
  abiDefinition: any,
  functionName: string,
  functionData: { constant: boolean; outputs: any[] }
): void {
  for (const functionDefinition of abiDefinition) {
    if (functionDefinition.name === functionName) {
      functionDefinition.constant = functionData.constant
      functionDefinition.outputs = functionData.outputs
    }
  }
}

/**
 * Use executeTransaction with `eth_call`.
 * @param {ethers.Contract} an ExecutionManager contract instance used for it's address & provider.
 * @param {Array} an array of parameters which should be fed into `executeTransaction(...)`.
 * @param {OutputTypes} an array ABI types which should be used to decode the output of the call.
 */
function callExecutionManagerExecuteTransaction(
  executionManager: Contract,
  parameters: any[],
  outputTypes: any[]
): Promise<any[]> {
  const modifiedAbi = cloneDeep(ExecutionManager.abi)
  overrideAbiFunctionData(modifiedAbi, 'executeTransaction', {
    constant: true,
    outputs: outputTypes,
  })
  const callableExecutionManager = new Contract(
    executionManager.address,
    modifiedAbi,
    executionManager.provider
  )
  return callableExecutionManager.executeTransaction.apply(null, parameters)
}

/*********
 * TESTS *
 *********/

describe('Execution Manager -- L1 <-> L2 Opcodes', () => {
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

  describe('OVM L2 -> L1 message passer', () => {
    it(`Should emit the right msg.sender and calldata when an L2->L1 call is made`, async () => {
      const bytesToSendToL1 = '0x123412341234deadbeef'
      const passMessageToL1MethodId = bufToHexString(
        ethereumjsAbi.methodID('passMessageToL1', ['bytes'])
      )
      const txData: string =
        encodeMethodId('executeTransactionRaw') +
        encodeRawArguments([
          getCurrentTime(),
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

  describe('L1 Message Sender', () => {
    const getL1MessageSenderMethodId = bufToHexString(
      ethereumjsAbi.methodID('getL1MessageSender', [])
    )

    it('should return the l1 message sender provided', async () => {
      const l1MessageSenderPrecompileAddr =
        '0x4200000000000000000000000000000000000001'
      const testL1MsgSenderAddress = '0x' + '01'.repeat(20)

      const callResult = await callExecutionManagerExecuteTransaction(
        executionManager,
        [
          getCurrentTime(),
          0,
          l1MessageSenderPrecompileAddr,
          getL1MessageSenderMethodId,
          ZERO_ADDRESS,
          testL1MsgSenderAddress,
          true,
        ],
        ['address']
      )
      callResult.should.equal(
        testL1MsgSenderAddress,
        'The returned l1 message sender address should equal the one given!'
      )
    })

    it('should fail if the transaction CALLER is set to a value other than the ZERO_ADDRESS', async () => {
      const l1MessageSenderPrecompileAddr =
        '0x4200000000000000000000000000000000000001'
      const testL1MsgSenderAddress = '0x' + '01'.repeat(20)

      let failed = false
      try {
        const callResult = await callExecutionManagerExecuteTransaction(
          executionManager,
          [
            0,
            0,
            l1MessageSenderPrecompileAddr,
            getL1MessageSenderMethodId,
            '0x' + '66'.repeat(20),
            testL1MsgSenderAddress,
            true,
          ],
          ['address']
        )
      } catch (e) {
        log.debug(JSON.stringify(e) + '  ' + e.stack)
        failed = true
      }

      failed.should.equal(true, `This call should have reverted!`)
    })

    it('should fail if the L1MessageSender is set to the ZERO_ADDRESS (ie. there is no L1 message sender)', async () => {
      const l1MessageSenderPrecompileAddr =
        '0x4200000000000000000000000000000000000001'

      let failed = false
      try {
        const callResult = await callExecutionManagerExecuteTransaction(
          executionManager,
          [
            0,
            0,
            l1MessageSenderPrecompileAddr,
            getL1MessageSenderMethodId,
            '0x' + '66'.repeat(20),
            ZERO_ADDRESS,
            true,
          ],
          ['address']
        )
      } catch (e) {
        log.debug(JSON.stringify(e) + '  ' + e.stack)
        failed = true
      }

      failed.should.equal(true, `This call should have reverted!`)
    })
  })
})
