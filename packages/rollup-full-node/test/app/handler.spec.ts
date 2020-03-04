import '../setup'
/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  FullnodeRpcServer,
  deployOvmContract,
  DefaultWeb3Handler,
} from '../../src/app'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'
import { ethers, ContractFactory } from 'ethers'
import { FullnodeHandler } from '../../src/types'

const log = getLogger('web3-handler', true)

const host = '0.0.0.0'
const port = 9999

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
})
