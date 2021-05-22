import { getContractFactory } from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/core-utils'
import { Contract, utils, Wallet } from 'ethers'
import {
  getAddressManager,
  l1Provider,
  l2Provider,
  l1Wallet,
  l2Wallet,
  fundUser,
  getOvmEth,
  getGateway,
} from './utils'
import {
  initWatcher,
  CrossDomainMessagePair,
  Direction,
  waitForXDomainTransaction,
} from './watcher-utils'
import { TransactionResponse } from '@ethersproject/providers'

/// Helper class for instantiating a test environment with a funded account
export class OptimismEnv {
  // L1 Contracts
  addressManager: Contract
  gateway: Contract
  l1Messenger: Contract
  ctc: Contract

  // L2 Contracts
  ovmEth: Contract
  l2Messenger: Contract

  // The L1 <> L2 State watcher
  watcher: Watcher

  // The wallets
  l1Wallet: Wallet
  l2Wallet: Wallet

  constructor(args: any) {
    this.addressManager = args.addressManager
    this.gateway = args.gateway
    this.l1Messenger = args.l1Messenger
    this.ovmEth = args.ovmEth
    this.l2Messenger = args.l2Messenger
    this.watcher = args.watcher
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.ctc = args.ctc
  }

  static async new(): Promise<OptimismEnv> {
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const gateway = await getGateway(l1Wallet, addressManager)

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    if (balance.isZero()) {
      await fundUser(watcher, gateway, utils.parseEther('10'))
    }

    const ovmEth = getOvmEth(l2Wallet)
    const l1Messenger = getContractFactory('iOVM_L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(watcher.l1.messengerAddress)
    const l2Messenger = getContractFactory('iOVM_L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(watcher.l2.messengerAddress)

    const ctcAddress = await addressManager.getAddress(
      'OVM_CanonicalTransactionChain'
    )
    const ctc = getContractFactory('OVM_CanonicalTransactionChain')
      .connect(l1Wallet)
      .attach(ctcAddress)

    return new OptimismEnv({
      addressManager,
      gateway,
      ctc,
      l1Messenger,
      ovmEth,
      l2Messenger,
      watcher,
      l1Wallet,
      l2Wallet,
    })
  }

  async waitForXDomainTransaction(
    tx: Promise<TransactionResponse> | TransactionResponse,
    direction: Direction
  ): Promise<CrossDomainMessagePair> {
    return waitForXDomainTransaction(this.watcher, tx, direction)
  }
}
