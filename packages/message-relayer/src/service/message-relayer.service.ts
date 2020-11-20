/* Imports: External */
import { Contract, ethers, Wallet, BigNumber } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { getContractInterface } from '@eth-optimism/contracts'
import * as rlp from 'rlp'

/* Imports: Internal */
import { BaseService } from './base.service'
import { sleep } from '../utils/common'
import { StateBatchHeader, SentMessage, MessageProof } from '../types/ovm.types'

interface MessageRelayerOptions {
  // Providers.
  l1RpcProvider: JsonRpcProvider
  l2RpcProvider: JsonRpcProvider

  // Contract addresses.
  stateCommitmentChainAddress: string
  l1CrossDomainMessengerAddress: string
  l2CrossDomainMessengerAddress: string
  l2ToL1MessagePasserAddress: string

  // Wallet.
  relaySigner: Wallet

  // Optionals.
  l2ChainStartingHeight?: number
  pollingInterval?: number
  blockOffset?: number
}

export class MessageRelayerService extends BaseService<MessageRelayerOptions> {
  private stateCommitmentChain: Contract
  private l1CrossDomainMessenger: Contract
  private l2CrossDomainMessenger: Contract
  private l2ToL1MessagePasser: Contract
  private pollingInterval: number
  private lastFinalizedTxHeight: number
  private nextUnfinalizedTxHeight: number
  private blockOffset: number

  protected async _init(): Promise<void> {
    this.stateCommitmentChain = new Contract(
      this.options.stateCommitmentChainAddress,
      getContractInterface('OVM_StateCommitmentChain'),
      this.options.l1RpcProvider
    )

    this.l1CrossDomainMessenger = new Contract(
      this.options.l1CrossDomainMessengerAddress,
      getContractInterface('OVM_L1CrossDomainMessenger'),
      this.options.l1RpcProvider
    )

    this.l2CrossDomainMessenger = new Contract(
      this.options.l2CrossDomainMessengerAddress,
      getContractInterface('OVM_L2CrossDomainMessenger'),
      this.options.l2RpcProvider
    )

    this.l2ToL1MessagePasser = new Contract(
      this.options.l2ToL1MessagePasserAddress,
      getContractInterface('OVM_L2ToL1MessagePasser'),
      this.options.l2RpcProvider
    )

    this.pollingInterval = this.options.pollingInterval || 5000
    this.lastFinalizedTxHeight = this.options.l2ChainStartingHeight || 0
    this.nextUnfinalizedTxHeight = this.options.l2ChainStartingHeight || 0
    this.blockOffset = this.options.blockOffset || 0
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      await sleep(this.pollingInterval)

      if (!(await this._isTransactionFinalized(this.nextUnfinalizedTxHeight))) {
        continue
      }

      this.lastFinalizedTxHeight = this.nextUnfinalizedTxHeight
      while (await this._isTransactionFinalized(this.nextUnfinalizedTxHeight)) {
        this.nextUnfinalizedTxHeight += 1
      }

      const messages = await this._getSentMessages(
        this.lastFinalizedTxHeight,
        this.nextUnfinalizedTxHeight
      )

      for (const message of messages) {
        if (await this._wasMessageRelayed(message)) {
          continue
        }

        const proof = await this._getMessageProof(message)
        await this._relayMessageToL1(message, proof)
      }
    }
  }

  private async _getStateBatchHeader(
    height: number
  ): Promise< any | undefined> {

    const filter = this.stateCommitmentChain.filters.StateBatchAppended()
    const events = await this.stateCommitmentChain.queryFilter(
      filter,
      //height + this.blockOffset, // ?
    )
    var event
    for (event of events) {  // need the blockOffset for heights?
      if (event.args.prevTotalElements < height && event.args.prevTotalElements + event.args.batchSize >= height) {
        break
      }
    }

    const transaction = _getTransaction(event.args.transactionHash) // dummy
    const stateRoots = _getStateRoots(transaction.callData) // dummy

    return {
      batchIndex: event._batchIndex,
      batchRoot: event._batchRoot,
      batchSize: event._batchSize,
      prevTotalElements: event._prevTotalElements,
      extraData: event._extraData,
      stateRoots: stateRoots
    }

  }

  private async _isTransactionFinalized(height: number): Promise<boolean> {
    const batch = await this._getStateBatchHeader(height)

    if (batch === undefined) {
      return false
    }

    return !(await this.stateCommitmentChain.insideFraudProofWindow(batch))
  }

  private async _getSentMessages(
    startHeight: number,
    endHeight: number
  ): Promise<SentMessage[]> {
    const filter = this.l2CrossDomainMessenger.filters.SentMessage()
    const events = await this.l2CrossDomainMessenger.queryFilter(
      filter,
      startHeight + this.blockOffset,
      endHeight + this.blockOffset
    )

    return events.map((event) => {
      const message = event.args.message
      const decoded = this.l2CrossDomainMessenger.interface.decodeFunctionData(
        'relayMessage',
        message
      )

      return {
        target: decoded._target,
        sender: decoded._sender,
        data: decoded._message,
        nonce: decoded._messageNonce,
        calldata: message,
        hash: ethers.utils.keccak256(message),
        height: event.blockNumber - this.blockOffset,
      }
    })
  }

  private async _wasMessageRelayed(message: SentMessage): Promise<boolean> {
    return this.l1CrossDomainMessenger.successfulMessages(message.hash)
  }

  private async _getMessageProof(message: SentMessage): Promise<MessageProof> {
    const messageSlot = ethers.utils.keccak256(
      ethers.utils.keccak256(
        message.calldata + this.l2CrossDomainMessenger.address.slice(2)
      ) + '00'.repeat(32)
    )

    // TODO: Complain if the proof doesn't exist.
    const proof = await this.options.l2RpcProvider.send('eth_getProof', [
      this.l2ToL1MessagePasser.address,
      [messageSlot],
    ])

    // TODO: Complain if the batch doesn't exist.
    const batch = await this._getStateBatchHeader(message.height)

    return {
      stateRoot: proof.stateRoot,
      stateRootBatchHeader: batch,
      stateRootProof: {
        index: 0,
        siblings: [],
      },
      stateTrieWitness: rlp.encode(proof.accountProof),
      storageTrieWitness: rlp.encode(proof.storageProof[0].proof),
    }
  }

  private async _relayMessageToL1(
    message: SentMessage,
    proof: MessageProof
  ): Promise<void> {
    const transaction = await this.l1CrossDomainMessenger.populateTransaction.relayMessage(
      message.target,
      message.sender,
      message.data,
      message.nonce,
      proof
    )

    // TODO: Figure out how to set these.
    transaction.gasLimit = BigNumber.from(1000000)
    transaction.gasPrice = BigNumber.from(0)

    const signed = await this.options.relaySigner.signTransaction(transaction)

    await this.options.l1RpcProvider.sendTransaction(signed)
  }
}
