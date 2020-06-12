import '../setup'
/* External Imports */
import { Address, deployContract, getWallets } from '@eth-optimism/rollup-core'

/* Internal Imports */
import { createProviderForHandler, TestWeb3Handler } from '../../src/app'

/* Contract Imports */

import * as SimpleStorage from '../contracts/build/transpiled/SimpleStorage.json'
import * as SimpleCaller from '../contracts/build/transpiled/SimpleCaller.json'
import * as SelfAware from '../contracts/build/transpiled/SelfAware.json'
import * as CallerGetter from '../contracts/build/transpiled/CallerGetter.json'
import * as OriginGetter from '../contracts/build/transpiled/OriginGetter.json'
import * as CallerReturner from '../contracts/build/transpiled/CallerReturner.json'
import * as TimeGetter from '../contracts/build/transpiled/TimeGetter.json'

describe.only(`Various opcodes should be usable in combination with transpiler and full node`, () => {
  let handler: TestWeb3Handler
  let provider
  let wallet

  beforeEach(async () => {
    handler = await TestWeb3Handler.create()
    provider = createProviderForHandler(handler)
    const wallets = getWallets(provider)
    wallet = wallets[0]
  })

  // TEST BASIC FUNCTIONALITY

  it('should process cross-ovm-contract calls', async () => {
    const simpleStorage = await deployContract(wallet, SimpleStorage, [], [])
    const simpleCaller = await deployContract(wallet, SimpleCaller, [], [])

    const storageKey = '0x' + '01'.repeat(32)
    const storageValue = '0x' + '02'.repeat(32)

    await simpleStorage.setStorage(storageKey, storageValue)

    const res = await simpleCaller.doGetStorageCall(
      simpleStorage.address,
      storageKey
    )
    res.should.equal(storageValue)
  })
  it.only('should work for address(this)', async () => {
    const selfAware = await deployContract(wallet, SelfAware, [], [])
    const deployedAddress: Address = selfAware.address
    const returnedAddress: Address = await selfAware.getMyAddress()
    deployedAddress.should.equal(returnedAddress)
  })
  it.only('should work for block.timestamp', async () => {
    const timeGetter = await deployContract(wallet, TimeGetter, [], [])
    const contractTime = await timeGetter.getTimestamp()
    contractTime.toNumber().should.be.gt(0)
  })
  it('should work for msg.sender', async () => {
    const callerReturner = await deployContract(wallet, CallerReturner, [], [])
    const callerGetter = await deployContract(wallet, CallerGetter, [], [])
    const result = await callerGetter.getMsgSenderFrom(callerReturner.address)
    result.should.equal(callerGetter.address)
  })
  it.only('should work for tx.origin', async () => {
    const originGetter = await deployContract(wallet, OriginGetter, [], [])
    const result = await originGetter.getTxOrigin()
    result.should.equal(wallet.address)
  })

  // SIMPLE STORAGE TEST
  it.only('should set storage & retrieve the value', async () => {
    const simpleStorage = await deployContract(wallet, SimpleStorage, [], [])
    // Create some constants we will use for storage
    const storageKey = '0x' + '01'.repeat(32)
    const storageValue = '0x' + '02'.repeat(32)
    // Set storage with our new storage elements
    await simpleStorage.setStorage(storageKey, storageValue)
    // Get the storage
    const res = await simpleStorage.getStorage(storageKey)
    // Verify we got the value!
    res.should.equal(storageValue)
  })
})
