/* eslint-disable @typescript-eslint/no-unused-vars */
import {
  Provider,
  BlockTag,
  TransactionReceipt,
  TransactionResponse,
  TransactionRequest,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import {
  ethers,
  BigNumber,
  Overrides,
  CallOverrides,
  PayableOverrides,
} from 'ethers'
import {
  sleep,
  remove0x,
  toHexString,
  toRpcHexString,
  encodeCrossDomainMessageV0,
  encodeCrossDomainMessageV1,
  BedrockOutputData,
  BedrockCrossChainMessageProof,
  decodeVersionedNonce,
  encodeVersionedNonce,
  getChainId,
  hashCrossDomainMessagev0,
  hashCrossDomainMessagev1,
} from '@eth-optimism/core-utils'
import { getContractInterface, predeploys } from '@eth-optimism/contracts'
import * as rlp from 'rlp'
import semver from 'semver'

import {
  OEContracts,
  OEContractsLike,
  MessageLike,
  MessageRequestLike,
  TransactionLike,
  AddressLike,
  NumberLike,
  SignerOrProviderLike,
  CrossChainMessage,
  CrossChainMessageRequest,
  CrossChainMessageProof,
  MessageDirection,
  MessageStatus,
  TokenBridgeMessage,
  MessageReceipt,
  MessageReceiptStatus,
  BridgeAdapterData,
  BridgeAdapters,
  StateRoot,
  StateRootBatch,
  IBridgeAdapter,
  ProvenWithdrawal,
  LowLevelMessage,
  FPACProvenWithdrawal,
} from './interfaces'
import {
  toSignerOrProvider,
  toNumber,
  toTransactionHash,
  DeepPartial,
  getAllOEContracts,
  getBridgeAdapters,
  makeMerkleTreeProof,
  makeStateTrieProof,
  hashLowLevelMessage,
  migratedWithdrawalGasLimit,
  DEPOSIT_CONFIRMATION_BLOCKS,
  CHAIN_BLOCK_TIMES,
  hashMessageHash,
  getContractInterfaceBedrock,
  toJsonRpcProvider,
} from './utils'

export class CrossChainMessenger {
  /**
   * Provider connected to the L1 chain.
   */
  public l1SignerOrProvider: Signer | Provider

  /**
   * Provider connected to the L2 chain.
   */
  public l2SignerOrProvider: Signer | Provider

  /**
   * Chain ID for the L1 network.
   */
  public l1ChainId: number

  /**
   * Chain ID for the L2 network.
   */
  public l2ChainId: number

  /**
   * Contract objects attached to their respective providers and addresses.
   */
  public contracts: OEContracts

  /**
   * List of custom bridges for the given network.
   */
  public bridges: BridgeAdapters

  /**
   * Number of blocks before a deposit is considered confirmed.
   */
  public depositConfirmationBlocks: number

  /**
   * Estimated average L1 block time in seconds.
   */
  public l1BlockTimeSeconds: number

  /**
   * Whether or not Bedrock compatibility is enabled.
   */
  public bedrock: boolean

  /**
   * Cache for output root validation. Output roots are expensive to verify, so we cache them.
   */
  private _outputCache: Array<{ root: string; valid: boolean }> = []

  /**
   * Creates a new CrossChainProvider instance.
   *
   * @param opts Options for the provider.
   * @param opts.l1SignerOrProvider Signer or Provider for the L1 chain, or a JSON-RPC url.
   * @param opts.l2SignerOrProvider Signer or Provider for the L2 chain, or a JSON-RPC url.
   * @param opts.l1ChainId Chain ID for the L1 chain.
   * @param opts.l2ChainId Chain ID for the L2 chain.
   * @param opts.depositConfirmationBlocks Optional number of blocks before a deposit is confirmed.
   * @param opts.l1BlockTimeSeconds Optional estimated block time in seconds for the L1 chain.
   * @param opts.contracts Optional contract address overrides.
   * @param opts.bridges Optional bridge address list.
   * @param opts.bedrock Whether or not to enable Bedrock compatibility.
   */
  constructor(opts: {
    l1SignerOrProvider: SignerOrProviderLike
    l2SignerOrProvider: SignerOrProviderLike
    l1ChainId: NumberLike
    l2ChainId: NumberLike
    depositConfirmationBlocks?: NumberLike
    l1BlockTimeSeconds?: NumberLike
    contracts?: DeepPartial<OEContractsLike>
    bridges?: BridgeAdapterData
    bedrock?: boolean
  }) {
    this.bedrock = opts.bedrock ?? true

    this.l1SignerOrProvider = toSignerOrProvider(opts.l1SignerOrProvider)
    this.l2SignerOrProvider = toSignerOrProvider(opts.l2SignerOrProvider)

    try {
      this.l1ChainId = toNumber(opts.l1ChainId)
    } catch (err) {
      throw new Error(`L1 chain ID is missing or invalid: ${opts.l1ChainId}`)
    }

    try {
      this.l2ChainId = toNumber(opts.l2ChainId)
    } catch (err) {
      throw new Error(`L2 chain ID is missing or invalid: ${opts.l2ChainId}`)
    }

    this.depositConfirmationBlocks =
      opts?.depositConfirmationBlocks !== undefined
        ? toNumber(opts.depositConfirmationBlocks)
        : DEPOSIT_CONFIRMATION_BLOCKS[this.l2ChainId] || 0

    this.l1BlockTimeSeconds =
      opts?.l1BlockTimeSeconds !== undefined
        ? toNumber(opts.l1BlockTimeSeconds)
        : CHAIN_BLOCK_TIMES[this.l1ChainId] || 1

    this.contracts = getAllOEContracts(this.l2ChainId, {
      l1SignerOrProvider: this.l1SignerOrProvider,
      l2SignerOrProvider: this.l2SignerOrProvider,
      overrides: opts.contracts,
    })

    this.bridges = getBridgeAdapters(this.l2ChainId, this, {
      overrides: opts.bridges,
      contracts: opts.contracts,
    })
  }

  /**
   * Provider connected to the L1 chain.
   */
  get l1Provider(): Provider {
    if (Provider.isProvider(this.l1SignerOrProvider)) {
      return this.l1SignerOrProvider
    } else {
      return this.l1SignerOrProvider.provider
    }
  }

  /**
   * Provider connected to the L2 chain.
   */
  get l2Provider(): Provider {
    if (Provider.isProvider(this.l2SignerOrProvider)) {
      return this.l2SignerOrProvider
    } else {
      return this.l2SignerOrProvider.provider
    }
  }

  /**
   * Signer connected to the L1 chain.
   */
  get l1Signer(): Signer {
    if (Provider.isProvider(this.l1SignerOrProvider)) {
      throw new Error(`messenger has no L1 signer`)
    } else {
      return this.l1SignerOrProvider
    }
  }

  /**
   * Signer connected to the L2 chain.
   */
  get l2Signer(): Signer {
    if (Provider.isProvider(this.l2SignerOrProvider)) {
      throw new Error(`messenger has no L2 signer`)
    } else {
      return this.l2SignerOrProvider
    }
  }

  /**
   * Uses portal version to determine if the messenger is using fpac contracts. Better not to cache
   * this value as it will change during the fpac upgrade and we want clients to automatically
   * begin using the new logic without throwing any errors.
   *
   * @returns Whether or not the messenger is using fpac contracts.
   */
  public async fpac(): Promise<boolean> {
    if (
      this.contracts.l1.OptimismPortal.address === ethers.constants.AddressZero
    ) {
      // Only really relevant for certain SDK tests where the portal is not deployed. We should
      // probably just update the tests so the portal gets deployed but feels like it's out of
      // scope for the FPAC changes.
      return false
    } else {
      return semver.gte(
        await this.contracts.l1.OptimismPortal.version(),
        '3.0.0'
      )
    }
  }

  /**
   * Retrieves all cross chain messages sent within a given transaction.
   *
   * @param transaction Transaction hash or receipt to find messages from.
   * @param opts Options object.
   * @param opts.direction Direction to search for messages in. If not provided, will attempt to
   * automatically search both directions under the assumption that a transaction hash will only
   * exist on one chain. If the hash exists on both chains, will throw an error.
   * @returns All cross chain messages sent within the transaction.
   */
  public async getMessagesByTransaction(
    transaction: TransactionLike,
    opts: {
      direction?: MessageDirection
    } = {}
  ): Promise<CrossChainMessage[]> {
    // Wait for the transaction receipt if the input is waitable.
    await (transaction as TransactionResponse).wait?.()

    // Convert the input to a transaction hash.
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
        // Try to pull out the value field, but only if the very next log is a SentMessageExtension1
        // event which was introduced in the Bedrock upgrade.
        let value = ethers.BigNumber.from(0)
        const next = receipt.logs.find((l) => {
          return (
            l.logIndex === log.logIndex + 1 && l.address === messenger.address
          )
        })
        if (next) {
          const nextParsed = messenger.interface.parseLog(next)
          if (nextParsed.name === 'SentMessageExtension1') {
            value = nextParsed.args.value
          }
        }

        // Convert each SentMessage log into a message object
        const parsed = messenger.interface.parseLog(log)
        return {
          direction: opts.direction,
          target: parsed.args.target,
          sender: parsed.args.sender,
          message: parsed.args.message,
          messageNonce: parsed.args.messageNonce,
          value,
          minGasLimit: parsed.args.gasLimit,
          logIndex: log.logIndex,
          blockNumber: log.blockNumber,
          transactionHash: log.transactionHash,
        }
      })
  }

  /**
   * Transforms a legacy message into its corresponding Bedrock representation.
   *
   * @param message Legacy message to transform.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Bedrock representation of the message.
   */
  public async toBedrockCrossChainMessage(
    message: MessageLike,
    messageIndex = 0
  ): Promise<CrossChainMessage> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)

    // Bedrock messages are already in the correct format.
    const { version } = decodeVersionedNonce(resolved.messageNonce)
    if (version.eq(1)) {
      return resolved
    }

    let value = BigNumber.from(0)
    if (
      resolved.direction === MessageDirection.L2_TO_L1 &&
      resolved.sender === this.contracts.l2.L2StandardBridge.address &&
      resolved.target === this.contracts.l1.L1StandardBridge.address
    ) {
      try {
        ;[, , value] =
          this.contracts.l1.L1StandardBridge.interface.decodeFunctionData(
            'finalizeETHWithdrawal',
            resolved.message
          )
      } catch (err) {
        // No problem, not a message with value.
      }
    }

    return {
      ...resolved,
      value,
      minGasLimit: BigNumber.from(0),
      messageNonce: encodeVersionedNonce(
        BigNumber.from(0),
        resolved.messageNonce
      ),
    }
  }

  /**
   * Transforms a CrossChainMessenger message into its low-level representation inside the
   * L2ToL1MessagePasser contract on L2.
   *
   * @param message Message to transform.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @return Transformed message.
   */
  public async toLowLevelMessage(
    message: MessageLike,
    messageIndex = 0
  ): Promise<LowLevelMessage> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`can only convert L2 to L1 messages to low level`)
    }

    // We may have to update the message if it's a legacy message.
    const { version } = decodeVersionedNonce(resolved.messageNonce)
    let updated: CrossChainMessage
    if (version.eq(0)) {
      updated = await this.toBedrockCrossChainMessage(resolved, messageIndex)
    } else {
      updated = resolved
    }

    // Encode the updated message, we need this for legacy messages.
    const encoded = encodeCrossDomainMessageV1(
      updated.messageNonce,
      updated.sender,
      updated.target,
      updated.value,
      updated.minGasLimit,
      updated.message
    )

    // EVERYTHING following here is basically repeating the logic from getMessagesByTransaction
    // consider cleaning this up
    // We need to figure out the final withdrawal data that was used to compute the withdrawal hash
    // inside the L2ToL1Message passer contract. Exact mechanism here depends on whether or not
    // this is a legacy message or a new Bedrock message.
    let gasLimit: BigNumber
    let messageNonce: BigNumber
    if (version.eq(0)) {
      const chainID = await getChainId(this.l2Provider)
      gasLimit = migratedWithdrawalGasLimit(encoded, chainID)
      messageNonce = resolved.messageNonce
    } else {
      const receipt = await this.l2Provider.getTransactionReceipt(
        (
          await this.toCrossChainMessage(message)
        ).transactionHash
      )

      const withdrawals: ethers.utils.Result[] = []
      for (const log of receipt.logs) {
        if (log.address === this.contracts.l2.BedrockMessagePasser.address) {
          const decoded =
            this.contracts.l2.L2ToL1MessagePasser.interface.parseLog(log)
          if (decoded.name === 'MessagePassed') {
            withdrawals.push(decoded.args)
          }
        }
      }

      // Should not happen.
      if (withdrawals.length === 0) {
        throw new Error(`no withdrawals found in receipt`)
      }

      const withdrawal = withdrawals[messageIndex]
      if (!withdrawal) {
        throw new Error(
          `withdrawal index ${messageIndex} out of bounds there are ${withdrawals.length} withdrawals`
        )
      }
      messageNonce = withdrawal.nonce
      gasLimit = withdrawal.gasLimit
    }

    return {
      messageNonce,
      sender: this.contracts.l2.L2CrossDomainMessenger.address,
      target: this.contracts.l1.L1CrossDomainMessenger.address,
      value: updated.value,
      minGasLimit: gasLimit,
      message: encoded,
    }
  }

  // public async getMessagesByAddress(
  //   address: AddressLike,
  //   opts?: {
  //     direction?: MessageDirection
  //     fromBlock?: NumberLike
  //     toBlock?: NumberLike
  //   }
  // ): Promise<CrossChainMessage[]> {
  //   throw new Error(`
  //     The function getMessagesByAddress is currently not enabled because the sender parameter of
  //     the SentMessage event is not indexed within the CrossChainMessenger contracts.
  //     getMessagesByAddress will be enabled by plugging in an Optimism Indexer (coming soon).
  //     See the following issue on GitHub for additional context:
  //     https://github.com/ethereum-optimism/optimism/issues/2129
  //   `)
  // }

  /**
   * Finds the appropriate bridge adapter for a given L1<>L2 token pair. Will throw if no bridges
   * support the token pair or if more than one bridge supports the token pair.
   *
   * @param l1Token L1 token address.
   * @param l2Token L2 token address.
   * @returns The appropriate bridge adapter for the given token pair.
   */
  public async getBridgeForTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<IBridgeAdapter> {
    const bridges: IBridgeAdapter[] = []
    for (const bridge of Object.values(this.bridges)) {
      try {
        if (await bridge.supportsTokenPair(l1Token, l2Token)) {
          bridges.push(bridge)
        }
      } catch (err) {
        if (
          !err?.message?.toString().includes('CALL_EXCEPTION') &&
          !err?.stack?.toString().includes('execution reverted')
        ) {
          console.error('Unexpected error when checking bridge', err)
        }
      }
    }

    if (bridges.length === 0) {
      throw new Error(`no supported bridge for token pair`)
    }

    if (bridges.length > 1) {
      throw new Error(`found more than one bridge for token pair`)
    }

    return bridges[0]
  }

  /**
   * Gets all deposits for a given address.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All deposit token bridge messages sent by the given address.
   */
  public async getDepositsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return (
      await Promise.all(
        Object.values(this.bridges).map(async (bridge) => {
          return bridge.getDepositsByAddress(address, opts)
        })
      )
    )
      .reduce((acc, val) => {
        return acc.concat(val)
      }, [])
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  /**
   * Gets all withdrawals for a given address.
   *
   * @param address Address to search for messages from.
   * @param opts Options object.
   * @param opts.fromBlock Block to start searching for messages from. If not provided, will start
   * from the first block (block #0).
   * @param opts.toBlock Block to stop searching for messages at. If not provided, will stop at the
   * latest known block ("latest").
   * @returns All withdrawal token bridge messages sent by the given address.
   */
  public async getWithdrawalsByAddress(
    address: AddressLike,
    opts: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    } = {}
  ): Promise<TokenBridgeMessage[]> {
    return (
      await Promise.all(
        Object.values(this.bridges).map(async (bridge) => {
          return bridge.getWithdrawalsByAddress(address, opts)
        })
      )
    )
      .reduce((acc, val) => {
        return acc.concat(val)
      }, [])
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  /**
   * Resolves a MessageLike into a CrossChainMessage object.
   * Unlike other coercion functions, this function is stateful and requires making additional
   * requests. For now I'm going to keep this function here, but we could consider putting a
   * similar function inside of utils/coercion.ts if people want to use this without having to
   * create an entire CrossChainProvider object.
   *
   * @param message MessageLike to resolve into a CrossChainMessage.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Message coerced into a CrossChainMessage.
   */
  public async toCrossChainMessage(
    message: MessageLike,
    messageIndex = 0
  ): Promise<CrossChainMessage> {
    if (!message) {
      throw new Error('message is undefined')
    }
    // TODO: Convert these checks into proper type checks.
    if ((message as CrossChainMessage).message) {
      return message as CrossChainMessage
    } else if (
      (message as TokenBridgeMessage).l1Token &&
      (message as TokenBridgeMessage).l2Token &&
      (message as TokenBridgeMessage).transactionHash
    ) {
      const messages = await this.getMessagesByTransaction(
        (message as TokenBridgeMessage).transactionHash
      )

      // The `messages` object corresponds to a list of SentMessage events that were triggered by
      // the same transaction. We want to find the specific SentMessage event that corresponds to
      // the TokenBridgeMessage (either a ETHDepositInitiated, ERC20DepositInitiated, or
      // WithdrawalInitiated event). We expect the behavior of bridge contracts to be that these
      // TokenBridgeMessage events are triggered and then a SentMessage event is triggered. Our
      // goal here is therefore to find the first SentMessage event that comes after the input
      // event.
      const found = messages
        .sort((a, b) => {
          // Sort all messages in ascending order by log index.
          return a.logIndex - b.logIndex
        })
        .find((m) => {
          return m.logIndex > (message as TokenBridgeMessage).logIndex
        })

      if (!found) {
        throw new Error(`could not find SentMessage event for message`)
      }

      return found
    } else {
      // TODO: Explicit TransactionLike check and throw if not TransactionLike
      const messages = await this.getMessagesByTransaction(
        message as TransactionLike
      )

      const out = messages[messageIndex]
      if (!out) {
        throw new Error(
          `withdrawal index ${messageIndex} out of bounds. There are ${messages.length} withdrawals`
        )
      }
      return out
    }
  }

  /**
   * Retrieves the status of a particular message as an enum.
   *
   * @param message Cross chain message to check the status of.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @param fromBlockOrBlockHash The start block to use for the query filter on the RECEIVING chain
   * @param toBlockOrBlockHash The end block to use for the query filter on the RECEIVING chain
   * @returns Status of the message.
   */
  public async getMessageStatus(
    message: MessageLike,
    // consider making this an options object next breaking release
    messageIndex = 0,
    /**
     * @deprecated no longer used since no log filters are used
     */
    fromBlockOrBlockHash?: BlockTag,
    /**
     * @deprecated no longer used since no log filters are used
     */
    toBlockOrBlockHash?: BlockTag
  ): Promise<MessageStatus> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    // legacy withdrawals relayed prebedrock are v1
    const messageHashV0 = hashCrossDomainMessagev0(
      resolved.target,
      resolved.sender,
      resolved.message,
      resolved.messageNonce
    )
    // bedrock withdrawals are v1
    // legacy withdrawals relayed postbedrock are v1
    // there is no good way to differentiate between the two types of legacy
    // so what we will check for both
    const messageHashV1 = hashCrossDomainMessagev1(
      resolved.messageNonce,
      resolved.sender,
      resolved.target,
      resolved.value,
      resolved.minGasLimit,
      resolved.message
    )

    // Here we want the messenger that will receive the message, not the one that sent it.
    const messenger =
      resolved.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l2.L2CrossDomainMessenger
        : this.contracts.l1.L1CrossDomainMessenger

    const success =
      (await messenger.successfulMessages(messageHashV0)) ||
      (await messenger.successfulMessages(messageHashV1))

    // Avoid the extra query if we already know the message was successful.
    if (success) {
      return MessageStatus.RELAYED
    }

    const failure =
      (await messenger.failedMessages(messageHashV0)) ||
      (await messenger.failedMessages(messageHashV1))

    if (resolved.direction === MessageDirection.L1_TO_L2) {
      if (failure) {
        return MessageStatus.FAILED_L1_TO_L2_MESSAGE
      } else {
        return MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE
      }
    } else {
      if (failure) {
        return MessageStatus.READY_FOR_RELAY
      } else {
        let timestamp: number
        if (this.bedrock) {
          const output = await this.getMessageBedrockOutput(
            resolved,
            messageIndex
          )
          if (output === null) {
            return MessageStatus.STATE_ROOT_NOT_PUBLISHED
          }

          // Convert the message to the low level message that was proven.
          const withdrawal = await this.toLowLevelMessage(
            resolved,
            messageIndex
          )

          // Attempt to fetch the proven withdrawal.
          const provenWithdrawal = await this.getProvenWithdrawal(
            hashLowLevelMessage(withdrawal)
          )

          // If the withdrawal hash has not been proven on L1, return READY_TO_PROVE.
          // Note that this will also apply in the case that a withdrawal has been proven but the
          // proposal used to create the proof was invalidated. This is fine because in that case
          // the withdrawal needs to be proven again anyway.
          if (provenWithdrawal === null) {
            return MessageStatus.READY_TO_PROVE
          }

          // Set the timestamp to the provenWithdrawal's timestamp
          timestamp = provenWithdrawal.timestamp.toNumber()
        } else {
          const stateRoot = await this.getMessageStateRoot(
            resolved,
            messageIndex
          )
          if (stateRoot === null) {
            return MessageStatus.STATE_ROOT_NOT_PUBLISHED
          }

          const bn = stateRoot.batch.blockNumber
          const block = await this.l1Provider.getBlock(bn)
          timestamp = block.timestamp
        }

        if (await this.fpac()) {
          // Convert the message to the low level message that was proven.
          const withdrawal = await this.toLowLevelMessage(
            resolved,
            messageIndex
          )

          // Get the withdrawal hash.
          const withdrawalHash = hashLowLevelMessage(withdrawal)

          // Grab the proven withdrawal data.
          const provenWithdrawal = await this.getProvenWithdrawal(
            withdrawalHash
          )

          // Sanity check, should've already happened above but do it just in case.
          if (provenWithdrawal === null) {
            // Ready to prove is the correct status here, we would not expect to hit this code path
            // unless there was an unexpected reorg on L1. Since this is unlikely we log a warning.
            console.warn(
              'Unexpected code path reached in getMessageStatus, returning READY_TO_PROVE'
            )
            return MessageStatus.READY_TO_PROVE
          }

          // Shouldn't happen, but worth checking just in case.
          if (!('proofSubmitter' in provenWithdrawal)) {
            throw new Error(
              `expected to get FPAC withdrawal but got legacy withdrawal`
            )
          }

          try {
            // If this doesn't revert then we should be fine to relay.
            await this.contracts.l1.OptimismPortal2.checkWithdrawal(
              hashLowLevelMessage(withdrawal),
              provenWithdrawal.proofSubmitter
            )

            return MessageStatus.READY_FOR_RELAY
          } catch (err) {
            return MessageStatus.IN_CHALLENGE_PERIOD
          }
        } else {
          const challengePeriod = await this.getChallengePeriodSeconds()
          const latestBlock = await this.l1Provider.getBlock('latest')

          if (timestamp + challengePeriod > latestBlock.timestamp) {
            return MessageStatus.IN_CHALLENGE_PERIOD
          } else {
            return MessageStatus.READY_FOR_RELAY
          }
        }
      }
    }
  }

  /**
   * Finds the receipt of the transaction that executed a particular cross chain message.
   *
   * @param message Message to find the receipt of.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @param fromBlockOrBlockHash The start block to use for the query filter on the RECEIVING chain
   * @param toBlockOrBlockHash The end block to use for the query filter on the RECEIVING chain
   * @returns CrossChainMessage receipt including receipt of the transaction that relayed the
   * given message.
   */
  public async getMessageReceipt(
    message: MessageLike,
    messageIndex = 0,
    fromBlockOrBlockHash?: BlockTag,
    toBlockOrHash?: BlockTag
  ): Promise<MessageReceipt> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    // legacy withdrawals relayed prebedrock are v1
    const messageHashV0 = hashCrossDomainMessagev0(
      resolved.target,
      resolved.sender,
      resolved.message,
      resolved.messageNonce
    )
    // bedrock withdrawals are v1
    // legacy withdrawals relayed postbedrock are v1
    // there is no good way to differentiate between the two types of legacy
    // so what we will check for both
    const messageHashV1 = hashCrossDomainMessagev1(
      resolved.messageNonce,
      resolved.sender,
      resolved.target,
      resolved.value,
      resolved.minGasLimit,
      resolved.message
    )

    // Here we want the messenger that will receive the message, not the one that sent it.
    const messenger =
      resolved.direction === MessageDirection.L1_TO_L2
        ? this.contracts.l2.L2CrossDomainMessenger
        : this.contracts.l1.L1CrossDomainMessenger

    // this is safe because we can guarantee only one of these filters max will return something
    const relayedMessageEvents = [
      ...(await messenger.queryFilter(
        messenger.filters.RelayedMessage(messageHashV0),
        fromBlockOrBlockHash,
        toBlockOrHash
      )),
      ...(await messenger.queryFilter(
        messenger.filters.RelayedMessage(messageHashV1),
        fromBlockOrBlockHash,
        toBlockOrHash
      )),
    ]

    // Great, we found the message. Convert it into a transaction receipt.
    if (relayedMessageEvents.length === 1) {
      return {
        receiptStatus: MessageReceiptStatus.RELAYED_SUCCEEDED,
        transactionReceipt:
          await relayedMessageEvents[0].getTransactionReceipt(),
      }
    } else if (relayedMessageEvents.length > 1) {
      // Should never happen!
      throw new Error(`multiple successful relays for message`)
    }

    // We didn't find a transaction that relayed the message. We now attempt to find
    // FailedRelayedMessage events instead.
    const failedRelayedMessageEvents = [
      ...(await messenger.queryFilter(
        messenger.filters.FailedRelayedMessage(messageHashV0),
        fromBlockOrBlockHash,
        toBlockOrHash
      )),
      ...(await messenger.queryFilter(
        messenger.filters.FailedRelayedMessage(messageHashV1),
        fromBlockOrBlockHash,
        toBlockOrHash
      )),
    ]

    // A transaction can fail to be relayed multiple times. We'll always return the last
    // transaction that attempted to relay the message.
    // TODO: Is this the best way to handle this?
    if (failedRelayedMessageEvents.length > 0) {
      return {
        receiptStatus: MessageReceiptStatus.RELAYED_FAILED,
        transactionReceipt: await failedRelayedMessageEvents[
          failedRelayedMessageEvents.length - 1
        ].getTransactionReceipt(),
      }
    }

    // TODO: If the user doesn't provide enough gas then there's a chance that FailedRelayedMessage
    // will never be triggered. We should probably fix this at the contract level by requiring a
    // minimum amount of input gas and designing the contracts such that the gas will always be
    // enough to trigger the event. However, for now we need a temporary way to find L1 => L2
    // transactions that fail but don't alert us because they didn't provide enough gas.
    // TODO: Talk with the systems and protocol team about coordinating a hard fork that fixes this
    // on both L1 and L2.

    // Just return null if we didn't find a receipt. Slightly nicer than throwing an error.
    return null
  }

  /**
   * Waits for a message to be executed and returns the receipt of the transaction that executed
   * the given message.
   *
   * @param message Message to wait for.
   * @param opts Options to pass to the waiting function.
   * @param opts.confirmations Number of transaction confirmations to wait for before returning.
   * @param opts.pollIntervalMs Number of milliseconds to wait between polling for the receipt.
   * @param opts.timeoutMs Milliseconds to wait before timing out.
   * @param opts.fromBlockOrBlockHash The start block to use for the query filter on the RECEIVING chain
   * @param opts.toBlockOrBlockHash The end block to use for the query filter on the RECEIVING chain
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns CrossChainMessage receipt including receipt of the transaction that relayed the
   * given message.
   */
  public async waitForMessageReceipt(
    message: MessageLike,
    opts: {
      fromBlockOrBlockHash?: BlockTag
      toBlockOrHash?: BlockTag
      confirmations?: number
      pollIntervalMs?: number
      timeoutMs?: number
    } = {},

    /**
     * The index of the withdrawal if multiple are made with multicall
     */
    messageIndex = 0
  ): Promise<MessageReceipt> {
    // Resolving once up-front is slightly more efficient.
    const resolved = await this.toCrossChainMessage(message, messageIndex)

    let totalTimeMs = 0
    while (totalTimeMs < (opts.timeoutMs || Infinity)) {
      const tick = Date.now()
      const receipt = await this.getMessageReceipt(
        resolved,
        messageIndex,
        opts.fromBlockOrBlockHash,
        opts.toBlockOrHash
      )
      if (receipt !== null) {
        return receipt
      } else {
        await sleep(opts.pollIntervalMs || 4000)
        totalTimeMs += Date.now() - tick
      }
    }

    throw new Error(`timed out waiting for message receipt`)
  }

  /**
   * Waits until the status of a given message changes to the expected status. Note that if the
   * status of the given message changes to a status that implies the expected status, this will
   * still return. If the status of the message changes to a status that exclues the expected
   * status, this will throw an error.
   *
   * @param message Message to wait for.
   * @param status Expected status of the message.
   * @param opts Options to pass to the waiting function.
   * @param opts.pollIntervalMs Number of milliseconds to wait when polling.
   * @param opts.timeoutMs Milliseconds to wait before timing out.
   * @param opts.fromBlockOrBlockHash The start block to use for the query filter on the RECEIVING chain
   * @param opts.toBlockOrBlockHash The end block to use for the query filter on the RECEIVING chain
   * @param messageIndex The index of the message, if multiple exist from multicall
   */
  public async waitForMessageStatus(
    message: MessageLike,
    status: MessageStatus,
    opts: {
      fromBlockOrBlockHash?: BlockTag
      toBlockOrBlockHash?: BlockTag
      pollIntervalMs?: number
      timeoutMs?: number
    } = {},
    messageIndex = 0
  ): Promise<void> {
    // Resolving once up-front is slightly more efficient.
    const resolved = await this.toCrossChainMessage(message, messageIndex)

    let totalTimeMs = 0
    while (totalTimeMs < (opts.timeoutMs || Infinity)) {
      const tick = Date.now()
      const currentStatus = await this.getMessageStatus(
        resolved,
        messageIndex,
        opts.fromBlockOrBlockHash,
        opts.toBlockOrBlockHash
      )

      // Handle special cases for L1 to L2 messages.
      if (resolved.direction === MessageDirection.L1_TO_L2) {
        // If we're at the expected status, we're done.
        if (currentStatus === status) {
          return
        }

        if (
          status === MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE &&
          currentStatus > status
        ) {
          // Anything other than UNCONFIRMED_L1_TO_L2_MESSAGE implies that the message was at one
          // point "unconfirmed", so we can stop waiting.
          return
        }

        if (
          status === MessageStatus.FAILED_L1_TO_L2_MESSAGE &&
          currentStatus === MessageStatus.RELAYED
        ) {
          throw new Error(
            `incompatible message status, expected FAILED_L1_TO_L2_MESSAGE got RELAYED`
          )
        }

        if (
          status === MessageStatus.RELAYED &&
          currentStatus === MessageStatus.FAILED_L1_TO_L2_MESSAGE
        ) {
          throw new Error(
            `incompatible message status, expected RELAYED got FAILED_L1_TO_L2_MESSAGE`
          )
        }
      }

      // Handle special cases for L2 to L1 messages.
      if (resolved.direction === MessageDirection.L2_TO_L1) {
        if (currentStatus >= status) {
          // For L2 to L1 messages, anything after the expected status implies the previous status,
          // so we can safely return if the current status enum is larger than the expected one.
          return
        }
      }

      await sleep(opts.pollIntervalMs || 4000)
      totalTimeMs += Date.now() - tick
    }

    throw new Error(`timed out waiting for message status change`)
  }

  /**
   * Estimates the amount of gas required to fully execute a given message on L2. Only applies to
   * L1 => L2 messages. You would supply this gas limit when sending the message to L2.
   *
   * @param message Message get a gas estimate for.
   * @param opts Options object.
   * @param opts.bufferPercent Percentage of gas to add to the estimate. Defaults to 20.
   * @param opts.from Address to use as the sender.
   * @returns Estimates L2 gas limit.
   */
  public async estimateL2MessageGasLimit(
    message: MessageRequestLike,
    opts?: {
      bufferPercent?: number
      from?: string
    },
    messageIndex = 0
  ): Promise<BigNumber> {
    let resolved: CrossChainMessage | CrossChainMessageRequest
    let from: string
    if ((message as CrossChainMessage).messageNonce === undefined) {
      resolved = message as CrossChainMessageRequest
      from = opts?.from
    } else {
      resolved = await this.toCrossChainMessage(
        message as MessageLike,
        messageIndex
      )
      from = opts?.from || (resolved as CrossChainMessage).sender
    }

    // L2 message gas estimation is only used for L1 => L2 messages.
    if (resolved.direction === MessageDirection.L2_TO_L1) {
      throw new Error(`cannot estimate gas limit for L2 => L1 message`)
    }

    const estimate = await this.l2Provider.estimateGas({
      from,
      to: resolved.target,
      data: resolved.message,
    })

    // Return the estimate plus a buffer of 20% just in case.
    const bufferPercent = opts?.bufferPercent || 20
    return estimate.mul(100 + bufferPercent).div(100)
  }

  /**
   * Returns the estimated amount of time before the message can be executed. When this is a
   * message being sent to L1, this will return the estimated time until the message will complete
   * its challenge period. When this is a message being sent to L2, this will return the estimated
   * amount of time until the message will be picked up and executed on L2.
   *
   * @param message Message to estimate the time remaining for.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @param opts.fromBlockOrBlockHash The start block to use for the query filter on the RECEIVING chain
   * @param opts.toBlockOrBlockHash The end block to use for the query filter on the RECEIVING chain
   * @returns Estimated amount of time remaining (in seconds) before the message can be executed.
   */
  public async estimateMessageWaitTimeSeconds(
    message: MessageLike,
    // consider making this an options object next breaking release
    messageIndex = 0,
    fromBlockOrBlockHash?: BlockTag,
    toBlockOrBlockHash?: BlockTag
  ): Promise<number> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    const status = await this.getMessageStatus(
      resolved,
      messageIndex,
      fromBlockOrBlockHash,
      toBlockOrBlockHash
    )
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      if (
        status === MessageStatus.RELAYED ||
        status === MessageStatus.FAILED_L1_TO_L2_MESSAGE
      ) {
        // Transactions that are relayed or failed are considered completed, so the wait time is 0.
        return 0
      } else {
        // Otherwise we need to estimate the number of blocks left until the transaction will be
        // considered confirmed by the Layer 2 system. Then we multiply this by the estimated
        // average L1 block time.
        const receipt = await this.l1Provider.getTransactionReceipt(
          resolved.transactionHash
        )
        const blocksLeft = Math.max(
          this.depositConfirmationBlocks - receipt.confirmations,
          0
        )
        return blocksLeft * this.l1BlockTimeSeconds
      }
    } else {
      if (
        status === MessageStatus.RELAYED ||
        status === MessageStatus.READY_FOR_RELAY
      ) {
        // Transactions that are relayed or ready for relay are considered complete.
        return 0
      } else if (status === MessageStatus.STATE_ROOT_NOT_PUBLISHED) {
        // If the state root hasn't been published yet, just assume it'll be published relatively
        // quickly and return the challenge period for now. In the future we could use more
        // advanced techniques to figure out average time between transaction execution and
        // state root publication.
        return this.getChallengePeriodSeconds()
      } else if (status === MessageStatus.IN_CHALLENGE_PERIOD) {
        // If the message is still within the challenge period, then we need to estimate exactly
        // the amount of time left until the challenge period expires. The challenge period starts
        // when the state root is published.
        const stateRoot = await this.getMessageStateRoot(resolved, messageIndex)
        const challengePeriod = await this.getChallengePeriodSeconds()
        const targetBlock = await this.l1Provider.getBlock(
          stateRoot.batch.blockNumber
        )
        const latestBlock = await this.l1Provider.getBlock('latest')
        return Math.max(
          challengePeriod - (latestBlock.timestamp - targetBlock.timestamp),
          0
        )
      } else {
        // Should not happen
        throw new Error(`unexpected message status`)
      }
    }
  }

  /**
   * Queries the current challenge period in seconds from the StateCommitmentChain.
   *
   * @returns Current challenge period in seconds.
   */
  public async getChallengePeriodSeconds(): Promise<number> {
    if (!this.bedrock) {
      return (
        await this.contracts.l1.StateCommitmentChain.FRAUD_PROOF_WINDOW()
      ).toNumber()
    }

    const oracleVersion = await this.contracts.l1.L2OutputOracle.version()
    const challengePeriod =
      oracleVersion === '1.0.0'
        ? // The ABI in the SDK does not contain FINALIZATION_PERIOD_SECONDS
          // in OptimismPortal, so making an explicit call instead.
          BigNumber.from(
            await this.contracts.l1.OptimismPortal.provider.call({
              to: this.contracts.l1.OptimismPortal.address,
              data: '0xf4daa291', // FINALIZATION_PERIOD_SECONDS
            })
          )
        : await this.contracts.l1.L2OutputOracle.FINALIZATION_PERIOD_SECONDS()
    return challengePeriod.toNumber()
  }

  /**
   * Queries the OptimismPortal contract's `provenWithdrawals` mapping
   * for a ProvenWithdrawal that matches the passed withdrawalHash
   *
   * @bedrock
   * Note: This function is bedrock-specific.
   *
   * @returns A ProvenWithdrawal object
   */
  public async getProvenWithdrawal(
    withdrawalHash: string
  ): Promise<ProvenWithdrawal | null> {
    if (!this.bedrock) {
      throw new Error('message proving only applies after the bedrock upgrade')
    }

    // Getting the withdrawal is easy before FPAC.
    if (!(await this.fpac())) {
      // Grab the proven withdrawal directly by hash.
      const provenWithdrawal =
        await this.contracts.l1.OptimismPortal.provenWithdrawals(withdrawalHash)

      // If the timestamp is 0 then the withdrawal has not been proven.
      if (provenWithdrawal.timestamp.eq(0)) {
        return null
      } else {
        return provenWithdrawal
      }
    }

    // Getting the withdrawal is a bit more complicated after FPAC.
    // First we need to get the number of proof submitters for this withdrawal.
    const numProofSubmitters = BigNumber.from(
      await this.contracts.l1.OptimismPortal2.numProofSubmitters(withdrawalHash)
    ).toNumber()

    // Now we need to find any withdrawal where the output proposal that the withdrawal was proven
    // against is actually valid. We can use the same output validation cache used elsewhere.
    for (let i = 0; i < numProofSubmitters; i++) {
      // Grab the proof submitter.
      const proofSubmitter =
        await this.contracts.l1.OptimismPortal2.proofSubmitters(
          withdrawalHash,
          i
        )

      // Grab the ProvenWithdrawal struct for this proof.
      const provenWithdrawal =
        await this.contracts.l1.OptimismPortal2.provenWithdrawals(
          withdrawalHash,
          proofSubmitter
        )

      // Grab the game that was proven against.
      const game = new ethers.Contract(
        provenWithdrawal.disputeGameProxy,
        getContractInterfaceBedrock('FaultDisputeGame'),
        this.l1SignerOrProvider
      )

      // Check the game status.
      const status = await game.status()
      if (status === 1) {
        // If status is CHALLENGER_WINS then it's no good.
        continue
      } else if (status === 2) {
        // If status is DEFENDER_WINS then it's a valid proof.
        return {
          ...provenWithdrawal,
          proofSubmitter,
        }
      } else if (status > 2) {
        // Shouldn't happen in practice.
        throw new Error('got invalid game status')
      }

      // Otherwise we're IN_PROGRESS.
      // Grab the block number from the extra data. Since this is not a standardized field we need
      // to be defensive and assume that the extra data could be anything. If the extra data does
      // not decode properly then we just skip this game.
      const extraData = await game.extraData()
      let l2BlockNumber: number
      try {
        ;[l2BlockNumber] = ethers.utils.defaultAbiCoder.decode(
          ['uint256'],
          extraData
        )
      } catch (err) {
        // Didn't decode properly, bad game.
        continue
      }

      // Finally we check if the output root is valid. If it is, then we can return the proven
      // withdrawal. If it isn't, then we act as if this proof does not exist because it isn't
      // useful for finalizing the withdrawal.
      if (await this.isValidOutputRoot(await game.rootClaim(), l2BlockNumber)) {
        return {
          ...provenWithdrawal,
          proofSubmitter,
        }
      }
    }

    // Return null if we didn't find a valid proof.
    return null
  }

  /**
   * Checks whether a given root claim is valid. Uses the L2 node that the SDK is connected to
   * when verifying the claim. Assumes that the connected L2 node is honest.
   *
   * @param outputRoot Output root to verify.
   * @param l2BlockNumber L2 block number the root is for.
   * @returns Whether or not the root is valid.
   */
  public async isValidOutputRoot(
    outputRoot: string,
    l2BlockNumber: number
  ): Promise<boolean> {
    // Use the cache if we can.
    const cached = this._outputCache.find((other) => {
      return other.root === outputRoot
    })

    // Skip if we can use the cached.
    if (cached) {
      return cached.valid
    }

    // If the cache ever gets to 10k elements, clear out the first half. Works well enough
    // since the cache will generally tend to be used in a FIFO manner.
    if (this._outputCache.length > 10000) {
      this._outputCache = this._outputCache.slice(5000)
    }

    // We didn't hit the cache so we're going to have to do the work.
    try {
      // Make sure this is a JSON RPC provider.
      const provider = toJsonRpcProvider(this.l2Provider)

      // Grab the block and storage proof at the same time.
      const [block, proof] = await Promise.all([
        provider.send('eth_getBlockByNumber', [
          toRpcHexString(l2BlockNumber),
          false,
        ]),
        makeStateTrieProof(
          provider,
          l2BlockNumber,
          this.contracts.l2.OVM_L2ToL1MessagePasser.address,
          ethers.constants.HashZero
        ),
      ])

      // Compute the output.
      const output = ethers.utils.solidityKeccak256(
        ['bytes32', 'bytes32', 'bytes32', 'bytes32'],
        [
          ethers.constants.HashZero,
          block.stateRoot,
          proof.storageRoot,
          block.hash,
        ]
      )

      // If the output matches the proposal then we're good.
      const valid = output === outputRoot
      this._outputCache.push({ root: outputRoot, valid })
      return valid
    } catch (err) {
      // Assume the game is invalid but don't add it to the cache just in case we had a temp error.
      return false
    }
  }

  /**
   * Returns the Bedrock output root that corresponds to the given message.
   *
   * @param message Message to get the Bedrock output root for.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Bedrock output root.
   */
  public async getMessageBedrockOutput(
    message: MessageLike,
    messageIndex = 0
  ): Promise<BedrockOutputData | null> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)

    // Outputs are only a thing for L2 to L1 messages.
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`cannot get a state root for an L1 to L2 message`)
    }

    let proposal: any
    let l2OutputIndex: BigNumber
    if (await this.fpac()) {
      // Get the respected game type from the portal.
      const gameType =
        await this.contracts.l1.OptimismPortal2.respectedGameType()

      // Get the total game count from the DisputeGameFactory since that will give us the end of
      // the array that we're searching over. We'll then use that to find the latest games.
      const gameCount = await this.contracts.l1.DisputeGameFactory.gameCount()

      // Find the latest 100 games (or as many as we can up to 100).
      const latestGames =
        await this.contracts.l1.DisputeGameFactory.findLatestGames(
          gameType,
          Math.max(0, gameCount.sub(1).toNumber()),
          Math.min(100, gameCount.toNumber())
        )

      // Find all games that are for proposals about blocks newer than the message block.
      const matches: any[] = []
      for (const game of latestGames) {
        try {
          const [blockNumber] = ethers.utils.defaultAbiCoder.decode(
            ['uint256'],
            game.extraData
          )
          if (blockNumber.gte(resolved.blockNumber)) {
            matches.push({
              ...game,
              l2BlockNumber: blockNumber,
            })
          }
        } catch (err) {
          // If we can't decode the extra data then we just skip this game.
          continue
        }
      }

      // Shuffle the list of matches. We shuffle here to avoid potential DoS vectors where the
      // latest games are all invalid and the SDK would be forced to make a bunch of archive calls.
      for (let i = matches.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        ;[matches[i], matches[j]] = [matches[j], matches[i]]
      }

      // Now we verify the proposals in the matches array.
      let match: any
      for (const option of matches) {
        if (
          await this.isValidOutputRoot(option.rootClaim, option.l2BlockNumber)
        ) {
          match = option
          break
        }
      }

      // If there's no match then we can't prove the message to the portal.
      if (!match) {
        return null
      }

      // Put the result into the same format as the old logic for now to reduce added code.
      l2OutputIndex = match.index
      proposal = {
        outputRoot: match.rootClaim,
        timestamp: match.timestamp,
        l2BlockNumber: match.l2BlockNumber,
      }
    } else {
      // Try to find the output index that corresponds to the block number attached to the message.
      // We'll explicitly handle "cannot get output" errors as a null return value, but anything else
      // needs to get thrown. Might need to revisit this in the future to be a little more robust
      // when connected to RPCs that don't return nice error messages.
      try {
        l2OutputIndex =
          await this.contracts.l1.L2OutputOracle.getL2OutputIndexAfter(
            resolved.blockNumber
          )
      } catch (err) {
        if (err.message.includes('L2OutputOracle: cannot get output')) {
          return null
        } else {
          throw err
        }
      }

      // Now pull the proposal out given the output index. Should always work as long as the above
      // codepath completed successfully.
      proposal = await this.contracts.l1.L2OutputOracle.getL2Output(
        l2OutputIndex
      )
    }

    // Format everything and return it nicely.
    return {
      outputRoot: proposal.outputRoot,
      l1Timestamp: proposal.timestamp.toNumber(),
      l2BlockNumber: proposal.l2BlockNumber.toNumber(),
      l2OutputIndex: l2OutputIndex.toNumber(),
    }
  }

  /**
   * Returns the state root that corresponds to a given message. This is the state root for the
   * block in which the transaction was included, as published to the StateCommitmentChain. If the
   * state root for the given message has not been published yet, this function returns null.
   *
   * @param message Message to find a state root for.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns State root for the block in which the message was created.
   */
  public async getMessageStateRoot(
    message: MessageLike,
    messageIndex = 0
  ): Promise<StateRoot | null> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)

    // State roots are only a thing for L2 to L1 messages.
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`cannot get a state root for an L1 to L2 message`)
    }

    // We need the block number of the transaction that triggered the message so we can look up the
    // state root batch that corresponds to that block number.
    const messageTxReceipt = await this.l2Provider.getTransactionReceipt(
      resolved.transactionHash
    )

    // Every block has exactly one transaction in it. Since there's a genesis block, the
    // transaction index will always be one less than the block number.
    const messageTxIndex = messageTxReceipt.blockNumber - 1

    // Pull down the state root batch, we'll try to pick out the specific state root that
    // corresponds to our message.
    const stateRootBatch = await this.getStateRootBatchByTransactionIndex(
      messageTxIndex
    )

    // No state root batch, no state root.
    if (stateRootBatch === null) {
      return null
    }

    // We have a state root batch, now we need to find the specific state root for our transaction.
    // First we need to figure out the index of the state root within the batch we found. This is
    // going to be the original transaction index offset by the total number of previous state
    // roots.
    const indexInBatch =
      messageTxIndex - stateRootBatch.header.prevTotalElements.toNumber()

    // Just a sanity check.
    if (stateRootBatch.stateRoots.length <= indexInBatch) {
      // Should never happen!
      throw new Error(`state root does not exist in batch`)
    }

    return {
      stateRoot: stateRootBatch.stateRoots[indexInBatch],
      stateRootIndexInBatch: indexInBatch,
      batch: stateRootBatch,
    }
  }

  /**
   * Returns the StateBatchAppended event that was emitted when the batch with a given index was
   * created. Returns null if no such event exists (the batch has not been submitted).
   *
   * @param batchIndex Index of the batch to find an event for.
   * @returns StateBatchAppended event for the batch, or null if no such batch exists.
   */
  public async getStateBatchAppendedEventByBatchIndex(
    batchIndex: number
  ): Promise<ethers.Event | null> {
    const events = await this.contracts.l1.StateCommitmentChain.queryFilter(
      this.contracts.l1.StateCommitmentChain.filters.StateBatchAppended(
        batchIndex
      )
    )

    if (events.length === 0) {
      return null
    } else if (events.length > 1) {
      // Should never happen!
      throw new Error(`found more than one StateBatchAppended event`)
    } else {
      return events[0]
    }
  }

  /**
   * Returns the StateBatchAppended event for the batch that includes the transaction with the
   * given index. Returns null if no such event exists.
   *
   * @param transactionIndex Index of the L2 transaction to find an event for.
   * @returns StateBatchAppended event for the batch that includes the given transaction by index.
   */
  public async getStateBatchAppendedEventByTransactionIndex(
    transactionIndex: number
  ): Promise<ethers.Event | null> {
    const isEventHi = (event: ethers.Event, index: number) => {
      const prevTotalElements = event.args._prevTotalElements.toNumber()
      return index < prevTotalElements
    }

    const isEventLo = (event: ethers.Event, index: number) => {
      const prevTotalElements = event.args._prevTotalElements.toNumber()
      const batchSize = event.args._batchSize.toNumber()
      return index >= prevTotalElements + batchSize
    }

    const totalBatches: ethers.BigNumber =
      await this.contracts.l1.StateCommitmentChain.getTotalBatches()
    if (totalBatches.eq(0)) {
      return null
    }

    let lowerBound = 0
    let upperBound = totalBatches.toNumber() - 1
    let batchEvent: ethers.Event | null =
      await this.getStateBatchAppendedEventByBatchIndex(upperBound)

    // Only happens when no batches have been submitted yet.
    if (batchEvent === null) {
      return null
    }

    if (isEventLo(batchEvent, transactionIndex)) {
      // Upper bound is too low, means this transaction doesn't have a corresponding state batch yet.
      return null
    } else if (!isEventHi(batchEvent, transactionIndex)) {
      // Upper bound is not too low and also not too high. This means the upper bound event is the
      // one we're looking for! Return it.
      return batchEvent
    }

    // Binary search to find the right event. The above checks will guarantee that the event does
    // exist and that we'll find it during this search.
    while (lowerBound < upperBound) {
      const middleOfBounds = Math.floor((lowerBound + upperBound) / 2)
      batchEvent = await this.getStateBatchAppendedEventByBatchIndex(
        middleOfBounds
      )

      if (isEventHi(batchEvent, transactionIndex)) {
        upperBound = middleOfBounds
      } else if (isEventLo(batchEvent, transactionIndex)) {
        lowerBound = middleOfBounds
      } else {
        break
      }
    }

    return batchEvent
  }

  /**
   * Returns information about the state root batch that included the state root for the given
   * transaction by index. Returns null if no such state root has been published yet.
   *
   * @param transactionIndex Index of the L2 transaction to find a state root batch for.
   * @returns State root batch for the given transaction index, or null if none exists yet.
   */
  public async getStateRootBatchByTransactionIndex(
    transactionIndex: number
  ): Promise<StateRootBatch | null> {
    const stateBatchAppendedEvent =
      await this.getStateBatchAppendedEventByTransactionIndex(transactionIndex)
    if (stateBatchAppendedEvent === null) {
      return null
    }

    const stateBatchTransaction = await stateBatchAppendedEvent.getTransaction()
    const [stateRoots] =
      this.contracts.l1.StateCommitmentChain.interface.decodeFunctionData(
        'appendStateBatch',
        stateBatchTransaction.data
      )

    return {
      blockNumber: stateBatchAppendedEvent.blockNumber,
      stateRoots,
      header: {
        batchIndex: stateBatchAppendedEvent.args._batchIndex,
        batchRoot: stateBatchAppendedEvent.args._batchRoot,
        batchSize: stateBatchAppendedEvent.args._batchSize,
        prevTotalElements: stateBatchAppendedEvent.args._prevTotalElements,
        extraData: stateBatchAppendedEvent.args._extraData,
      },
    }
  }

  /**
   * Generates the proof required to finalize an L2 to L1 message.
   *
   * @param message Message to generate a proof for.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Proof that can be used to finalize the message.
   */
  public async getMessageProof(
    message: MessageLike,
    messageIndex = 0
  ): Promise<CrossChainMessageProof> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`can only generate proofs for L2 to L1 messages`)
    }

    const stateRoot = await this.getMessageStateRoot(resolved, messageIndex)
    if (stateRoot === null) {
      throw new Error(`state root for message not yet published`)
    }

    // We need to calculate the specific storage slot that demonstrates that this message was
    // actually included in the L2 chain. The following calculation is based on the fact that
    // messages are stored in the following mapping on L2:
    // https://github.com/ethereum-optimism/optimism/blob/c84d3450225306abbb39b4e7d6d82424341df2be/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol#L23
    // You can read more about how Solidity storage slots are computed for mappings here:
    // https://docs.soliditylang.org/en/v0.8.4/internals/layout_in_storage.html#mappings-and-dynamic-arrays
    const messageSlot = ethers.utils.keccak256(
      ethers.utils.keccak256(
        encodeCrossDomainMessageV0(
          resolved.target,
          resolved.sender,
          resolved.message,
          resolved.messageNonce
        ) + remove0x(this.contracts.l2.L2CrossDomainMessenger.address)
      ) + '00'.repeat(32)
    )

    const stateTrieProof = await makeStateTrieProof(
      toJsonRpcProvider(this.l2Provider),
      resolved.blockNumber,
      this.contracts.l2.OVM_L2ToL1MessagePasser.address,
      messageSlot
    )

    return {
      stateRoot: stateRoot.stateRoot,
      stateRootBatchHeader: stateRoot.batch.header,
      stateRootProof: {
        index: stateRoot.stateRootIndexInBatch,
        siblings: makeMerkleTreeProof(
          stateRoot.batch.stateRoots,
          stateRoot.stateRootIndexInBatch
        ),
      },
      stateTrieWitness: toHexString(rlp.encode(stateTrieProof.accountProof)),
      storageTrieWitness: toHexString(rlp.encode(stateTrieProof.storageProof)),
    }
  }

  /**
   * Generates the bedrock proof required to finalize an L2 to L1 message.
   *
   * @param message Message to generate a proof for.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Proof that can be used to finalize the message.
   */
  public async getBedrockMessageProof(
    message: MessageLike,
    messageIndex = 0
  ): Promise<BedrockCrossChainMessageProof> {
    const resolved = await this.toCrossChainMessage(message, messageIndex)
    if (resolved.direction === MessageDirection.L1_TO_L2) {
      throw new Error(`can only generate proofs for L2 to L1 messages`)
    }

    const output = await this.getMessageBedrockOutput(resolved, messageIndex)
    if (output === null) {
      throw new Error(`state root for message not yet published`)
    }

    const withdrawal = await this.toLowLevelMessage(resolved, messageIndex)
    const hash = hashLowLevelMessage(withdrawal)
    const messageSlot = hashMessageHash(hash)

    const provider = toJsonRpcProvider(this.l2Provider)

    const stateTrieProof = await makeStateTrieProof(
      provider,
      output.l2BlockNumber,
      this.contracts.l2.BedrockMessagePasser.address,
      messageSlot
    )

    const block = await provider.send('eth_getBlockByNumber', [
      toRpcHexString(output.l2BlockNumber),
      false,
    ])

    return {
      outputRootProof: {
        version: ethers.constants.HashZero,
        stateRoot: block.stateRoot,
        messagePasserStorageRoot: stateTrieProof.storageRoot,
        latestBlockhash: block.hash,
      },
      withdrawalProof: stateTrieProof.storageProof,
      l2OutputIndex: output.l2OutputIndex,
    }
  }

  /**
   * Sends a given cross chain message. Where the message is sent depends on the direction attached
   * to the message itself.
   *
   * @param message Cross chain message to send.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message sending transaction.
   */
  public async sendMessage(
    message: CrossChainMessageRequest,
    opts?: {
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    const tx = await this.populateTransaction.sendMessage(message, opts)
    if (message.direction === MessageDirection.L1_TO_L2) {
      return (opts?.signer || this.l1Signer).sendTransaction(tx)
    } else {
      return (opts?.signer || this.l2Signer).sendTransaction(tx)
    }
  }

  /**
   * Resends a given cross chain message with a different gas limit. Only applies to L1 to L2
   * messages. If provided an L2 to L1 message, this function will throw an error.
   *
   * @param message Cross chain message to resend.
   * @param messageGasLimit New gas limit to use for the message.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the message resending transaction.
   */
  public async resendMessage(
    message: MessageLike,
    messageGasLimit: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.resendMessage(
        message,
        messageGasLimit,
        opts
      )
    )
  }

  /**
   * Proves a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
   * messages.
   *
   * @param message Message to finalize.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the finalization transaction.
   */
  public async proveMessage(
    message: MessageLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    },
    /**
     * The index of the withdrawal if multiple are made with multicall
     */
    messageIndex: number = 0
  ): Promise<TransactionResponse> {
    const tx = await this.populateTransaction.proveMessage(
      message,
      opts,
      messageIndex
    )
    return (opts?.signer || this.l1Signer).sendTransaction(tx)
  }

  /**
   * Finalizes a cross chain message that was sent from L2 to L1. Only applicable for L2 to L1
   * messages. Will throw an error if the message has not completed its challenge period yet.
   *
   * @param message Message to finalize.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @param messageIndex The index of the message, if multiple exist from multicall
   * @returns Transaction response for the finalization transaction.
   */
  public async finalizeMessage(
    message: MessageLike,
    opts?: {
      signer?: Signer
      overrides?: PayableOverrides
    },
    messageIndex = 0
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.finalizeMessage(
        message,
        opts,
        messageIndex
      )
    )
  }

  /**
   * Deposits some ETH into the L2 chain.
   *
   * @param amount Amount of ETH to deposit (in wei).
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  public async depositETH(
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.depositETH(amount, opts)
    )
  }

  /**
   * Withdraws some ETH back to the L1 chain.
   *
   * @param amount Amount of ETH to withdraw.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  public async withdrawETH(
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l2Signer).sendTransaction(
      await this.populateTransaction.withdrawETH(amount, opts)
    )
  }

  /**
   * Queries the account's approval amount for a given L1 token.
   *
   * @param l1Token The L1 token address.
   * @param l2Token The L2 token address.
   * @param opts Additional options.
   * @param opts.signer Optional signer to get the approval for.
   * @returns Amount of tokens approved for deposits from the account.
   */
  public async approval(
    l1Token: AddressLike,
    l2Token: AddressLike,
    opts?: {
      signer?: Signer
    }
  ): Promise<BigNumber> {
    const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
    return bridge.approval(l1Token, l2Token, opts?.signer || this.l1Signer)
  }

  /**
   * Approves a deposit into the L2 chain.
   *
   * @param l1Token The L1 token address.
   * @param l2Token The L2 token address.
   * @param amount Amount of the token to approve.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the approval transaction.
   */
  public async approveERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.approveERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  /**
   * Deposits some ERC20 tokens into the L2 chain.
   *
   * @param l1Token Address of the L1 token.
   * @param l2Token Address of the L2 token.
   * @param amount Amount to deposit.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
   * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the deposit transaction.
   */
  public async depositERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      l2GasLimit?: NumberLike
      overrides?: CallOverrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l1Signer).sendTransaction(
      await this.populateTransaction.depositERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  /**
   * Withdraws some ERC20 tokens back to the L1 chain.
   *
   * @param l1Token Address of the L1 token.
   * @param l2Token Address of the L2 token.
   * @param amount Amount to withdraw.
   * @param opts Additional options.
   * @param opts.signer Optional signer to use to send the transaction.
   * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
   * @param opts.overrides Optional transaction overrides.
   * @returns Transaction response for the withdraw transaction.
   */
  public async withdrawERC20(
    l1Token: AddressLike,
    l2Token: AddressLike,
    amount: NumberLike,
    opts?: {
      recipient?: AddressLike
      signer?: Signer
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return (opts?.signer || this.l2Signer).sendTransaction(
      await this.populateTransaction.withdrawERC20(
        l1Token,
        l2Token,
        amount,
        opts
      )
    )
  }

  /**
   * Object that holds the functions that generate transactions to be signed by the user.
   * Follows the pattern used by ethers.js.
   */
  populateTransaction = {
    /**
     * Generates a transaction that sends a given cross chain message. This transaction can be signed
     * and executed by a signer.
     *
     * @param message Cross chain message to send.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to send the message.
     */
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.contracts.l1.L1CrossDomainMessenger.populateTransaction.sendMessage(
          message.target,
          message.message,
          opts?.l2GasLimit || (await this.estimateL2MessageGasLimit(message)),
          opts?.overrides || {}
        )
      } else {
        return this.contracts.l2.L2CrossDomainMessenger.populateTransaction.sendMessage(
          message.target,
          message.message,
          0, // Gas limit goes unused when sending from L2 to L1
          opts?.overrides || {}
        )
      }
    },

    /**
     * Generates a transaction that resends a given cross chain message. Only applies to L1 to L2
     * messages. This transaction can be signed and executed by a signer.
     *
     * @param message Cross chain message to resend.
     * @param messageGasLimit New gas limit to use for the message.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to resend the message.
     */
    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      },
      /**
       * The index of the withdrawal if multiple are made with multicall
       */
      messageIndex = 0
    ): Promise<TransactionRequest> => {
      const resolved = await this.toCrossChainMessage(message, messageIndex)
      if (resolved.direction === MessageDirection.L2_TO_L1) {
        throw new Error(`cannot resend L2 to L1 message`)
      }

      if (this.bedrock) {
        return this.populateTransaction.finalizeMessage(
          resolved,
          {
            ...(opts || {}),
            overrides: {
              ...opts?.overrides,
              gasLimit: messageGasLimit,
            },
          },
          messageIndex
        )
      } else {
        const legacyL1XDM = new ethers.Contract(
          this.contracts.l1.L1CrossDomainMessenger.address,
          getContractInterface('L1CrossDomainMessenger'),
          this.l1SignerOrProvider
        )
        return legacyL1XDM.populateTransaction.replayMessage(
          resolved.target,
          resolved.sender,
          resolved.message,
          resolved.messageNonce,
          resolved.minGasLimit,
          messageGasLimit,
          opts?.overrides || {}
        )
      }
    },

    /**
     * Generates a message proving transaction that can be signed and executed. Only
     * applicable for L2 to L1 messages.
     *
     * @param message Message to generate the proving transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @param messageIndex The index of the message, if multiple exist from multicall
     * @returns Transaction that can be signed and executed to prove the message.
     */
    proveMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: PayableOverrides
      },
      messageIndex = 0
    ): Promise<TransactionRequest> => {
      const resolved = await this.toCrossChainMessage(message, messageIndex)
      if (resolved.direction === MessageDirection.L1_TO_L2) {
        throw new Error('cannot finalize L1 to L2 message')
      }

      if (!this.bedrock) {
        throw new Error(
          'message proving only applies after the bedrock upgrade'
        )
      }

      const withdrawal = await this.toLowLevelMessage(resolved, messageIndex)
      const proof = await this.getBedrockMessageProof(resolved, messageIndex)

      const args = [
        [
          withdrawal.messageNonce,
          withdrawal.sender,
          withdrawal.target,
          withdrawal.value,
          withdrawal.minGasLimit,
          withdrawal.message,
        ],
        proof.l2OutputIndex,
        [
          proof.outputRootProof.version,
          proof.outputRootProof.stateRoot,
          proof.outputRootProof.messagePasserStorageRoot,
          proof.outputRootProof.latestBlockhash,
        ],
        proof.withdrawalProof,
        opts?.overrides || {},
      ] as const

      return this.contracts.l1.OptimismPortal.populateTransaction.proveWithdrawalTransaction(
        ...args
      )
    },

    /**
     * Generates a message finalization transaction that can be signed and executed. Only
     * applicable for L2 to L1 messages. Will throw an error if the message has not completed
     * its challenge period yet.
     *
     * @param message Message to generate the finalization transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @param messageIndex The index of the message, if multiple exist from multicall
     * @returns Transaction that can be signed and executed to finalize the message.
     */
    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: PayableOverrides
      },
      messageIndex = 0
    ): Promise<TransactionRequest> => {
      const resolved = await this.toCrossChainMessage(message, messageIndex)
      if (resolved.direction === MessageDirection.L1_TO_L2) {
        throw new Error(`cannot finalize L1 to L2 message`)
      }

      if (this.bedrock) {
        // get everything we need to finalize
        const messageHashV1 = hashCrossDomainMessagev1(
          resolved.messageNonce,
          resolved.sender,
          resolved.target,
          resolved.value,
          resolved.minGasLimit,
          resolved.message
        )

        // fetch the following
        // 1. Whether it needs to be replayed because it failed
        // 2. The withdrawal as a low level message
        const [isFailed, withdrawal] = await Promise.allSettled([
          this.contracts.l1.L1CrossDomainMessenger.failedMessages(
            messageHashV1
          ),
          this.toLowLevelMessage(resolved, messageIndex),
        ])

        // handle errors
        if (
          isFailed.status === 'rejected' ||
          withdrawal.status === 'rejected'
        ) {
          const rejections = [isFailed, withdrawal]
            .filter((p) => p.status === 'rejected')
            .map((p: PromiseRejectedResult) => p.reason)
          throw rejections.length > 1
            ? new AggregateError(rejections)
            : rejections[0]
        }

        if (isFailed.value === true) {
          const xdmWithdrawal =
            this.contracts.l1.L1CrossDomainMessenger.interface.decodeFunctionData(
              'relayMessage',
              withdrawal.value.message
            )
          return this.contracts.l1.L1CrossDomainMessenger.populateTransaction.relayMessage(
            xdmWithdrawal._nonce,
            xdmWithdrawal._sender,
            xdmWithdrawal._target,
            xdmWithdrawal._value,
            xdmWithdrawal._minGasLimit,
            xdmWithdrawal._message,
            opts?.overrides || {}
          )
        }

        return this.contracts.l1.OptimismPortal.populateTransaction.finalizeWithdrawalTransaction(
          [
            withdrawal.value.messageNonce,
            withdrawal.value.sender,
            withdrawal.value.target,
            withdrawal.value.value,
            withdrawal.value.minGasLimit,
            withdrawal.value.message,
          ],
          opts?.overrides || {}
        )
      } else {
        // L1CrossDomainMessenger relayMessage is the only method that isn't fully backwards
        // compatible, so we need to use the legacy interface. When we fully upgrade to Bedrock we
        // should be able to remove this code.
        const proof = await this.getMessageProof(resolved, messageIndex)
        const legacyL1XDM = new ethers.Contract(
          this.contracts.l1.L1CrossDomainMessenger.address,
          getContractInterface('L1CrossDomainMessenger'),
          this.l1SignerOrProvider
        )
        return legacyL1XDM.populateTransaction.relayMessage(
          resolved.target,
          resolved.sender,
          resolved.message,
          resolved.messageNonce,
          proof,
          opts?.overrides || {}
        )
      }
    },

    /**
     * Generates a transaction for depositing some ETH into the L2 chain.
     *
     * @param amount Amount of ETH to deposit.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the ETH.
     */
    depositETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: PayableOverrides
      },
      isEstimatingGas: boolean = false
    ): Promise<TransactionRequest> => {
      const getOpts = async () => {
        if (isEstimatingGas) {
          return opts
        }
        const gasEstimation = await this.estimateGas.depositETH(amount, opts)
        return {
          ...opts,
          overrides: {
            ...opts?.overrides,
            gasLimit: gasEstimation.add(gasEstimation.div(2)),
          },
        }
      }
      return this.bridges.ETH.populateTransaction.deposit(
        ethers.constants.AddressZero,
        predeploys.OVM_ETH,
        amount,
        await getOpts()
      )
    },

    /**
     * Generates a transaction for withdrawing some ETH back to the L1 chain.
     *
     * @param amount Amount of ETH to withdraw.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the ETH.
     */
    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      return this.bridges.ETH.populateTransaction.withdraw(
        ethers.constants.AddressZero,
        predeploys.OVM_ETH,
        amount,
        opts
      )
    },

    /**
     * Generates a transaction for approving some tokens to deposit into the L2 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to approve.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction response for the approval transaction.
     */
    approveERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      return bridge.populateTransaction.approve(l1Token, l2Token, amount, opts)
    },

    /**
     * Generates a transaction for depositing some ERC20 tokens into the L2 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to deposit.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to deposit the tokens.
     */
    depositERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: CallOverrides
      },
      isEstimatingGas: boolean = false
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      // we need extra buffer for gas limit
      const getOpts = async () => {
        if (isEstimatingGas) {
          return opts
        }
        // if we don't include the users address the estimation will fail from lack of allowance
        if (!ethers.Signer.isSigner(this.l1SignerOrProvider)) {
          throw new Error('unable to deposit without an l1 signer')
        }
        const from = (this.l1SignerOrProvider as Signer).getAddress()
        const gasEstimation = await this.estimateGas.depositERC20(
          l1Token,
          l2Token,
          amount,
          {
            ...opts,
            overrides: {
              ...opts?.overrides,
              from: opts?.overrides?.from ?? from,
            },
          }
        )
        return {
          ...opts,
          overrides: {
            ...opts?.overrides,
            gasLimit: gasEstimation.add(gasEstimation.div(2)),
            from: opts?.overrides?.from ?? from,
          },
        }
      }
      return bridge.populateTransaction.deposit(
        l1Token,
        l2Token,
        amount,
        await getOpts()
      )
    },

    /**
     * Generates a transaction for withdrawing some ERC20 tokens back to the L1 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to withdraw.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction that can be signed and executed to withdraw the tokens.
     */
    withdrawERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const bridge = await this.getBridgeForTokenPair(l1Token, l2Token)
      return bridge.populateTransaction.withdraw(l1Token, l2Token, amount, opts)
    },
  }

  /**
   * Object that holds the functions that estimates the gas required for a given transaction.
   * Follows the pattern used by ethers.js.
   */
  estimateGas = {
    /**
     * Estimates gas required to send a cross chain message.
     *
     * @param message Cross chain message to send.
     * @param opts Additional options.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.sendMessage(message, opts)
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.l1Provider.estimateGas(tx)
      } else {
        return this.l2Provider.estimateGas(tx)
      }
    },

    /**
     * Estimates gas required to resend a cross chain message. Only applies to L1 to L2 messages.
     *
     * @param message Cross chain message to resend.
     * @param messageGasLimit New gas limit to use for the message.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.resendMessage(
          message,
          messageGasLimit,
          opts
        )
      )
    },

    /**
     * Estimates gas required to prove a cross chain message. Only applies to L2 to L1 messages.
     *
     * @param message Message to generate the proving transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @param messageIndex The index of the message, if multiple exist from multicall
     * @returns Gas estimate for the transaction.
     */
    proveMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: CallOverrides
      },
      messageIndex = 0
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.proveMessage(message, opts, messageIndex)
      )
    },

    /**
     * Estimates gas required to finalize a cross chain message. Only applies to L2 to L1 messages.
     *
     * @param message Message to generate the finalization transaction for.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @param messageIndex The index of the message, if multiple exist from multicall
     * @returns Gas estimate for the transaction.
     */
    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: CallOverrides
      },
      messageIndex = 0
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.finalizeMessage(
          message,
          opts,
          messageIndex
        )
      )
    },

    /**
     * Estimates gas required to deposit some ETH into the L2 chain.
     *
     * @param amount Amount of ETH to deposit.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    depositETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.depositETH(amount, opts, true)
      )
    },

    /**
     * Estimates gas required to withdraw some ETH back to the L1 chain.
     *
     * @param amount Amount of ETH to withdraw.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l2Provider.estimateGas(
        await this.populateTransaction.withdrawETH(amount, opts)
      )
    },

    /**
     * Estimates gas required to approve some tokens to deposit into the L2 chain.
     *
     * @param l1Token The L1 token address.
     * @param l2Token The L2 token address.
     * @param amount Amount of the token to approve.
     * @param opts Additional options.
     * @param opts.overrides Optional transaction overrides.
     * @returns Transaction response for the approval transaction.
     */
    approveERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.approveERC20(
          l1Token,
          l2Token,
          amount,
          opts
        )
      )
    },

    /**
     * Estimates gas required to deposit some ERC20 tokens into the L2 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to deposit.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L2. Defaults to sender.
     * @param opts.l2GasLimit Optional gas limit to use for the transaction on L2.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    depositERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l1Provider.estimateGas(
        await this.populateTransaction.depositERC20(
          l1Token,
          l2Token,
          amount,
          opts,
          true
        )
      )
    },

    /**
     * Estimates gas required to withdraw some ERC20 tokens back to the L1 chain.
     *
     * @param l1Token Address of the L1 token.
     * @param l2Token Address of the L2 token.
     * @param amount Amount to withdraw.
     * @param opts Additional options.
     * @param opts.recipient Optional address to receive the funds on L1. Defaults to sender.
     * @param opts.overrides Optional transaction overrides.
     * @returns Gas estimate for the transaction.
     */
    withdrawERC20: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: CallOverrides
      }
    ): Promise<BigNumber> => {
      return this.l2Provider.estimateGas(
        await this.populateTransaction.withdrawERC20(
          l1Token,
          l2Token,
          amount,
          opts
        )
      )
    },
  }
}
