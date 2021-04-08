/* Imports: Internal */
import { getContractFactory } from '@eth-optimism/contracts'
import { injectL2Context } from './shared/l2provider'

/* Imports: External */
import { Contract, Signer, Wallet, providers } from 'ethers'
import { expect } from 'chai'
import { sleep } from './shared/utils'

// This test ensures that the transactions which get `enqueue`d get
// added to the L2 blocks by the Sync Service (which queries the DTL)
describe('Queue Ingestion', () => {
  const RETRIES = 20
  const numTxs = 5
  let startBlock: number
  let endBlock: number

  let l1Signer: Signer
  let l2Provider: providers.JsonRpcProvider

  let addressResolver: Contract
  let canonicalTransactionChain: Contract

  const receipts = []
  before(async () => {
    const httpPort = 8545
    const l1HttpPort = 9545
    l2Provider = injectL2Context(
      new providers.JsonRpcProvider(`http://localhost:${httpPort}`)
    )
    l1Signer = new Wallet(
      '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      new providers.JsonRpcProvider(`http://localhost:${l1HttpPort}`)
    )

    const addressResolverAddress = '0x5FbDB2315678afecb367f032d93F642f64180aa3'
    addressResolver = getContractFactory('Lib_AddressManager')
      .connect(l1Signer)
      .attach(addressResolverAddress)

    const ctcAddress = await addressResolver.getAddress(
      'OVM_CanonicalTransactionChain'
    )
    canonicalTransactionChain = getContractFactory(
      'OVM_CanonicalTransactionChain'
    ).attach(ctcAddress)
  })

  // The transactions are enqueue'd with a `to` address of i.repeat(40)
  // meaning that the `to` value is different each iteration in a deterministic
  // way. They need to be inserted into the L2 chain in an ascending order.
  // Keep track of the receipts so that the blockNumber can be compared
  // against the `L1BlockNumber` on the tx objects.
  before(async () => {
    // Keep track of the L2 tip before submitting any transactions so that
    // the subsequent transactions can be queried for in the next test
    startBlock = (await l2Provider.getBlockNumber()) + 1
    endBlock = startBlock + numTxs - 1

    // Enqueue some transactions by building the calldata and then sending
    // the transaction to Layer 1
    for (let i = 0; i < numTxs; i++) {
      const input = ['0x' + `${i}`.repeat(40), 500_000, `0x0${i}`]
      const calldata = canonicalTransactionChain.interface.encodeFunctionData(
        'enqueue',
        input
      )

      const txResponse = await l1Signer.sendTransaction({
        data: calldata,
        to: canonicalTransactionChain.address,
      })

      const receipt = await txResponse.wait()
      receipts.push(receipt)
    }
  })

  // The batch submitter will notice that there are transactions
  // that are in the queue and submit them. L2 will pick up the
  // sequencer batch appended event and play the transactions.
  it('should order transactions correctly', async () => {
    // Wait until each tx from the previous test has
    // been executed
    let i: number
    for (i = 0; i < RETRIES; i++) {
      const tip = await l2Provider.getBlockNumber()
      if (tip >= endBlock) {
        break
      }
      await sleep(1000)
    }

    if (i === RETRIES) {
      throw new Error(
        'timed out waiting for queued transactions to be inserted'
      )
    }

    const from = await l1Signer.getAddress()
    // Keep track of an index into the receipts list and
    // increment it for each block fetched.
    let receiptIndex = 0
    // Fetch blocks
    for (i = 0; i < numTxs; i++) {
      const block = await l2Provider.getBlock(startBlock + i)
      const hash = block.transactions[0]
      // Use as any hack because additional properties are
      // added to the transaction response
      const tx = await (l2Provider.getTransaction(hash) as any)

      // The `to` addresses are defined in the previous test and
      // increment sequentially.
      expect(tx.to).to.be.equal('0x' + `${i}`.repeat(40))
      // The transaction type is EIP155
      expect(tx.txType).to.be.equal('EIP155')
      // The queue origin is Layer 1
      expect(tx.queueOrigin).to.be.equal('l1')
      // the L1TxOrigin is equal to the Layer one from
      expect(tx.l1TxOrigin).to.be.equal(from.toLowerCase())
      expect(typeof tx.l1BlockNumber).to.be.equal('number')
      // Get the receipt and increment the recept index
      const receipt = receipts[receiptIndex++]
      expect(tx.l1BlockNumber).to.be.equal(receipt.blockNumber)
    }
  })
})
