/* External Imports */
import { Contract, ethers } from 'ethers'
import {
  TransactionResponse,
  TransactionRequest,
} from '@ethersproject/abstract-provider'
import {
  AppendSequencerBatchParams,
  BatchContext,
  encodeAppendSequencerBatch,
  sequencerBatch,
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

const getEncodedCalldata = (params: AppendSequencerBatchParams): string => {
  return sequencerBatch.encode(params)
}
