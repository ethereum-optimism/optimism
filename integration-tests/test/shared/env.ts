/* Imports: External */
import { utils, Wallet, providers, Transaction } from 'ethers'
import {
  TransactionResponse,
  TransactionReceipt,
} from '@ethersproject/providers'
import { getChainId, sleep } from '@eth-optimism/core-utils'
import {
  CrossChainMessenger,
  MessageStatus,
  MessageDirection,
  StandardBridgeAdapter,
  ETHBridgeAdapter,
  BridgeAdapterData,
} from '@eth-optimism/sdk'
import { predeploys } from '@eth-optimism/contracts'

/* Imports: Internal */
import {
  l1Provider,
  l2Provider,
  replicaProvider,
  verifierProvider,
  l1Wallet,
  l2Wallet,
  fundUser,
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
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.messenger = args.messenger
    this.l1Provider = args.l1Provider
    this.l2Provider = args.l2Provider
    this.replicaProvider = args.replicaProvider
    this.verifierProvider = args.verifierProvider
  }

  static async new(): Promise<OptimismEnv> {
    let bridgeOverrides: BridgeAdapterData
    if (envConfig.L1_STANDARD_BRIDGE) {
      bridgeOverrides = {
        Standard: {
          Adapter: StandardBridgeAdapter,
          l1Bridge: envConfig.L1_STANDARD_BRIDGE,
          l2Bridge: predeploys.L2StandardBridge,
        },
        ETH: {
          Adapter: ETHBridgeAdapter,
          l1Bridge: envConfig.L1_STANDARD_BRIDGE,
          l2Bridge: predeploys.L2StandardBridge,
        },
      }
    }

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: l1Wallet,
      l2SignerOrProvider: l2Wallet,
      l1ChainId: await getChainId(l1Provider),
      l2ChainId: await getChainId(l2Provider),
      contracts: {
        l1: {
          AddressManager: envConfig.ADDRESS_MANAGER,
          L1CrossDomainMessenger: envConfig.L1_CROSS_DOMAIN_MESSENGER,
          L1StandardBridge: envConfig.L1_STANDARD_BRIDGE,
          StateCommitmentChain: envConfig.STATE_COMMITMENT_CHAIN,
          CanonicalTransactionChain: envConfig.CANONICAL_TRANSACTION_CHAIN,
          BondManager: envConfig.BOND_MANAGER,
        },
      },
      bridges: bridgeOverrides,
    })

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    const min = envConfig.L2_WALLET_MIN_BALANCE_ETH.toString()
    const topUp = envConfig.L2_WALLET_TOP_UP_AMOUNT_ETH.toString()
    if (balance.lt(utils.parseEther(min))) {
      await fundUser(messenger, utils.parseEther(topUp))
    }

    return new OptimismEnv({
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
      await this.messenger.waitForMessageStatus(
        message,
        MessageStatus.READY_FOR_RELAY
      )

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
