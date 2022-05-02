/* Imports: External */
import { BigNumber, ethers } from 'ethers'
import { serialize } from '@ethersproject/transactions'
import { padHexString } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { TransportDB } from '../../../db/transport-db'
import {
  DecodedSequencerBatchTransaction,
  StateRootEntry,
  TransactionEntry,
} from '../../../types'
import { parseSignatureVParam } from '../../../utils'
import { MissingElementError } from './errors'

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
      BigNumber.from(transaction.blockNumber).toNumber() - 1

    // We make the assumption that you don't need to sync the genesis block
    if (transactionIndex < 0) {
      throw new Error('should not happen, attempted to sync genesis block')
    }

    let transactionEntry: Partial<TransactionEntry> = {
      // Legacy support.
      index: transactionIndex,
      value: transaction.value,
      batchIndex: null,
      blockNumber: BigNumber.from(transaction.l1BlockNumber).toNumber(),
      timestamp: BigNumber.from(transaction.l1Timestamp).toNumber(),
      queueOrigin: transaction.queueOrigin,
      confirmed: false,
    }

    if (transaction.queueOrigin === 'sequencer') {
      const decodedTransaction: DecodedSequencerBatchTransaction = {
        sig: {
          v: parseSignatureVParam(transaction.v, chainId),
          r: padHexString(transaction.r, 32),
          s: padHexString(transaction.s, 32),
        },
        value: transaction.value,
        gasLimit: BigNumber.from(transaction.gas).toString(),
        gasPrice: BigNumber.from(transaction.gasPrice).toString(),
        nonce: BigNumber.from(transaction.nonce).toString(),
        target: transaction.to,
        data: transaction.input,
      }

      transactionEntry = {
        ...transactionEntry,
        gasLimit: BigNumber.from(0).toString(),
        target: ethers.constants.AddressZero,
        origin: null,
        data: serialize(
          {
            value: transaction.value,
            gasLimit: transaction.gas,
            gasPrice: transaction.gasPrice,
            nonce: transaction.nonce,
            to: transaction.to,
            data: transaction.input,
            chainId,
          },
          {
            v: BigNumber.from(transaction.v).toNumber(),
            r: padHexString(transaction.r, 32),
            s: padHexString(transaction.s, 32),
          }
        ),
        decoded: decodedTransaction,
        queueIndex: null,
      }
    } else {
      transactionEntry = {
        ...transactionEntry,
        gasLimit: BigNumber.from(transaction.gas).toString(),
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
    if (entry.transactionEntry.index > 0) {
      const prevTransactionEntry = await db.getUnconfirmedTransactionByIndex(
        entry.transactionEntry.index - 1
      )

      // We should *always* have a previous transaction here.
      if (prevTransactionEntry === null) {
        throw new MissingElementError('SequencerTransaction')
      }
    }

    // Having separate indices for confirmed/unconfirmed means we never have to worry about
    // accidentally overwriting a confirmed transaction with an unconfirmed one. Unconfirmed
    // transactions are purely extra information.
    await db.putUnconfirmedTransactionEntries([entry.transactionEntry])
    await db.putUnconfirmedStateRootEntries([entry.stateRootEntry])
  },
}
