import '../setup'
/* External Imports */
import { getLogger, hexStrToNumber } from '@eth-optimism/core-utils'
import { ethers, ContractFactory, Wallet, Contract } from 'ethers'
import { resolve } from 'path'
import * as rimraf from 'rimraf'
import * as fs from 'fs'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
} from '../../src/app'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'
import { FullnodeHandler } from '../../src/types'

const log = getLogger('web3-handler', true)

const host = '0.0.0.0'
const port = 9999

const tmpFilePath = resolve(__dirname, `./.test_db`)

const getWallet = (httpProvider) => {
  const privateKey = '0x' + '60'.repeat(32)
  const wallet = new ethers.Wallet(privateKey, httpProvider)
  log.debug('Wallet address:', wallet.address)
  return wallet
}

const deploySimpleStorage = async (wallet: Wallet): Promise<Contract> => {
  const factory = new ContractFactory(
    SimpleStorage.abi,
    SimpleStorage.bytecode,
    wallet
  )

  // Deploy tx normally
  const simpleStorage = await factory.deploy()
  // Get the deployment tx receipt
  const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
    simpleStorage.deployTransaction.hash
  )
  // Verify that the contract which was deployed is correct
  deploymentTxReceipt.contractAddress.should.equal(simpleStorage.address)

  return simpleStorage
}

const setAndGetStorage = async (
  simpleStorage: Contract,
  httpProvider,
  executionManagerAddress
): Promise<void> => {
  // Create some constants we will use for storage
  const storageKey = '0x' + '01'.repeat(32)
  const storageValue = '0x' + '02'.repeat(32)
  // Set storage with our new storage elements
  const networkInfo = await httpProvider.getNetwork()
  const tx = await simpleStorage.setStorage(
    executionManagerAddress,
    storageKey,
    storageValue
  )
  // Get the storage
  const receipt = await httpProvider.getTransactionReceipt(tx.hash)
  const res = await simpleStorage.getStorage(
    executionManagerAddress,
    storageKey
  )
  // Verify we got the value!
  res.should.equal(storageValue)
}

/*********
 * TESTS *
 *********/

describe('Web3Handler', () => {
  let fullnodeHandler: FullnodeHandler
  let fullnodeRpcServer: FullnodeRpcServer
  let baseUrl: string

  beforeEach(async () => {
    fullnodeHandler = await TestWeb3Handler.create()
    fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

    fullnodeRpcServer.listen()

    baseUrl = `http://${host}:${port}`
  })

  afterEach(() => {
    if (!!fullnodeRpcServer) {
      fullnodeRpcServer.close()
    }
  })

  describe('the getBlockByNumber endpoint', () => {
    it('should return a block with the correct timestamp', async () => {
      const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      const timestamp = await httpProvider.send('evm_getTime', [])
      const block = await httpProvider.getBlock('latest')

      block.timestamp.should.eq(hexStrToNumber(timestamp))
    })

    it('should strip the execution manager deployment transaction from the transactions object', async () => {
      const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      const block = await httpProvider.getBlock(1, true)

      block['transactions'].should.be.empty
    })

    it.only('should increase the timestamp when blocks are created', async () => {
      const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      const executionManagerAddress = await httpProvider.send(
        'ovm_getExecutionManagerAddress',
        []
      )
      const { timestamp } = await httpProvider.getBlock("latest")
      const wallet = getWallet(httpProvider)
      const simpleStorage = await deploySimpleStorage(wallet)
      await setAndGetStorage(
        simpleStorage,
        httpProvider,
        executionManagerAddress
      )

      const block = await httpProvider.getBlock('latest', true)
      block.timestamp.should.be.gt(timestamp)
    })
  })

  describe('ephemeral node', () => {
    describe('SimpleStorage integration test', () => {
      it('should set storage & retrieve the value', async () => {
        const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )

        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)

        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )
      })
    })

    describe('snapshot and revert', () => {
      it('should  fail (snapshot and revert should only be available in the TestHandler)', async () => {
        const httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_snapshot', []).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })

        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_snapshot', []).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })

        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_revert', ['0x01']).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })
      })
    })
  })

  describe('persisted node', () => {
    let emAddress: string
    let wallet: Wallet
    let simpleStorage: Contract
    let httpProvider

    before(() => {
      rimraf.sync(tmpFilePath)
      fs.mkdirSync(tmpFilePath)
      process.env.LOCAL_L2_NODE_PERSISTENT_DB_PATH = tmpFilePath
    })
    after(() => {
      rimraf.sync(tmpFilePath)
      delete process.env.LOCAL_L2_NODE_PERSISTENT_DB_PATH
    })

    it('1/2 deploys the contracts', async () => {
      httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      emAddress = await httpProvider.send('ovm_getExecutionManagerAddress', [])
      wallet = getWallet(httpProvider)

      simpleStorage = await deploySimpleStorage(wallet)
    })

    it('2/2 uses previously deployed contract', async () => {
      await setAndGetStorage(simpleStorage, httpProvider, emAddress)
    })
  })
})
