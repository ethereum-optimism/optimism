/* Imports: External */
import { Contract, utils, Wallet, providers } from 'ethers'
import { TransactionResponse } from '@ethersproject/providers'
import { getContractFactory, predeploys } from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/core-utils'
import { getMessagesAndProofsForL2Transaction } from '@eth-optimism/message-relayer'

/* Imports: Internal */
import {
  getAddressManager,
  l1Provider,
  l2Provider,
  replicaProvider,
  l1Wallet,
  l2Wallet,
  gasPriceOracleWallet,
  fundUser,
  getOvmEth,
  getL1Bridge,
  getL2Bridge,
  sleep,
  envConfig,
  DEFAULT_TEST_GAS_L1,
} from './utils'
import {
  initWatcher,
  CrossDomainMessagePair,
  Direction,
  waitForXDomainTransaction,
} from './watcher-utils'

/// Helper class for instantiating a test environment with a funded account
export class OptimismEnv {
  // L1 Contracts
  addressManager: Contract
  l1Bridge: Contract
  l1Messenger: Contract
  l1BlockNumber: Contract
  ctc: Contract
  scc: Contract

  // L2 Contracts
  ovmEth: Contract
  l2Bridge: Contract
  l2Messenger: Contract
  gasPriceOracle: Contract
  sequencerFeeVault: Contract

  // The L1 <> L2 State watcher
  watcher: Watcher

  // The wallets
  l1Wallet: Wallet
  l2Wallet: Wallet

  // The providers
  l1Provider: providers.JsonRpcProvider
  l2Provider: providers.JsonRpcProvider
  replicaProvider: providers.JsonRpcProvider

  constructor(args: any) {
    this.addressManager = args.addressManager
    this.l1Bridge = args.l1Bridge
    this.l1Messenger = args.l1Messenger
    this.l1BlockNumber = args.l1BlockNumber
    this.ovmEth = args.ovmEth
    this.l2Bridge = args.l2Bridge
    this.l2Messenger = args.l2Messenger
    this.gasPriceOracle = args.gasPriceOracle
    this.sequencerFeeVault = args.sequencerFeeVault
    this.watcher = args.watcher
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.l1Provider = args.l1Provider
    this.l2Provider = args.l2Provider
    this.replicaProvider = args.replicaProvider
    this.ctc = args.ctc
    this.scc = args.scc
  }

  static async new(): Promise<OptimismEnv> {
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    const min = envConfig.L2_WALLET_MIN_BALANCE_ETH.toString()
    const topUp = envConfig.L2_WALLET_TOP_UP_AMOUNT_ETH.toString()
    if (balance.lt(utils.parseEther(min))) {
      await fundUser(watcher, l1Bridge, utils.parseEther(topUp))
    }
    const l1Messenger = getContractFactory('L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(watcher.l1.messengerAddress)
    const ovmEth = getOvmEth(l2Wallet)
    const l2Bridge = await getL2Bridge(l2Wallet)
    const l2Messenger = getContractFactory('L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(watcher.l2.messengerAddress)

    const ctcAddress = await addressManager.getAddress(
      'CanonicalTransactionChain'
    )
    const ctc = getContractFactory('CanonicalTransactionChain')
      .connect(l1Wallet)
      .attach(ctcAddress)

    const gasPriceOracle = getContractFactory('OVM_GasPriceOracle')
      .connect(gasPriceOracleWallet)
      .attach(predeploys.OVM_GasPriceOracle)

    const sccAddress = await addressManager.getAddress('StateCommitmentChain')
    const scc = getContractFactory('StateCommitmentChain')
      .connect(l1Wallet)
      .attach(sccAddress)

    const sequencerFeeVault = getContractFactory('OVM_SequencerFeeVault')
      .connect(l2Wallet)
      .attach(predeploys.OVM_SequencerFeeVault)

    const l1BlockNumber = getContractFactory('iOVM_L1BlockNumber')
      .connect(l2Wallet)
      .attach(predeploys.OVM_L1BlockNumber)

    return new OptimismEnv({
      addressManager,
      l1Bridge,
      ctc,
      scc,
      l1Messenger,
      l1BlockNumber,
      ovmEth,
      gasPriceOracle,
      sequencerFeeVault,
      l2Bridge,
      l2Messenger,
      watcher,
      l1Wallet,
      l2Wallet,
      l1Provider,
      l2Provider,
      replicaProvider,
    })
  }

  async waitForXDomainTransaction(
    tx: Promise<TransactionResponse> | TransactionResponse,
    direction: Direction
  ): Promise<CrossDomainMessagePair> {
    return waitForXDomainTransaction(this.watcher, tx, direction)
  }

  /**
   * Relays all L2 => L1 messages found in a given L2 transaction.
   *
   * @param tx Transaction to find messages in.
   */
  async relayXDomainMessages(
    tx: Promise<TransactionResponse> | TransactionResponse
  ): Promise<void> {
    tx = await tx
    await tx.wait()

    let messagePairs = []
    while (true) {
      try {
        messagePairs = await getMessagesAndProofsForL2Transaction(
          l1Provider,
          l2Provider,
          this.scc.address,
          predeploys.L2CrossDomainMessenger,
          tx.hash
        )
        break
      } catch (err) {
        if (err.message.includes('unable to find state root batch for tx')) {
          await sleep(5000)
        } else {
          throw err
        }
      }
    }

    for (const { message, proof } of messagePairs) {
      while (true) {
        try {
          const result = await this.l1Messenger
            .connect(this.l1Wallet)
            .relayMessage(
              message.target,
              message.sender,
              message.message,
              message.messageNonce,
              proof,
              {
                gasLimit: DEFAULT_TEST_GAS_L1 * 10,
              }
            )
          await result.wait()
          break
        } catch (err) {
          if (err.message.includes('execution failed due to an exception')) {
            await sleep(5000)
          } else if (err.message.includes('Nonce too low')) {
            await sleep(5000)
          } else if (err.message.includes('transaction was replaced')) {
            // this happens when we run tests in parallel
            await sleep(5000)
          } else if (
            err.message.includes(
              'another transaction with same nonce in the queue'
            )
          ) {
            // this happens when we run tests in parallel
            await sleep(5000)
          } else if (
            err.message.includes('message has already been received')
          ) {
            break
          } else {
            throw err
          }
        }
      }
    }
  }
}
