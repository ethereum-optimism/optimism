import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  getCurrentTime,
  add0x,
  bufToHexString,
  ZERO_ADDRESS,
  NULL_ADDRESS,
  abi,
} from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import {
  GAS_LIMIT,
  L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS,
  Address,
  manuallyDeployOvmContract,
  addressToBytes32Address,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  callExecutionManagerExecuteTransaction,
  encodeFunctionData
} from '../../../test-helpers'

/* Logging */
const log = getLogger('l2-to-l1-messaging', true)

/* Tests */
describe('Execution Manager -- L1 <-> L2 Opcodes', () => {
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
  let SimpleCall: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    SimpleCall = await ethers.getContractFactory('SimpleCall')
  })

  let executionManager: Contract
  before(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [resolver.addressResolver.address, NULL_ADDRESS, GAS_LIMIT],
      }
    )
  })

  let callContractAddress: Address
  before(async () => {
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
      const callBytes = encodeFunctionData(
        'makeCall',
        [
          addressToBytes32Address(L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS),
          ethereumjsAbi.methodID('passMessageToL1', ['bytes']),
          abi.encode(['bytes'], [bytesToSendToL1]),
        ]
      )

      const data = executionManager.interface.encodeFunctionData(
        'executeTransaction',
        [
          getCurrentTime(),
          0,
          callContractAddress,
          callBytes,
          ZERO_ADDRESS,
          ZERO_ADDRESS,
          true,
        ]
      )

      const txResult = await wallet.sendTransaction({
        to: executionManager.address,
        data: add0x(data),
        gasLimit: GAS_LIMIT,
      })

      const receipt = await provider.getTransactionReceipt(txResult.hash)
      const txLogs = receipt.logs

      const l2ToL1EventTopic = ethers.utils.id(
        'L2ToL1Message(uint256,address,bytes)'
      )
      const crossChainMessageEvent = txLogs.find((logged) => {
        // console.log(logged)
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
