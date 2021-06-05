/* External Imports */
import { Contract, BigNumber, ethers } from 'ethers'
import {
  TransactionResponse,
  TransactionRequest,
} from '@ethersproject/abstract-provider'
import { JsonRpcProvider } from '@ethersproject/providers'
import { keccak256 } from 'ethers/lib/utils'
import {
  AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
  encodeHex,
  remove0x,
} from '@eth-optimism/core-utils'

export { encodeAppendSequencerBatch, BatchContext, AppendSequencerBatchParams }

/*
 * OVM_CanonicalTransactionChainContract is a wrapper around a normal Ethers contract
 * where the `appendSequencerBatch(...)` function uses a specialized encoding for improved efficiency.
 */
export class CanonicalTransactionChainContract extends Contract {
  public customPopulateTransaction = {
    appendSequencerBatch: async (
      batch: AppendSequencerBatchParams
    ): Promise<ethers.PopulatedTransaction> => {
      const nonce = await this.signer.getTransactionCount()
      const to = this.address
      const data = getEncodedCalldata(batch)
      const gasLimit = await this.signer.provider.estimateGas({
        to,
        from: await this.signer.getAddress(),
        data,
      })
      const value = 0

      return {
        nonce,
        to,
        data,
        gasLimit,
      }
    },
  }
  public async appendSequencerBatch(
    batch: AppendSequencerBatchParams,
    options?: TransactionRequest
  ): Promise<TransactionResponse> {
    return appendSequencerBatch(this, batch, options)
  }
}

/**********************
 * Internal Functions *
 *********************/

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'

const appendSequencerBatch = async (
  OVM_CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams,
  options?: TransactionRequest
): Promise<TransactionResponse> => {
  return OVM_CanonicalTransactionChain.signer.sendTransaction({
    to: OVM_CanonicalTransactionChain.address,
    data: getEncodedCalldata(batch),
    ...options,
  })
}

const getEncodedCalldata = (batch: AppendSequencerBatchParams): string => {
  const methodId = keccak256(
    Buffer.from(APPEND_SEQUENCER_BATCH_METHOD_ID)
  ).slice(2, 10)
  const calldata = encodeAppendSequencerBatch(batch)
  return '0x' + remove0x(methodId) + remove0x(calldata)
}

const encodeBatchContext = (context: BatchContext): string => {
  return (
    encodeHex(context.numSequencedTransactions, 6) +
    encodeHex(context.numSubsequentQueueTransactions, 6) +
    encodeHex(context.timestamp, 10) +
    encodeHex(context.blockNumber, 10)
  )
}
