import { getContractInterface, getContractFactory } from '@eth-optimism/contracts'
import { Contract, utils, Wallet } from 'ethers'
import { Watcher } from './watcher'

import {
  getAddressManager,
  
  l1Provider,
  l2Provider,
  
  bobl1Wallet,
  bobl2Wallet,
  
  alicel1Wallet,
  alicel2Wallet,

  katel1Wallet,
  katel2Wallet,

  getL2ETHGateway,
  getL1ETHGateway,
  l1MessengerFastAddress,
} from './utils'

import {
  initWatcher,
  initWatcherMessengerFast,
  CrossDomainMessagePair,
  Direction,
  Relayer,
  waitForXDomainTransaction,
} from './watcher-utils'

import * as fs from 'fs'

import { TransactionResponse } from '@ethersproject/providers'

/// Helper class for instantiating a test environment with a funded account
export class OptimismEnv {
  // L1 Contracts
  addressManager: Contract
  L1ETHGateway: Contract
  l1Messenger: Contract
  l1MessengerAddress: String
  ctc: Contract

  l1Provider
  l2Provider

  // L2 Contracts
  L2ETHGateway: Contract
  l2Messenger: Contract

  // The L1 <> L2 State watcher
  watcher: Watcher

  watcherMessengerFast: Watcher

  // The wallets
  bobl1Wallet: Wallet
  bobl2Wallet: Wallet
  
  alicel1Wallet: Wallet
  alicel2Wallet: Wallet

  katel1Wallet: Wallet
  katel2Wallet: Wallet

  constructor(args: any) {
    this.addressManager = args.addressManager
    this.L1ETHGateway = args.L1ETHGateway
    this.l1Messenger = args.l1Messenger
    this.l1MessengerAddress = args.l1MessengerAddress
    this.L2ETHGateway = args.L2ETHGateway
    this.l2Messenger = args.l2Messenger
    this.watcher = args.watcher
    this.watcherMessengerFast = args.watcherMessengerFast
    this.bobl1Wallet = args.bobl1Wallet
    this.bobl2Wallet = args.bobl2Wallet
    this.alicel1Wallet = args.alicel1Wallet
    this.alicel2Wallet = args.alicel2Wallet
    this.katel1Wallet = args.katel1Wallet
    this.katel2Wallet = args.katel2Wallet
    this.l1Provider = args.l1Provider
    this.l2Provider = args.l2Provider
    this.ctc = args.ctc
  }

  static async new(): Promise<OptimismEnv> {

    const addressManager = getAddressManager(bobl1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const watcherMessengerFast = await initWatcherMessengerFast(l1Provider, l2Provider, addressManager, l1MessengerFastAddress)

    const L1ETHGateway = await getL1ETHGateway(bobl1Wallet, addressManager)
    const L2ETHGateway = getL2ETHGateway(bobl2Wallet)

    const l1Messenger = getContractFactory('iOVM_L1CrossDomainMessenger')
      .connect(bobl1Wallet)
      .attach(watcher.l1.messengerAddress)

    const l1MessengerAddress = l1Messenger.address;  
    
    const l2Messenger = getContractFactory('iOVM_L2CrossDomainMessenger')
      .connect(bobl2Wallet)
      .attach(watcher.l2.messengerAddress)

    const ctcAddress = await addressManager.getAddress(
      'OVM_CanonicalTransactionChain'
    )
    const ctc = getContractFactory('OVM_CanonicalTransactionChain')
      .connect(bobl1Wallet)
      .attach(ctcAddress)

    return new OptimismEnv({
      addressManager,
      L1ETHGateway,
      ctc,
      l1Messenger,
      l1MessengerAddress,
      L2ETHGateway,
      l2Messenger,
      
      watcher,
      watcherMessengerFast,

      bobl1Wallet,
      bobl2Wallet,

      alicel1Wallet,
      alicel2Wallet,
      
      katel1Wallet,
      katel2Wallet,

      l1Provider,
      l2Provider
    })
  }

  async waitForXDomainTransaction(
    tx: Promise<TransactionResponse> | TransactionResponse,
    direction: Direction,
  ): Promise<CrossDomainMessagePair> {
    return waitForXDomainTransaction(this.watcher, tx, direction)
  }

  async waitForXDomainTransactionMessengerFast(
    tx: Promise<TransactionResponse> | TransactionResponse,
    direction: Direction
  ): Promise<CrossDomainMessagePair> {
    return waitForXDomainTransaction(this.watcherMessengerFast, tx, direction)
  }

  async waitForRevertXDomainTransaction(
     tx: Promise<TransactionResponse> | TransactionResponse,
     direction: Direction
   ) {
     const {remoteReceipt} = await waitForXDomainTransaction(this.watcher, tx, direction)
     const [xDomainMsgHash] = await this.watcher.getMessageHashesFromL1Tx(remoteReceipt.transactionHash)
     await this.watcher.getL2TransactionReceipt(xDomainMsgHash)
   }
}
