/* eslint-disable @typescript-eslint/no-unused-vars */
import { Overrides, Signer, BigNumber } from 'ethers'
import {
  TransactionRequest,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { predeploys } from '@eth-optimism/contracts'

import {
  CrossChainMessageRequest,
  ICrossChainMessenger,
  ICrossChainProvider,
  MessageLike,
  NumberLike,
  MessageDirection,
} from './interfaces'
import { omit } from './utils'

export class CrossChainMessenger implements ICrossChainMessenger {
  provider: ICrossChainProvider
  l1Signer: Signer
  l2Signer: Signer

  /**
   * Creates a new CrossChainMessenger instance.
   *
   * @param opts Options for the messenger.
   * @param opts.provider CrossChainProvider to use to send messages.
   * @param opts.l1Signer Signer to use to send messages on L1.
   * @param opts.l2Signer Signer to use to send messages on L2.
   */
  constructor(opts: {
    provider: ICrossChainProvider
    l1Signer: Signer
    l2Signer: Signer
  }) {
    this.provider = opts.provider
    this.l1Signer = opts.l1Signer
    this.l2Signer = opts.l2Signer
  }

  public async sendMessage(
    message: CrossChainMessageRequest,
    opts?: {
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    const tx = await this.populateTransaction.sendMessage(message, opts)
    if (message.direction === MessageDirection.L1_TO_L2) {
      return this.l1Signer.sendTransaction(tx)
    } else {
      return this.l2Signer.sendTransaction(tx)
    }
  }

  public async resendMessage(
    message: MessageLike,
    messageGasLimit: NumberLike,
    opts?: {
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return this.l1Signer.sendTransaction(
      await this.populateTransaction.resendMessage(
        message,
        messageGasLimit,
        opts
      )
    )
  }

  public async finalizeMessage(
    message: MessageLike,
    opts?: {
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    throw new Error('Not implemented')
  }

  public async depositETH(
    amount: NumberLike,
    opts?: {
      l2GasLimit?: NumberLike
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return this.l1Signer.sendTransaction(
      await this.populateTransaction.depositETH(amount, opts)
    )
  }

  public async withdrawETH(
    amount: NumberLike,
    opts?: {
      overrides?: Overrides
    }
  ): Promise<TransactionResponse> {
    return this.l2Signer.sendTransaction(
      await this.populateTransaction.withdrawETH(amount, opts)
    )
  }

  populateTransaction = {
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.provider.contracts.l1.L1CrossDomainMessenger.connect(
          this.l1Signer
        ).populateTransaction.sendMessage(
          message.target,
          message.message,
          opts?.l2GasLimit ||
            (await this.provider.estimateL2MessageGasLimit(message)),
          omit(opts?.overrides || {}, 'l2GasLimit')
        )
      } else {
        return this.provider.contracts.l2.L2CrossDomainMessenger.connect(
          this.l2Signer
        ).populateTransaction.sendMessage(
          message.target,
          message.message,
          0, // Gas limit goes unused when sending from L2 to L1
          omit(opts?.overrides || {}, 'l2GasLimit')
        )
      }
    },

    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      const resolved = await this.provider.toCrossChainMessage(message)
      if (resolved.direction === MessageDirection.L2_TO_L1) {
        throw new Error(`cannot resend L2 to L1 message`)
      }

      return this.provider.contracts.l1.L1CrossDomainMessenger.connect(
        this.l1Signer
      ).populateTransaction.replayMessage(
        resolved.target,
        resolved.sender,
        resolved.message,
        resolved.messageNonce,
        resolved.gasLimit,
        messageGasLimit,
        opts?.overrides || {}
      )
    },

    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      throw new Error('Not implemented')
    },

    depositETH: async (
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      return this.provider.contracts.l1.L1StandardBridge.populateTransaction.depositETH(
        opts?.l2GasLimit || 200000, // 200k gas is fine as a default
        '0x', // No data
        {
          ...omit(opts?.overrides || {}, 'l2GasLimit', 'value'),
          value: amount,
        }
      )
    },

    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      return this.provider.contracts.l2.L2StandardBridge.populateTransaction.withdraw(
        predeploys.OVM_ETH,
        amount,
        0, // No need to supply gas here
        '0x', // No data,
        opts?.overrides || {}
      )
    },
  }

  estimateGas = {
    sendMessage: async (
      message: CrossChainMessageRequest,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.sendMessage(message, opts)
      if (message.direction === MessageDirection.L1_TO_L2) {
        return this.provider.l1Provider.estimateGas(tx)
      } else {
        return this.provider.l2Provider.estimateGas(tx)
      }
    },

    resendMessage: async (
      message: MessageLike,
      messageGasLimit: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.resendMessage(
        message,
        messageGasLimit,
        opts
      )
      return this.provider.l1Provider.estimateGas(tx)
    },

    finalizeMessage: async (
      message: MessageLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      throw new Error('Not implemented')
    },

    depositETH: async (
      amount: NumberLike,
      opts?: {
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.depositETH(amount, opts)
      return this.provider.l1Provider.estimateGas(tx)
    },

    withdrawETH: async (
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<BigNumber> => {
      const tx = await this.populateTransaction.withdrawETH(amount, opts)
      return this.provider.l2Provider.estimateGas(tx)
    },
  }
}
