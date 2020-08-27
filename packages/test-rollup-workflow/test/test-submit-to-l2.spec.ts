import './setup'

/* External Imports */
import { getLogger, keccak256FromUtf8, sleep } from '@eth-optimism/core-utils'
import {
  CHAIN_ID,
  GAS_LIMIT,
  DefaultL2NodeService,
  GethSubmission,
  L2NodeService,
} from '@eth-optimism/rollup-core'

import { Contract, ContractFactory, Wallet } from 'ethers'
import { UnsignedTransaction } from 'ethers/utils'
import { JsonRpcProvider, Provider, TransactionReceipt } from 'ethers/providers'

import * as SimpleStorageContract from '../build/SimpleStorage.json'

const log = getLogger('rollup-workflow-test')

const getDeployTx = (wallet: Wallet): UnsignedTransaction => {
  const factory = new ContractFactory(
    SimpleStorageContract.abi,
    SimpleStorageContract.bytecode,
    wallet
  )

  const deployTx = factory.getDeployTransaction()
  deployTx.gasPrice = 0
  return deployTx
}

describe('Test Sending Transactions Directly To L2', () => {
  let wallet: Wallet
  let provider: Provider
  let simpleStorage: Contract
  const gethNodeUrl: string = 'http://0.0.0.0:8545'

  describe('Sending transactions to L2 Geth', () => {
    before(async () => {
      provider = new JsonRpcProvider(gethNodeUrl)
      wallet = Wallet.createRandom().connect(provider)

      log.debug(`connected to provider at ${gethNodeUrl}`)
      const deployTx = getDeployTx(wallet)

      const res = await wallet.sendTransaction(deployTx)
      log.debug(`Deploy tx sent. Hash: ${res.hash}`)
      const receipt: TransactionReceipt = await provider.waitForTransaction(
        res.hash
      )
      receipt.status.should.equal(1, `Deploy transaction failed`)

      log.debug(`Contract deployed. Address: ${receipt.contractAddress}`)

      simpleStorage = new Contract(
        receipt.contractAddress, //'0x6151c37e5F46349B405fEF2839D1B8183ff517C0',
        SimpleStorageContract.abi,
        wallet
      )
    })

    const numTxsToSend: number = 50
    it(`Sets storage ${numTxsToSend} times`, async () => {
      const key: string = 'test'
      for (let i = 0; i < numTxsToSend; i++) {
        log.debug(`Sending tx to set storage key ${key}`)
        const res = await simpleStorage.setStorage(key, `${key}${i}`)
        await sleep(1)
      }
    }).timeout(100_000)
  })

  describe.skip('Sending Rollup Transactions', () => {
    let l2NodeService: L2NodeService

    beforeEach(async () => {
      provider = new JsonRpcProvider(gethNodeUrl, CHAIN_ID)
      // Address for wallet: 0x6a399F0A626A505e2F6C2b5Da181d98D722dC86D
      wallet = new Wallet(
        'efb6aa1f37082ac40884a340684672ccbb5a4e6000860953afcf73c90c33e4f9',
        provider
      )
      l2NodeService = new DefaultL2NodeService(wallet)
    })

    it('Sends RollupTransactions to geth eth_sendRollupTransactions endpoint', async () => {
      const deployTx = getDeployTx(wallet)

      const gethSubmission: GethSubmission = {
        submissionNumber: 1,
        timestamp: 1,
        blockNumber: 1,
        rollupTransactions: [
          {
            l1RollupTxId: 1,
            indexWithinSubmission: 1,
            gasLimit: GAS_LIMIT,
            nonce: await wallet.getTransactionCount(),
            sender: Wallet.createRandom().address,
            target: deployTx.to,
            calldata: deployTx.data.toString(),
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
})
