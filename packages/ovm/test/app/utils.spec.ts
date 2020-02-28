import { Contract, ethers } from 'ethers'
import { add0x, ZERO_ADDRESS, TestUtils } from '@eth-optimism/core-utils'
/* Contract Imports */
import { getWallets } from 'ethereum-waffle'
import { TransactionReceipt, JsonRpcProvider, Log } from 'ethers/providers'
import {
  convertInternalLogsToOvmLogs,
  getOvmTransactionMetadata,
} from '../../src/app'
import { buildLog } from '../helpers'

const ALICE = '0x0000000000000000000000000000000000000001'
const BOB = '0x0000000000000000000000000000000000000002'
const CONTRACT = '0x000000000000000000000000000000000000000C'
const CODE_CONTRACT = '0x00000000000000000000000000000000000000CC'
const CODE_CONTRACT_HASH = add0x('00'.repeat(32))
// We're not actually making any calls to the
// Execution manager so this can be the zero address
const EXECUTION_MANAGER_ADDRESS = ZERO_ADDRESS

describe('convertInternalLogsToOvmLogs', () => {
  it('should replace the address of the event with the address of the last active contract event', async () => {
    convertInternalLogsToOvmLogs(
      [
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [ALICE]],
        [EXECUTION_MANAGER_ADDRESS, 'EventFromAlice()', []],
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [BOB]],
        [EXECUTION_MANAGER_ADDRESS, 'EventFromBob()', []],
      ].map((args) => buildLog.apply(null, args))
    ).should.deep.eq(
      [
        [ALICE, 'EventFromAlice()', []],
        [BOB, 'EventFromBob()', []],
      ].map((args) => buildLog.apply(null, args))
    )
  })
})

describe('getOvmTransactionMetadata', () => {
  it('should return transaction metadata from calls from externally owned accounts', async () => {
    const transactionReceipt: TransactionReceipt = {
      byzantium: true,
      logs: [
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [ALICE]],
        [EXECUTION_MANAGER_ADDRESS, 'CallingWithEOA(address)', [ALICE]],
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [ALICE]],
        [EXECUTION_MANAGER_ADDRESS, 'EOACreatedContract(address)', [CONTRACT]],
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [CONTRACT]],
        [
          EXECUTION_MANAGER_ADDRESS,
          'CreatedContract(address,address,bytes32)',
          [CONTRACT, CODE_CONTRACT, CODE_CONTRACT_HASH],
        ],
      ].map((args) => buildLog.apply(null, args)),
    }

    getOvmTransactionMetadata(transactionReceipt).should.deep.eq({
      ovmCreatedContractAddress: CONTRACT,
      ovmFrom: ALICE,
      ovmTo: CONTRACT,
      ovmTxSucceeded: true,
    })
  })

  it('should return with ovmTxSucceeded equal to false if the transaction reverted', async () => {
    const transactionReceipt: TransactionReceipt = {
      byzantium: true,
      logs: [[EXECUTION_MANAGER_ADDRESS, 'EOACallRevert()', []]].map((args) =>
        buildLog.apply(null, args)
      ),
    }

    getOvmTransactionMetadata(transactionReceipt).ovmTxSucceeded.should.eq(
      false
    )
  })
})
