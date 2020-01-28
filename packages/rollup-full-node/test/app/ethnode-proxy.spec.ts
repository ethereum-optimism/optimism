import '../setup'
/* External Imports */
import { getLogger } from '@pigi/core-utils'
import { DB, newInMemoryDB } from '@pigi/core-db/'

/* Internal Imports */
import {
  EthnodeProxy,
  createMockOvmProvider,
  deployOvmContract,
} from '../../src/app'
import * as SimpleStorage from '../contracts/build/SimpleStorage.json'
import { getWallets } from 'ethereum-waffle'

const log = getLogger('ethnode-proxy', true)

/*********
 * TESTS *
 *********/

describe('EthnodeProxy', () => {
  describe('SimpleStorage integration test', () => {
    it('should set storage & retrieve the value', async () => {
      let provider
      let executionManagerAddress
      ;[provider, executionManagerAddress] = await createMockOvmProvider()
      const wallet = getWallets(provider)[0]
      // Deploy the contract
      const simpleStorage = await deployOvmContract(wallet, SimpleStorage)
      // Create some constants we will use for storage
      const storageKey = '0x' + '01'.repeat(32)
      const storageValue = '0x' + '02'.repeat(32)
      // Set storage with our new storage elements
      const tx = await simpleStorage.setStorage(
        executionManagerAddress,
        storageKey,
        storageValue
      )
      // Get the storage
      const receipt = await provider.getTransactionReceipt(tx.hash)
      const res = await simpleStorage.getStorage(
        executionManagerAddress,
        storageKey
      )
      // Verify we got the value!
      res.should.equal(storageValue)
    })
  })
})
