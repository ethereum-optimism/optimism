/* eslint-disable @typescript-eslint/no-unused-vars */
import {
  Provider,
  BlockTag,
  TransactionReceipt,
} from '@ethersproject/abstract-provider'
import { ethers, BigNumber, Event } from 'ethers'
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
  CustomBridges,
  CustomBridgesLike,
} from './interfaces'
import {
  toProvider,
  toBigNumber,
  toTransactionHash,
  DeepPartial,
  getAllOEContracts,
  getCustomBridges,
} from './utils'

export class CrossChainProvider implements ICrossChainProvider {
  public l1Provider: Provider
  public l2Provider: Provider
  public l1ChainId: number
  public contracts: OEContracts
  public bridges: CustomBridges

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
    bridges?: Partial<CustomBridgesLike>
  }) {
    this.l1Provider = toProvider(opts.l1Provider)
    this.l2Provider = toProvider(opts.l2Provider)
    this.l1ChainId = toBigNumber(opts.l1ChainId).toNumber()
    this.contracts = getAllOEContracts(this.l1ChainId, {
      l1SignerOrProvider: this.l1Provider,
      l2SignerOrProvider: this.l2Provider,
      overrides: opts.contracts,
    })
    this.bridges = getCustomBridges(this.l1ChainId, {
      l1SignerOrProvider: this.l1Provider,
      l2SignerOrProvider: this.l2Provider,
      overrides: opts.bridges,
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
          logIndex: log.logIndex,
          blockNumber: log.blockNumber,
          transactionHash: log.transactionHash,
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
    opts: {
      direction?: MessageDirection
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    const parseTokenEvent = (
      event: Event,
      dir: MessageDirection
    ): TokenBridgeMessage => {
      return {
        direction: dir,
        from: event.args._from,
        to: event.args._to,
        l1Token: event.args._l1Token || ethers.constants.AddressZero,
        l2Token: event.args._l2Token || this.contracts.l2.OVM_ETH.address,
        amount: event.args._amount,
        data: event.args._data,
        logIndex: event.logIndex,
        blockNumber: event.blockNumber,
        transactionHash: event.transactionHash,
      }
    }

    // Make sure you provide a direction if you specify a block range. Block ranges don't make
    // sense to use on both chains at the same time.
    if (opts.fromBlock !== undefined || opts.toBlock !== undefined) {
      if (opts.direction === undefined) {
        throw new Error('direction must be specified when using a block range')
      }
    }

    // Keep track of all of the messages triggered by the address in question.
    // We'll add messages to this list as we find them, based on the direction that the user has
    // requested we find messages in. If the user hasn't requested a direction, we find messages in
    // both directions.
    const messages: TokenBridgeMessage[] = []

    // First find all messages in the L1 to L2 direction.
    if (
      opts.direction === undefined ||
      opts.direction === MessageDirection.L1_TO_L2
    ) {
      // Find all ETH deposit events and push them into the messages array.
      const ethDepositEvents =
        await this.contracts.l1.L1StandardBridge.queryFilter(
          this.contracts.l1.L1StandardBridge.filters.ETHDepositInitiated(
            address
          ),
          opts.fromBlock,
          opts.toBlock
        )
      for (const event of ethDepositEvents) {
        messages.push(parseTokenEvent(event, MessageDirection.L1_TO_L2))
      }

      // Send an event query for every L1 bridge, this will return an array of arrays.
      const erc20DepositEventSets = await Promise.all(
        [
          this.contracts.l1.L1StandardBridge,
          ...Object.values(this.bridges.l1),
        ].map(async (bridge) => {
          return bridge.queryFilter(
            bridge.filters.ERC20DepositInitiated(undefined, undefined, address),
            opts.fromBlock,
            opts.toBlock
          )
        })
      )

      for (const erc20DepositEvents of erc20DepositEventSets) {
        for (const event of erc20DepositEvents) {
          messages.push(parseTokenEvent(event, MessageDirection.L1_TO_L2))
        }
      }
    }

    // Next find all messages in the L2 to L1 direction.
    if (
      opts.direction === undefined ||
      opts.direction === MessageDirection.L2_TO_L1
    ) {
      // ETH withdrawals and ERC20 withdrawals are the same event on L2.
      // Send an event query for every L2 bridge, this will return an array of arrays.
      const withdrawalEventSets = await Promise.all(
        [
          this.contracts.l2.L2StandardBridge,
          ...Object.values(this.bridges.l2),
        ].map(async (bridge) => {
          return bridge.queryFilter(
            bridge.filters.WithdrawalInitiated(undefined, undefined, address),
            opts.fromBlock,
            opts.toBlock
          )
        })
      )

      for (const withdrawalEvents of withdrawalEventSets) {
        for (const event of withdrawalEvents) {
          messages.push(parseTokenEvent(event, MessageDirection.L2_TO_L1))
        }
      }
    }

    return messages
  }

  public async getDepositsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return this.getTokenBridgeMessagesByAddress(address, {
      ...opts,
      direction: MessageDirection.L1_TO_L2,
    })
  }

  public async getWithdrawalsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return this.getTokenBridgeMessagesByAddress(address, {
      ...opts,
      direction: MessageDirection.L2_TO_L1,
    })
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
