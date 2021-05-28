import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory, utils } from 'ethers'
import { Direction } from './shared/watcher-utils'

/* Imports: Internal */
import l1SimpleStorageJson from '../artifacts/contracts/SimpleStorage.sol/SimpleStorage.json'
import l2SimpleStorageJson from '../artifacts-ovm/contracts/SimpleStorage.sol/SimpleStorage.json'
import { OptimismEnv } from './shared/env'

describe('Basic L1<>L2 Communication', async () => {
  let Factory__L1SimpleStorage: ContractFactory
  let Factory__L2SimpleStorage: ContractFactory
  let L1SimpleStorage: Contract
  let L2SimpleStorage: Contract
  let env: OptimismEnv

  before(async () => {
    env = await OptimismEnv.new()
    Factory__L1SimpleStorage = new ContractFactory(
      l1SimpleStorageJson.abi,
      l1SimpleStorageJson.bytecode,
      env.l1Wallet
    )
    Factory__L2SimpleStorage = new ContractFactory(
      l2SimpleStorageJson.abi,
      l2SimpleStorageJson.bytecode,
      env.l2Wallet
    )
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
    const transaction = await env.l2Messenger.sendMessage(
      L1SimpleStorage.address,
      L1SimpleStorage.interface.encodeFunctionData('setValue', [value]),
      5000000
    )
    await env.waitForXDomainTransaction(transaction, Direction.L2ToL1)
    expect(await L1SimpleStorage.msgSender()).to.equal(env.l1Messenger.address)
    expect(await L1SimpleStorage.xDomainSender()).to.equal(env.l2Wallet.address)
    expect(await L1SimpleStorage.value()).to.equal(value)
    expect((await L1SimpleStorage.totalCount()).toNumber()).to.equal(1)
  })

  it('should deposit from L1 -> L2', async () => {
    const value = `0x${'42'.repeat(32)}`

    // Send L1 -> L2 message.
    const transaction = await env.l1Messenger.sendMessageViaChainId(
      420,
      L2SimpleStorage.address,
      L2SimpleStorage.interface.encodeFunctionData('setValue', [value]),
      5000000
    )

    await env.waitForXDomainTransaction(transaction, Direction.L1ToL2)

    expect(await L2SimpleStorage.msgSender()).to.equal(env.l2Messenger.address)
    expect(await L2SimpleStorage.xDomainSender()).to.equal(env.l1Wallet.address)
    expect(await L2SimpleStorage.value()).to.equal(value)
    expect((await L2SimpleStorage.totalCount()).toNumber()).to.equal(1)
  })
})
