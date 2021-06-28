import { getContractFactory, predeploys } from '@eth-optimism/contracts'
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
  getL1Bridge,
  getL2Bridge,
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
  l1Bridge: Contract
  l1Messenger: Contract
  ctc: Contract

  // L2 Contracts
  ovmEth: Contract
  l2Bridge: Contract
  l2Messenger: Contract
  gasPriceOracle: Contract

  // The L1 <> L2 State watcher
  watcher: Watcher

  // The wallets
  l1Wallet: Wallet
  l2Wallet: Wallet

  constructor(args: any) {
    this.addressManager = args.addressManager
    this.l1Bridge = args.l1Bridge
    this.l1Messenger = args.l1Messenger
    this.ovmEth = args.ovmEth
    this.l2Bridge = args.l2Bridge
    this.l2Messenger = args.l2Messenger
    this.gasPriceOracle = args.gasPriceOracle
    this.watcher = args.watcher
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.ctc = args.ctc
  }

  static async new(): Promise<OptimismEnv> {
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    if (balance.isZero()) {
      await fundUser(watcher, l1Bridge, utils.parseEther('20'))
    }
    const l1Messenger = getContractFactory('iOVM_L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(watcher.l1.messengerAddress)
    const ovmEth = getOvmEth(l2Wallet)
    const l2Bridge = await getL2Bridge(l2Wallet)
    const l2Messenger = getContractFactory('iOVM_L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(watcher.l2.messengerAddress)

    const ctcAddress = await addressManager.getAddress(
      'OVM_CanonicalTransactionChain'
    )
    const ctc = getContractFactory('OVM_CanonicalTransactionChain')
      .connect(l1Wallet)
      .attach(ctcAddress)

    const gasPriceOracle = getContractFactory('OVM_GasPriceOracle')
      .connect(l2Wallet)
      .attach(predeploys.OVM_GasPriceOracle)

    return new OptimismEnv({
      addressManager,
      l1Bridge,
      ctc,
      l1Messenger,
      ovmEth,
      gasPriceOracle,
      l2Bridge,
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
