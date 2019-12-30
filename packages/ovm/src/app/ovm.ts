/* External Imports */
import {
  add0x,
  BigNumber,
  getLogger,
  logError,
  ONE,
  TWO,
  ZERO,
} from '@pigi/core-utils'
import {
  Address,
  ContractStorage,
  StorageElement,
  StorageSlot,
  StorageValue,
  Transaction,
  TransactionLog,
  TransactionReceipt,
  TransactionResult,
} from '@pigi/rollup-core'

import { Contract } from 'ethers'

/* Internal Imports */
import {
  RollupStateMachine,
  TransactionExecutionError,
  TransactionReceiptError,
} from '../types'
import { abiDecodeTransactionReceipt } from './serialization'

const log = getLogger('data-types')

export class OvmStateMachine implements RollupStateMachine {
  private transactionNumber: BigNumber

  constructor(
    private readonly stateManagerContract: Contract,
    private readonly executionManagerContract: Contract
  ) {
    // TODO: Make this persisted and recoverable
    this.transactionNumber = ZERO
  }

  public async getStorageAt(
    targetContract: Address,
    targetStorageKey: StorageSlot
  ): Promise<StorageValue> {
    try {
      return await this.stateManagerContract.getStorage(
        add0x(targetContract),
        add0x(targetStorageKey)
      )
    } catch (e) {
      logError(
        log,
        `Error fetching contract [${targetContract}] storage key [${targetStorageKey}]`,
        e
      )
      throw e
    }
  }

  public async applyTransaction(
    abiEncodedTransaction: string
  ): Promise<TransactionResult> {
    log.debug(`Received transaction to apply: [${abiEncodedTransaction}].`)

    // TODO: Some TX checking?

    let result: string
    try {
      log.debug(
        `Submitting tx to execution manager: tx: [${abiEncodedTransaction}]`
      )
      result = await this.executionManagerContract.executeTransaction(
        abiEncodedTransaction
      )
    } catch (e) {
      // If we're doing our job, this should not happen =D
      const msg: string = `Error executing tx: [${abiEncodedTransaction}]. Investigate Immediately!`
      logError(log, msg, e)
      throw new TransactionExecutionError(msg)
    }

    log.debug(
      `Received tx result: [${JSON.stringify(
        result
      )}] for Tx: [${abiEncodedTransaction}]`
    )

    try {
      const transactionReceipt: TransactionReceipt = abiDecodeTransactionReceipt(
        result
      )

      const updatedStorage: StorageElement[] = []
      const updatedContracts: ContractStorage[] = []
      for (const txLog of transactionReceipt.logs) {
        updatedStorage.push(this.getStorageElementFromLog(txLog))
        updatedContracts.push(this.getContractStorageFromLog(txLog))
      }

      this.transactionNumber = this.transactionNumber.add(ONE)
      return {
        transactionNumber: this.transactionNumber,
        transactionReceipt,
        abiEncodedTransaction,
        updatedStorage,
        updatedContracts,
      }
    } catch (e) {
      const msg: string = `Error parsing transaction result. Tx: [${JSON.stringify(
        abiEncodedTransaction
      )}], Result: [${JSON.stringify(result)}]`
      logError(log, msg, e)
      throw new TransactionReceiptError(msg)
    }
  }

  public async getTransactionResultsSince(
    transactionNumber: BigNumber
  ): Promise<TransactionResult[]> {
    // TODO: Get receipts from chain, parse logs, look up stored signed transaction and number from DB.
    // Dummy result:
    return [
      {
        transactionNumber: new BigNumber(10),
        transactionReceipt: {
          status: true,
          transactionHash:
            '0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b',
          transactionIndex: ONE,
          blockHash:
            '0x9fc76417374aa880d4449a1f7f31ec597f00b1f6f3dd2d66f4c9c6c445836d8b',
          blockNumber: ONE,
          contractAddress: '0x11f4d0A3c12e86B4b5F39B213F7E19D048276DAe',
          cumulativeGasUsed: TWO,
          gasUsed: TWO,
          logs: [],
        },
        abiEncodedTransaction: '0x0000000000000000',
        updatedStorage: [
          {
            contractAddress: '0x1234',
            storageSlot: '0x4321',
            storageValue: '0x00',
          },
        ],
        updatedContracts: [
          {
            contractAddress: '0x1234',
            contractNonce: ONE,
            contractCode: '',
          },
        ],
      },
    ]
  }

  private getStorageElementFromLog(txLog: TransactionLog): StorageElement {
    // TODO: Parse storage value from log
    return undefined
  }

  private getContractStorageFromLog(txLog: TransactionLog): ContractStorage {
    // TODO: Parse storage value from log
    return undefined
  }
}
