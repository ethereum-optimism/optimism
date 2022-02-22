/* Imports: External */
import { Contract, utils, Wallet, providers, Transaction } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/providers'
import { getContractFactory, predeploys } from '@eth-optimism/contracts'
import { sleep } from '@eth-optimism/core-utils'
import {
  CrossChainMessenger,
  MessageStatus,
  MessageDirection,
} from '@eth-optimism/sdk'

/* Imports: Internal */
import {
  getAddressManager,
  l1Provider,
  l2Provider,
  replicaProvider,
  verifierProvider,
  l1Wallet,
  l2Wallet,
  gasPriceOracleWallet,
  fundUser,
  getOvmEth,
  getL1Bridge,
  getL2Bridge,
  envConfig,
} from './utils'

export interface CrossDomainMessagePair {
  tx: Transaction
  receipt: TransactionReceipt
  remoteTx: Transaction
  remoteReceipt: TransactionReceipt
}

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

  // The wallets
  l1Wallet: Wallet
  l2Wallet: Wallet

  // The providers
  messenger: CrossChainMessenger
  l1Provider: providers.JsonRpcProvider
  l2Provider: providers.JsonRpcProvider
  replicaProvider: providers.JsonRpcProvider
  verifierProvider: providers.JsonRpcProvider

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
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.messenger = args.messenger
    this.l1Provider = args.l1Provider
    this.l2Provider = args.l2Provider
    this.replicaProvider = args.replicaProvider
    this.verifierProvider = args.verifierProvider
    this.ctc = args.ctc
    this.scc = args.scc
  }

  static async new(): Promise<OptimismEnv> {
    const network = await l1Provider.getNetwork()

    const addressManager = getAddressManager(l1Wallet)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    const l1MessengerAddress = await addressManager.getAddress(
      'Proxy__OVM_L1CrossDomainMessenger'
    )
    const l2MessengerAddress = await addressManager.getAddress(
      'L2CrossDomainMessenger'
    )
    const l1Messenger = getContractFactory('L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(l1MessengerAddress)
    const ovmEth = getOvmEth(l2Wallet)
    const l2Bridge = await getL2Bridge(l2Wallet)
    const l2Messenger = getContractFactory('L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(l2MessengerAddress)
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

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: l1Wallet,
      l2SignerOrProvider: l2Wallet,
      l1ChainId: network.chainId,
      contracts: {
        l1: {
          AddressManager: envConfig.ADDRESS_MANAGER,
          L1CrossDomainMessenger: l1Messenger.address,
          L1StandardBridge: l1Bridge.address,
          StateCommitmentChain: sccAddress,
          CanonicalTransactionChain: ctcAddress,
          BondManager: await addressManager.getAddress('BondManager'),
        },
      },
    })

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    const min = envConfig.L2_WALLET_MIN_BALANCE_ETH.toString()
    const topUp = envConfig.L2_WALLET_TOP_UP_AMOUNT_ETH.toString()
    if (balance.lt(utils.parseEther(min))) {
      await fundUser(messenger, utils.parseEther(topUp))
    }

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
      l1Wallet,
      l2Wallet,
      messenger,
      l1Provider,
      l2Provider,
      verifierProvider,
      replicaProvider,
    })
  }

  async waitForXDomainTransaction(
    tx: Promise<TransactionResponse> | TransactionResponse
  ): Promise<CrossDomainMessagePair> {
    // await it if needed
    tx = await tx

    const receipt = await tx.wait()
    const resolved = await this.messenger.toCrossChainMessage(tx)
    const messageReceipt = await this.messenger.waitForMessageReceipt(tx)
    let fullTx: any
    let remoteTx: any
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      fullTx = await this.messenger.l1Provider.getTransaction(tx.hash)
      remoteTx = await this.messenger.l2Provider.getTransaction(
        messageReceipt.transactionReceipt.transactionHash
      )
    } else {
      fullTx = await this.messenger.l2Provider.getTransaction(tx.hash)
      remoteTx = await this.messenger.l1Provider.getTransaction(
        messageReceipt.transactionReceipt.transactionHash
      )
    }

    return {
      tx: fullTx,
      receipt,
      remoteTx,
      remoteReceipt: messageReceipt.transactionReceipt,
    }
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

    const messages = await this.messenger.getMessagesByTransaction(tx)
    if (messages.length === 0) {
      return
    }

    for (const message of messages) {
      let status: MessageStatus
      while (
        status !== MessageStatus.READY_FOR_RELAY &&
        status !== MessageStatus.RELAYED
      ) {
        status = await this.messenger.getMessageStatus(message)
        await sleep(1000)
      }

      let relayed = false
      while (!relayed) {
        try {
          await this.messenger.finalizeMessage(message)
          relayed = true
        } catch (err) {
          if (
            err.message.includes('Nonce too low') ||
            err.message.includes('transaction was replaced') ||
            err.message.includes(
              'another transaction with same nonce in the queue'
            )
          ) {
            // Sometimes happens when we run tests in parallel.
            await sleep(5000)
          } else if (
            err.message.includes('message has already been received')
          ) {
            // Message already relayed, this is fine.
            relayed = true
          } else {
            throw err
          }
        }
      }

      await this.messenger.waitForMessageReceipt(message)
    }
  }
}
