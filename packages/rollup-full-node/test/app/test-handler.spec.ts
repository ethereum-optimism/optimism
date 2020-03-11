import '../setup'
/* External Imports */
import { add0x, getLogger, remove0x } from '@eth-optimism/core-utils'
import { ethers, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  Web3RpcMethods,
  TestWeb3Handler,
  FullnodeRpcServer,
  DefaultWeb3Handler,
} from '../../src'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'

const log = getLogger('test-web3-handler', true)

const secondsSinceEopch = (): number => {
  return Math.round(Date.now() / 1000)
}
const host = '0.0.0.0'
const port = 9998
const baseUrl = `http://${host}:${port}`

describe('TestHandler', () => {
  let testHandler: TestWeb3Handler

  beforeEach(async () => {
    testHandler = await TestWeb3Handler.create()
  })

  describe('Timestamps', () => {
    it('should get timestamp', async () => {
      const currentTime = secondsSinceEopch()
      const res: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const timeAfter = secondsSinceEopch()

      const timestamp: number = parseInt(remove0x(res), 16)
      timestamp.should.be.gte(currentTime, 'Timestamp out of range')
      timestamp.should.be.lte(timeAfter, 'Timestamp out of range')
    })

    it('should increase timestamp', async () => {
      const previous: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const previousTimestamp: number = parseInt(remove0x(previous), 16)

      const increase: number = 9999
      const setRes: string = await testHandler.handleRequest(
        Web3RpcMethods.increaseTimestamp,
        [increase.toString()]
      )
      setRes.should.equal(
        TestWeb3Handler.successString,
        'Should increase timestamp!'
      )

      const fetched: string = await testHandler.handleRequest(
        Web3RpcMethods.getTimestamp,
        []
      )
      const fetchedTimestamp: number = parseInt(remove0x(fetched), 16)
      fetchedTimestamp.should.be.gte(
        previousTimestamp + increase,
        'Timestamp was not increased properly!'
      )
    })
  })

  describe('Snapshot and revert', () => {
    it('should revert state', async () => {
      const testRpcServer = new FullnodeRpcServer(testHandler, host, port)

      testRpcServer.listen()
      const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      const executionManagerAddress = await httpProvider.send(
        'ovm_getExecutionManagerAddress',
        []
      )
      const privateKey = '0x' + '60'.repeat(32)
      const wallet = new ethers.Wallet(privateKey, httpProvider)
      log.debug('Wallet address:', wallet.address)
      const factory = new ContractFactory(
        SimpleStorage.abi,
        SimpleStorage.bytecode,
        wallet
      )

      const simpleStorage = await factory.deploy()
      const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
        simpleStorage.deployTransaction.hash
      )

      const storageKey = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '02'.repeat(32)
      const storageValue2 = '0x' + '03'.repeat(32)
      // Set storage with our new storage elements
      const networkInfo = await httpProvider.getNetwork()
      const tx = await simpleStorage.setStorage(
        executionManagerAddress,
        storageKey,
        storageValue
      )
      const snapShotId = await httpProvider.send('evm_snapshot', [])
      const tx2 = await simpleStorage.setStorage(
        executionManagerAddress,
        storageKey,
        storageValue2
      )
      const receipt = await httpProvider.getTransactionReceipt(tx.hash)
      const receipt2 = await httpProvider.getTransactionReceipt(tx2.hash)
      const response2 = await httpProvider.send('evm_revert', [snapShotId])
      const res = await simpleStorage.getStorage(
        executionManagerAddress,
        storageKey
      )
      res.should.equal(storageValue)
    })
  })
})
