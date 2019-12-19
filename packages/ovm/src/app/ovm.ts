/* External Imports */
import { BigNumber, getLogger } from '@pigi/core-utils'
/* Internal Imports */
import {
  RollupStateMachine,
  Transaction,
  Address,
  StorageSlot,
  StorageValue,
  TransactionResult,
} from '../types'

/* Constants */
const DEPLOY_CONTRACT_ENTRYPOINT = '0x42'

const log = getLogger('data-types', true)

const fakeTxResult: TransactionResult = {
  transactionNumber: new BigNumber(10),
  transaction: {
    ovmEntrypoint: 'entrypoint',
    ovmCalldata: 'calldata',
  },
  updatedStorage: [
    {
      contractAddress: '0x1234',
      storageSlot: '0x4321',
      storageValue: '0x00',
    },
  ],
}

class EvmStateMachine implements RollupStateMachine {
  public async getStorageAt(
    targetContract: Address,
    targetStorageKey: StorageSlot
  ): Promise<StorageValue> {
    return 'getStorageAt(...) UNIMPLEMENTED'
  }

  public async applyTransaction(
    transaction: Transaction
  ): Promise<TransactionResult> {
    // Check if transaction is a contract deployment or normal tx
    // IF deploy contract: register the new contract deployment immediately on Ethereum. Wait until Ethereum confirmation to add ct to off-chain state.
    // ELSE apply tx: apply the tx off-chain.
    return fakeTxResult
  }

  public async getTransactionResultsSince(
    transactionNumber: BigNumber
  ): Promise<TransactionResult[]> {
    return [fakeTxResult]
  }
}
