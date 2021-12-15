/* eslint-disable @typescript-eslint/no-unused-vars */
import {
  Provider,
  BlockTag,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { BigNumber } from 'ethers'
import {
  ICrossChainProvider,
  OEContracts,
  OEContractsLike,
  MessageLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  ProviderLike,
  CrossChainMessage,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  MessageReceipt,
} from './interfaces'
import {
  toProvider,
  toBigNumber,
  toTransactionHash,
  DeepPartial,
  getAllOEContracts,
} from './utils'

export class CrossChainProvider implements ICrossChainProvider {
  public l1Provider: Provider
  public l2Provider: Provider
  public l1ChainId: number
  public contracts: OEContracts

  /**
   * Creates a new CrossChainProvider instance.
   *
   * @param opts Options for the provider.
   * @param opts.l1Provider Provider for the L1 chain, or a JSON-RPC url.
   * @param opts.l2Provider Provider for the L2 chain, or a JSON-RPC url.
   * @param opts.l1ChainId Chain ID for the L1 chain.
   * @param opts.contracts Optional contract address overrides.
   */
  constructor(opts: {
    l1Provider: ProviderLike
    l2Provider: ProviderLike
    l1ChainId: NumberLike
    contracts?: DeepPartial<OEContractsLike>
  }) {
    this.l1Provider = toProvider(opts.l1Provider)
    this.l2Provider = toProvider(opts.l2Provider)
    this.l1ChainId = toBigNumber(opts.l1ChainId).toNumber()
    this.contracts = getAllOEContracts(this.l1ChainId, {
      l1SignerOrProvider: this.l1Provider,
      l2SignerOrProvider: this.l2Provider,
      overrides: opts.contracts,
    })
  }

  public async getMessagesByTransaction(
    transaction: TransactionLike,
    opts: {
      direction?: MessageDirection
    } = {}
  ): Promise<CrossChainMessage[]> {
    const txHash = toTransactionHash(transaction)

    let receipt: TransactionReceipt
    if (opts.direction !== undefined) {
      // Get the receipt for the requested direction.
      if (opts.direction === MessageDirection.L1_TO_L2) {
        receipt = await this.l1Provider.getTransactionReceipt(txHash)
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
      }
    } else {
      // Try both directions, starting with L1 => L2.
      receipt = await this.l1Provider.getTransactionReceipt(txHash)
      if (receipt) {
        opts.direction = MessageDirection.L1_TO_L2
      } else {
        receipt = await this.l2Provider.getTransactionReceipt(txHash)
        opts.direction = MessageDirection.L2_TO_L1
      }
    }

    if (!receipt) {
      throw new Error(`unable to find transaction receipt for ${txHash}`)
    }

    // By this point opts.direction will always be defined.
    const messenger =
      opts.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l1.L1CrossDomainMessenger
        : this.contracts.l2.L2CrossDomainMessenger

    return receipt.logs
      .filter((log) => {
        // Only look at logs emitted by the messenger address
        return log.address === messenger.address
      })
      .filter((log) => {
        // Only look at SentMessage logs specifically
        const parsed = messenger.interface.parseLog(log)
        return parsed.name === 'SentMessage'
      })
      .map((log) => {
        // Convert each SentMessage log into a message object
        const parsed = messenger.interface.parseLog(log)
        return {
          direction: opts.direction,
          target: parsed.args.target,
          sender: parsed.args.sender,
          message: parsed.args.message,
          messageNonce: parsed.args.messageNonce,
        }
      })
  }

  public async getMessagesByAddress(
    address: AddressLike,
    opts?: {
      direction?: MessageDirection
      fromBlock?: NumberLike
      toBlock?: NumberLike
    }
  ): Promise<CrossChainMessage[]> {
    throw new Error('Not implemented')
  }

  public async getTokenBridgeMessagesByAddress(
    address: AddressLike,
    opts?: {
      direction?: MessageDirection
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]> {
    throw new Error('Not implemented')
  }

  public async getMessageStatus(message: MessageLike): Promise<MessageStatus> {
    throw new Error('Not implemented')
  }

  public async getMessageReceipt(
    message: MessageLike
  ): Promise<MessageReceipt> {
    throw new Error('Not implemented')
  }

  public async waitForMessageReciept(
    message: MessageLike,
    opts?: {
      confirmations?: number
      pollIntervalMs?: number
      timeoutMs?: number
    }
  ): Promise<MessageReceipt> {
    throw new Error('Not implemented')
  }

  public async estimateL2MessageGasLimit(
    message: MessageLike
  ): Promise<BigNumber> {
    throw new Error('Not implemented')
  }

  public async estimateMessageWaitTimeSeconds(
    message: MessageLike
  ): Promise<number> {
    throw new Error('Not implemented')
  }

  public async estimateMessageWaitTimeBlocks(
    message: MessageLike
  ): Promise<number> {
    throw new Error('Not implemented')
  }
}
