import '../setup'
/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { ethers, ContractFactory, Wallet, Contract } from 'ethers'
import { resolve } from 'path'
import * as rimraf from 'rimraf'
import * as fs from 'fs'

/* Internal Imports */
import { FullnodeRpcServer, DefaultWeb3Handler } from '../../src/app'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'
import { FullnodeHandler } from '../../src/types'

const log = getLogger('web3-handler', true)

const host = '0.0.0.0'
const port = 9999

const tmpFilePath = resolve(__dirname, `./.test_db`)

/*********
 * TESTS *
 *********/

describe('Web3Handler', () => {
  let fullnodeHandler: FullnodeHandler
  let fullnodeRpcServer: FullnodeRpcServer
  let baseUrl: string

  beforeEach(async () => {
    fullnodeHandler = await DefaultWeb3Handler.create()
    fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

    fullnodeRpcServer.listen()

    baseUrl = `http://${host}:${port}`
  })

  afterEach(() => {
    if (!!fullnodeRpcServer) {
      fullnodeRpcServer.close()
    }
  })

  describe('ephemeral node', () => {
    describe('SimpleStorage integration test', () => {
      it('should set storage & retrieve the value', async () => {
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

        // Deploy tx normally
        const simpleStorage = await factory.deploy()
        // Get the deployment tx receipt
        const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
          simpleStorage.deployTransaction.hash
        )
        // Verify that the contract which was deployed is correct
        deploymentTxReceipt.contractAddress.should.equal(simpleStorage.address)

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
      process.env.PERSISTED_L2_GANACHE_DB_FILE_PATH = tmpFilePath
    })
    after(() => {
      rimraf.sync(tmpFilePath)
    })

    it('1/2 deploys the contracts', async () => {
      httpProvider = new ethers.providers.JsonRpcProvider(baseUrl)
      emAddress = await httpProvider.send('ovm_getExecutionManagerAddress', [])
      const privateKey = '0x' + '60'.repeat(32)
      wallet = new ethers.Wallet(privateKey, httpProvider)
      log.debug('Wallet address:', wallet.address)
      const factory = new ContractFactory(
        SimpleStorage.abi,
        SimpleStorage.bytecode,
        wallet
      )

      // Deploy tx normally
      simpleStorage = await factory.deploy()
      // Get the deployment tx receipt
      const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
        simpleStorage.deployTransaction.hash
      )
      // Verify that the contract which was deployed is correct
      deploymentTxReceipt.contractAddress.should.equal(simpleStorage.address)
    })

    it('2/2 uses previously deployed contract', async () => {
      // Create some constants we will use for storage
      const storageKey = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '02'.repeat(32)
      // Set storage with our new storage elements
      const tx = await simpleStorage.setStorage(
        emAddress,
        storageKey,
        storageValue
      )
      // Get the storage
      const res = await simpleStorage.getStorage(emAddress, storageKey)
      // Verify we got the value!
      res.should.equal(storageValue)
    })
  })
})
