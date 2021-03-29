/* Imports: External */
import { ctcCoder } from '@eth-optimism/core-utils'
import { BigNumber, constants, ethers } from 'ethers'

/* Imports: Internal */
import { TransportDB } from '../../../db/transport-db'
import {
  DecodedSequencerBatchTransaction,
  StateRootEntry,
  TransactionEntry,
} from '../../../types'
import {
  padHexString,
  SEQUENCER_ENTRYPOINT_ADDRESS,
  SEQUENCER_GAS_LIMIT,
} from '../../../utils'

export const handleSequencerBlock = {
  parseBlock: async (
    block: any,
    chainId: number
  ): Promise<{
    transactionEntry: TransactionEntry
    stateRootEntry: StateRootEntry
  }> => {
    const transaction = block.transactions[0]
    const transactionIndex =
      transaction.index === null || transaction.index === undefined
        ? BigNumber.from(transaction.blockNumber).toNumber() - 1
        : BigNumber.from(transaction.index).toNumber()

    let transactionEntry: Partial<TransactionEntry> = {
      // Legacy support.
      index: transactionIndex,
      batchIndex: null,
      blockNumber: BigNumber.from(transaction.l1BlockNumber).toNumber(),
      timestamp: BigNumber.from(transaction.l1Timestamp).toNumber(),
      queueOrigin: transaction.queueOrigin,
      type: parseTxType(transaction.txType),
      confirmed: false,
    }

    if (transaction.queueOrigin === 'sequencer') {
      const decodedTransaction: DecodedSequencerBatchTransaction = {
        sig: {
          v: BigNumber.from(transaction.v).toNumber() - 2 * chainId - 35,
          r: padHexString(transaction.r, 32),
          s: padHexString(transaction.s, 32),
        },
        gasLimit: BigNumber.from(transaction.gas).toNumber(),
        gasPrice: BigNumber.from(transaction.gasPrice).toNumber(), // ?
        nonce: BigNumber.from(transaction.nonce).toNumber(),
        target: transaction.to || constants.AddressZero, // ?
        data: transaction.input,
        type: transaction.txType,
      }

      transactionEntry = {
        ...transactionEntry,
        gasLimit: SEQUENCER_GAS_LIMIT, // ?
        target: SEQUENCER_ENTRYPOINT_ADDRESS,
        origin: null,
        data: maybeEncodeSequencerBatchTransaction(
          decodedTransaction,
          transaction.txType
        ),
        decoded: decodedTransaction,
        queueIndex: null,
      }
    } else {
      transactionEntry = {
        ...transactionEntry,
        gasLimit: BigNumber.from(transaction.gas).toNumber(),
        target: ethers.utils.getAddress(transaction.to),
        origin: ethers.utils.getAddress(transaction.l1TxOrigin),
        data: transaction.input,
        decoded: null,
        queueIndex:
          transaction.queueIndex === null ||
          transaction.queueIndex === undefined
            ? BigNumber.from(transaction.nonce).toNumber()
            : BigNumber.from(transaction.queueIndex).toNumber(),
      }
    }

    const stateRootEntry: StateRootEntry = {
      index: transactionIndex,
      batchIndex: null,
      value: block.stateRoot,
      confirmed: false,
    }

    return {
      transactionEntry: transactionEntry as TransactionEntry, // Not the cleanest thing in the world. Could be improved.
      stateRootEntry,
    }
  },
  storeBlock: async (
    entry: {
      transactionEntry: TransactionEntry
      stateRootEntry: StateRootEntry
    },
    db: TransportDB
  ): Promise<void> => {
    // Having separate indices for confirmed/unconfirmed means we never have to worry about
    // accidentally overwriting a confirmed transaction with an unconfirmed one. Unconfirmed
    // transactions are purely extra information.
    await db.putUnconfirmedTransactionEntries([entry.transactionEntry])
    await db.putUnconfirmedStateRootEntries([entry.stateRootEntry])
  },
}

/**
 * Attempts to encode a sequencer batch transaction.
 * @param transaction Transaction to encode.
 * @param type Transaction type.
 */
const maybeEncodeSequencerBatchTransaction = (
  transaction: DecodedSequencerBatchTransaction,
  type: 'EIP155' | 'EthSign' | null
): string => {
  if (type === 'EIP155') {
    return ctcCoder.eip155TxData.encode(transaction).toLowerCase()
  } else if (type === 'EthSign') {
    return ctcCoder.ethSignTxData.encode(transaction).toLowerCase()
  } else {
    // Throw?
    return
  }
}

/**
 * Handles differences between the sequencer's enum strings and our own.
 * Will probably want to move this into core-utils eventually.
 * @param type Sequencer transaction type to parse.
 */
const parseTxType = (
  type: 'EIP155' | 'EthSign' | null
): 'EIP155' | 'ETH_SIGN' | null => {
  if (type === 'EthSign') {
    return 'ETH_SIGN'
  } else {
    return type
  }
}
