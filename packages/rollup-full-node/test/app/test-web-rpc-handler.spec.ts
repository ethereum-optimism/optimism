import '../setup'
/* External Imports */
import {
  add0x,
  getLogger,
  remove0x,
  castToNumber,
  hexStrToBuf,
  TestUtils,
  hexStrToNumber,
} from '@eth-optimism/core-utils'
import { ethers, ContractFactory } from 'ethers'
import { getWallets, deployContract } from 'ethereum-waffle'

/* Internal Imports */
import {
  Web3RpcMethods,
  TestWeb3Handler,
  FullnodeRpcServer,
  DefaultWeb3Handler,
} from '../../src'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'
import * as EmptyContract from '../contracts/build/untranspiled/EmptyContract.json'
import * as CallerStorer from '../contracts/build/transpiled/CallerStorer.json'
import { getOvmTransactionMetadata } from '@eth-optimism/ovm'

const log = getLogger('test-web3-handler', true)

const secondsSinceEpoch = (): number => {
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
      const currentTime = secondsSinceEpoch()
      const latestBlock = await testHandler.handleRequest(
        Web3RpcMethods.getBlockByNumber,
        ['latest', false]
      )
      const timeAfter = secondsSinceEpoch()

      const timestamp: number = hexStrToNumber(latestBlock['timestamp'])
      timestamp.should.be.lte(currentTime, 'Timestamp out of range')
      timestamp.should.be.lte(timeAfter, 'Timestamp out of range')
    })

    it('should increase timestamp', async () => {
      let latestBlock = await testHandler.handleRequest(
        Web3RpcMethods.getBlockByNumber,
        ['latest', false]
      )
      const previousTimestamp: number = hexStrToNumber(latestBlock['timestamp'])

      const increase: number = 9999
      const setRes: string = await testHandler.handleRequest(
        Web3RpcMethods.increaseTimestamp,
        [increase]
      )
      setRes.should.equal(
        TestWeb3Handler.successString,
        'Should increase timestamp!'
      )

      latestBlock = await testHandler.handleRequest(
        Web3RpcMethods.getBlockByNumber,
        ['latest', false]
      )
      const fetchedTimestamp: number = hexStrToNumber(latestBlock['timestamp'])
      fetchedTimestamp.should.be.lte(
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
      testRpcServer.close()
    })

    it('should revert changes to the timestamp', async () => {
      const testRpcServer = new FullnodeRpcServer(testHandler, host, port)
      testRpcServer.listen()
      const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      let latestBlock = await httpProvider.getBlock('latest', false)
      const startTimestamp = await latestBlock['timestamp']
      // Increase timestamp by 1 second
      await httpProvider.send('evm_increaseTime', [1])
      // Take a snapshot at timestamp + 1
      const snapShotId = await httpProvider.send('evm_snapshot', [])
      // Increase timestamp by 1 second again
      await httpProvider.send('evm_increaseTime', [1])
      const response2 = await httpProvider.send('evm_revert', [snapShotId])
      latestBlock = await httpProvider.getBlock('latest', false)
      const timestamp = await latestBlock['timestamp']

      castToNumber(timestamp).should.eq(startTimestamp)
      testRpcServer.close()
    })
  })

  describe('the getCode endpoint', () => {
    let testRpcServer
    let httpProvider
    let wallet

    beforeEach(async () => {
      testRpcServer = new FullnodeRpcServer(testHandler, host, port)
      testRpcServer.listen()
      httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      wallet = getWallets(httpProvider)[0]
    })

    afterEach(async () => {
      await testRpcServer.close()
    })

    it('should be successful if the default block parameter is "latest"', async () => {
      const factory = new ethers.ContractFactory(
        EmptyContract.abi,
        EmptyContract.bytecode,
        wallet
      )
      const emptyContract = await deployContract(wallet, EmptyContract, [])
      const code = await httpProvider.getCode(emptyContract.address, 'latest')
      hexStrToBuf(code).byteLength.should.be.greaterThan(0)
    })

    it('should be successful if the default block parameter is set to the latest block number', async () => {
      const factory = new ethers.ContractFactory(
        EmptyContract.abi,
        EmptyContract.bytecode,
        wallet
      )
      const emptyContract = await deployContract(wallet, EmptyContract, [])
      const curentBlockNumber = await httpProvider.getBlockNumber()
      const code = await httpProvider.getCode(
        emptyContract.address,
        curentBlockNumber
      )
      hexStrToBuf(code).byteLength.should.be.greaterThan(0)
    })

    it('should be fail if the default block parameter is set to a block number before the current one', async () => {
      const factory = new ethers.ContractFactory(
        EmptyContract.abi,
        EmptyContract.bytecode,
        wallet
      )
      const emptyContract = await deployContract(wallet, EmptyContract, [])
      const curentBlockNumber = await httpProvider.getBlockNumber()
      TestUtils.assertThrowsAsync(async () =>
        httpProvider.getCode(emptyContract.address, curentBlockNumber - 1)
      )
    })
  })

  describe('the sendTransaction endpoint', () => {
    let testRpcServer
    let httpProvider
    let wallet

    beforeEach(async () => {
      testRpcServer = new FullnodeRpcServer(testHandler, host, port)
      testRpcServer.listen()
      httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      wallet = getWallets(httpProvider)[0]
    })

    afterEach(async () => {
      await testRpcServer.close()
    })

    it('should run the transaction for arbitrary from, to, and data, correctly filling optional fields including nonce', async () => {
      const storageKey = add0x('01'.repeat(32))
      const storageValue = add0x('02'.repeat(32))
      const executionManagerAddress = await httpProvider.send(
        'ovm_getExecutionManagerAddress',
        []
      )
      const factory = new ContractFactory(
        SimpleStorage.abi,
        SimpleStorage.bytecode,
        wallet
      )
      const simpleStorage = await factory.deploy()
      const transactionData = await simpleStorage.interface.functions[
        'setStorage'
      ].encode([executionManagerAddress, storageKey, storageValue])
      const transaction = {
        from: wallet.address,
        to: simpleStorage.address,
        data: transactionData,
      }

      await httpProvider.send('eth_sendTransaction', [transaction])
      const res = await simpleStorage.getStorage(
        executionManagerAddress,
        storageKey
      )
      res.should.equal(storageValue)
    })

    it('the EVM should actually see the arbitrary .from as the sender of the tx', async () => {
      const factory = new ContractFactory(
        CallerStorer.abi,
        CallerStorer.bytecode,
        wallet
      )
      const callerStorer = await factory.deploy()
      const setData = await callerStorer.interface.functions[
        'storeMsgSender'
      ].encode([])
      const randomFromAddress = add0x('02'.repeat(20))
      const transaction = {
        from: randomFromAddress,
        to: callerStorer.address,
        data: setData,
      }
      await httpProvider.send('eth_sendTransaction', [transaction])
      const res = await callerStorer.getLastMsgSender()
      res.should.equal(randomFromAddress)
    })
  })
})
