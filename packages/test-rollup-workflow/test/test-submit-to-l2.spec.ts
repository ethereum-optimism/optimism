import './setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

import { Contract, ContractFactory, Wallet } from 'ethers'
import { JsonRpcProvider, Provider, TransactionReceipt } from 'ethers/providers'

import * as SimpleStorageContract from '../build/SimpleStorage.json'

const log = getLogger('rollup-workflow-test', true)

describe('Test Sending Transactions Directly To L2', () => {
  let wallet: Wallet
  let provider: Provider
  let simpleStorage: Contract

  before(async () => {
    const nodeURL: string = 'http://0.0.0.0:8545'
    provider = new JsonRpcProvider(nodeURL)
    wallet = Wallet.createRandom().connect(provider)

    log.debug(`connected to provider at ${nodeURL}`)

    const factory = new ContractFactory(
      SimpleStorageContract.abi,
      SimpleStorageContract.bytecode,
      wallet
    )

    const deployTx = factory.getDeployTransaction()
    deployTx.gasPrice = 0
    const res = await wallet.sendTransaction(deployTx)
    log.debug(`Deploy tx sent. Hash: ${res.hash}`)
    const receipt: TransactionReceipt = await provider.waitForTransaction(
      res.hash
    )
    receipt.status.should.equal(1, `Deploy transaction failed`)

    log.debug(`Contract deployed. Address: ${receipt.contractAddress}`)

    simpleStorage = new Contract(
      receipt.contractAddress, // '0x97673537F19b51289E1279288734D981e7527CA4',
      SimpleStorageContract.abi,
      wallet
    )
  })

  it('Sets storage N times', async () => {
    const key: string = 'test'
    for (let i = 0; i < 501; i++) {
      log.debug(`Sending tx to set storage key ${key}`)
      const res = await simpleStorage.setStorage(key, `${key}${i}`)
      const receipt: TransactionReceipt = await provider.waitForTransaction(
        res.hash
      )
      receipt.status.should.equal(
        1,
        `Transaction ${i} failed! ${JSON.stringify(receipt)}`
      )

      const setStorage = await simpleStorage.getStorage(key)
      setStorage.should.equal(`${key}${i}`, `Storage not set to ${key}${i}`)
    }
  }).timeout(100_000)
})
