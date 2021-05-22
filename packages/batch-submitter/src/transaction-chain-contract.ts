/* External Imports */
import { Contract, BigNumber } from 'ethers'
import {
  TransactionResponse,
  TransactionRequest,
} from '@ethersproject/abstract-provider'
import { keccak256 } from 'ethers/lib/utils'
import { remove0x, encodeHex } from './utils'
import {
  //AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
} from '@eth-optimism/core-utils'

interface AppendSequencerBatchParams {
    chainId: number;
    shouldStartAtElement: number;
    totalElementsToAppend: number;
    contexts: BatchContext[];
    transactions: string[];
}

export { encodeAppendSequencerBatch, BatchContext, AppendSequencerBatchParams }

/*
 * OVM_CanonicalTransactionChainContract is a wrapper around a normal Ethers contract
 * where the `appendSequencerBatchByChainId(...)` function uses a specialized encoding for improved efficiency.
 */
export class CanonicalTransactionChainContract extends Contract {
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

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatchByChainId()'

const appendSequencerBatch = async (
  OVM_CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams,
  options?: TransactionRequest
): Promise<TransactionResponse> => {
  const methodId = keccak256(
    Buffer.from(APPEND_SEQUENCER_BATCH_METHOD_ID)
  ).slice(2, 10)
  var calldata = encodeAppendSequencerBatch(batch)
  //add chain id parameter before original batch
  calldata=encodeHex(batch.chainId, 64)+calldata
  return OVM_CanonicalTransactionChain.signer.sendTransaction({
    to: OVM_CanonicalTransactionChain.address,
    data: '0x' + methodId + calldata,
    ...options,
  })
}

const encodeBatchContext = (context: BatchContext): string => {
  return (
    encodeHex(context.numSequencedTransactions, 6) +
    encodeHex(context.numSubsequentQueueTransactions, 6) +
    encodeHex(context.timestamp, 10) +
    encodeHex(context.blockNumber, 10)
  )
}
