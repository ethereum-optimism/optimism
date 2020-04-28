import {
  abi,
  add0x,
  bufToHexString,
  hexStrToBuf,
  remove0x,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
/* Contract Imports */
import { TransactionReceipt } from 'ethers/providers'
import {
  convertInternalLogsToOvmLogs,
  getSuccessfulOvmTransactionMetadata,
  OvmTransactionMetadata,
  revertMessagePrefix,
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
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [ALICE], 0],
        [CODE_CONTRACT, 'EventFromAlice()', [], 1],
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [BOB], 2],
        [CODE_CONTRACT, 'EventFromBob()', [], 3],
      ].map((args) => buildLog.apply(null, args)),
      EXECUTION_MANAGER_ADDRESS
    ).should.deep.eq(
      [
        [ALICE, 'EventFromAlice()', [], 0],
        [BOB, 'EventFromBob()', [], 1],
      ].map((args) => buildLog.apply(null, args))
    )
  })
})

describe('getSuccessfulOvmTransactionMetadata', () => {
  it('should return transaction metadata from calls from externally owned accounts', async () => {
    const transactionReceipt: TransactionReceipt = {
      byzantium: true,
      logs: [
        [EXECUTION_MANAGER_ADDRESS, 'ActiveContract(address)', [ALICE]],
        [
          EXECUTION_MANAGER_ADDRESS,
          'CallingWithEOA(address,address)',
          [ALICE, CONTRACT],
        ],
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

    getSuccessfulOvmTransactionMetadata(transactionReceipt).should.deep.eq({
      ovmCreatedContractAddress: CONTRACT,
      ovmFrom: ALICE,
      ovmTo: CONTRACT,
      ovmTxSucceeded: true,
    })
  })
})
