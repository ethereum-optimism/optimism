/* External Imports */
import { Signer } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

/* Internal Imports */
import { CanonicalTransactionChainContract } from '.'

type Address = string

export class BatchSubmitter {
    txChain: CanonicalTransactionChainContract
    signer: Signer

    constructor(canonicalTransactionChainAddress: Address, signer: Signer) {
        this.txChain = new CanonicalTransactionChainContract(
          canonicalTransactionChainAddress,
          getContractInterface('OVM_CanonicalTransactionChain'),
          signer
        )
        this.signer = signer
    }

    async submitNextBatch():Promise<void> {
        const data = '0x' + '12'.repeat(32)
        const timestamp = (await this.signer.provider.getBlock('latest')).timestamp - 10
        const blockNumber = (await this.signer.provider.getBlockNumber()) - 1

        const txRes = await this.txChain.appendSequencerBatch({
            shouldStartAtBatch: 0,
            totalElementsToAppend: 1,
            contexts: [
              {
                numSequencedTransactions: 1,
                numSubsequentQueueTransactions: 0,
                timestamp: timestamp,
                blockNumber: blockNumber,
              },
            ],
            transactions: [data]
        })
        console.log(txRes)
    }
}
