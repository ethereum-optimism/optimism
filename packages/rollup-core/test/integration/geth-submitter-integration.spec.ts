/* External Imports */
import { keccak256FromUtf8 } from '@eth-optimism/core-utils'
import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { CHAIN_ID, DefaultL2NodeService } from '../../src/app'
import { GethSubmission, L2NodeService } from '../../src/types'


// TODO: Can be used to submit Rollup Transactions to geth.
describe.skip('Optimistic Canonical Chain Batch Submitter', () => {
  let l2NodeService: L2NodeService

  beforeEach(async () => {
    // Address for wallet: 0x6a399F0A626A505e2F6C2b5Da181d98D722dC86D
    const wallet = new Wallet(
      'efb6aa1f37082ac40884a340684672ccbb5a4e6000860953afcf73c90c33e4f9',
      new JsonRpcProvider('http://127.0.0.1:8545', CHAIN_ID)
    )
    l2NodeService = new DefaultL2NodeService(wallet)
  })

  it('should send a batch to Geth', async () => {
    const gethSubmission: GethSubmission = {
      submissionNumber: 1,
      timestamp: 1,
      blockNumber: 1,
      rollupTransactions: [
        {
          l1RollupTxId: 2,
          indexWithinSubmission: 1,
          gasLimit: 0,
          nonce: 0,
          sender: Wallet.createRandom().address,
          target: Wallet.createRandom().address,
          calldata: keccak256FromUtf8('calldata'),
          l1Timestamp: 1,
          l1BlockNumber: 1,
          l1TxHash: keccak256FromUtf8('tx hash'),
          l1TxIndex: 0,
          l1TxLogIndex: 0,
          queueOrigin: 1,
        },
      ],
    }

    await l2NodeService.sendGethSubmission(gethSubmission)
  })
})
