import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory, Wallet, providers } from 'ethers'
import { Watcher } from '@eth-optimism/core-utils'
import {
  initWatcher,
  waitForXDomainTransaction,
  Direction,
} from './shared/watcher-utils'
import { getContractFactory } from '@eth-optimism/contracts'

/* Imports: Internal */
import l1SimpleStorageJson from '../artifacts/contracts/SimpleStorage.sol/SimpleStorage.json'
import l2SimpleStorageJson from '../artifacts-ovm/contracts/SimpleStorage.sol/SimpleStorage.json'

describe('Basic L1<>L2 Communication', async () => {
  let l1Wallet: Wallet
  let l2Wallet: Wallet
  let l1Provider: providers.JsonRpcProvider
  let l2Provider: providers.JsonRpcProvider
  let AddressManager: Contract

  let Factory__L1SimpleStorage: ContractFactory
  let Factory__L2SimpleStorage: ContractFactory
  let L1CrossDomainMessenger: Contract
  let L2CrossDomainMessenger: Contract

  let watcher: Watcher

  let L1SimpleStorage: Contract
  let L2SimpleStorage: Contract

  before(async () => {
    const httpPort = 8545
    const l1HttpPort = 9545
    l1Provider = new providers.JsonRpcProvider(`http://localhost:${l1HttpPort}`)
    l2Provider = new providers.JsonRpcProvider(`http://localhost:${httpPort}`)
    l1Wallet = new Wallet(
      '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      l1Provider
    )
    l2Wallet = Wallet.createRandom().connect(l2Provider)

    const addressManagerAddress = '0x5FbDB2315678afecb367f032d93F642f64180aa3'
    AddressManager = getContractFactory('Lib_AddressManager')
      .connect(l1Wallet)
      .attach(addressManagerAddress)

    Factory__L1SimpleStorage = new ContractFactory(
      l1SimpleStorageJson.abi,
      l1SimpleStorageJson.bytecode,
      l1Wallet
    )
    Factory__L2SimpleStorage = new ContractFactory(
      l2SimpleStorageJson.abi,
      l2SimpleStorageJson.bytecode,
      l2Wallet
    )

    watcher = await initWatcher(l1Provider, l2Provider, AddressManager)
    L1CrossDomainMessenger = getContractFactory('iOVM_L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(watcher.l1.messengerAddress)
    L2CrossDomainMessenger = getContractFactory('iOVM_L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(watcher.l2.messengerAddress)
  })

  beforeEach(async () => {
    L1SimpleStorage = await Factory__L1SimpleStorage.deploy()
    await L1SimpleStorage.deployTransaction.wait()
    L2SimpleStorage = await Factory__L2SimpleStorage.deploy()
    await L2SimpleStorage.deployTransaction.wait()
  })

  it('should withdraw from L2 -> L1', async () => {
    const value = `0x${'77'.repeat(32)}`

    // Send L2 -> L1 message.
    const transaction = await L2CrossDomainMessenger.sendMessage(
      L1SimpleStorage.address,
      L1SimpleStorage.interface.encodeFunctionData('setValue', [value]),
      5000000,
      { gasLimit: 7000000 }
    )

    await waitForXDomainTransaction(watcher, transaction, Direction.L2ToL1)

    expect(await L1SimpleStorage.msgSender()).to.equal(
      L1CrossDomainMessenger.address
    )
    expect(await L1SimpleStorage.xDomainSender()).to.equal(l2Wallet.address)
    expect(await L1SimpleStorage.value()).to.equal(value)
    expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(1)
  })

  it('should deposit from L1 -> L2', async () => {
    const value = `0x${'42'.repeat(32)}`

    // Send L1 -> L2 message.
    const transaction = await L1CrossDomainMessenger.sendMessage(
      L2SimpleStorage.address,
      L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
      5000000,
      { gasLimit: 7000000 }
    )

    await waitForXDomainTransaction(watcher, transaction, Direction.L1ToL2)

    expect(await L2SimpleStorage.msgSender()).to.equal(
      L2CrossDomainMessenger.address
    )
    expect(await L2SimpleStorage.xDomainSender()).to.equal(l1Wallet.address)
    expect(await L2SimpleStorage.value()).to.equal(value)
    expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
  })
})
