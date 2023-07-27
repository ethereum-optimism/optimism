/* eslint-disable @typescript-eslint/no-unused-vars */
import { ethers, Overrides, TransactionRequest, BlockTag, EventLog } from 'ethers'
import { predeploys } from '@eth-optimism/core-utils'
import { hexStringEquals } from '@eth-optimism/core-utils'

import {
  NumberLike,
  AddressLike,
  TokenBridgeMessage,
  MessageDirection,
} from '../interfaces'
import { toAddress, omit } from '../utils'
import { StandardBridgeAdapter } from './standard-bridge'

/**
 * Bridge adapter for the ETH bridge.
 */
export class ETHBridgeAdapter extends StandardBridgeAdapter {
  public async approval(
    l1Token: AddressLike,
    l2Token: AddressLike,
    signer: ethers.Signer
  ): Promise<BigInt> {
    throw new Error(`approval not necessary for ETH bridge`)
  }

  public async getDepositsByAddress(
    address: AddressLike,
    opts?: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]> {
    const events = await this.l1Bridge.queryFilter(
      this.l1Bridge.filters.ETHDepositInitiated(address),
      opts?.fromBlock,
      opts?.toBlock
    )

    return events
      .map((event: EventLog) => {
        return {
          direction: MessageDirection.L1_TO_L2,
          from: event.args.from,
          to: event.args.to,
          l1Token: ethers.ZeroAddress,
          l2Token: predeploys.LegacyERC20ETH,
          amount: event.args.amount,
          data: event.args.extraData,
          logIndex: event.index,
          blockNumber: event.blockNumber,
          transactionHash: event.transactionHash,
        }
      })
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  public async getWithdrawalsByAddress(
    address: AddressLike,
    opts?: {
      fromBlock?: BlockTag
      toBlock?: BlockTag
    }
  ): Promise<TokenBridgeMessage[]> {
    const events = await this.l2Bridge.queryFilter(
      this.l2Bridge.filters.WithdrawalInitiated(undefined, undefined, address),
      opts?.fromBlock,
      opts?.toBlock
    )

    return events
      .filter((event: EventLog) => {
        // Only find ETH withdrawals.
        return (
          hexStringEquals(event.args.l1Token, ethers.ZeroAddress) &&
          hexStringEquals(event.args.l2Token, predeploys.LegacyERC20ETH)
        )
      })
      .map((event: EventLog) => {
        return {
          direction: MessageDirection.L2_TO_L1,
          from: event.args.from,
          to: event.args.to,
          l1Token: event.args.l1Token,
          l2Token: event.args.l2Token,
          amount: event.args.amount,
          data: event.args.extraData,
          logIndex: event.index,
          blockNumber: event.blockNumber,
          transactionHash: event.transactionHash,
        }
      })
      .sort((a, b) => {
        // Sort descending by block number
        return b.blockNumber - a.blockNumber
      })
  }

  public async supportsTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<boolean> {
    // Only support ETH deposits and withdrawals.
    return (
      hexStringEquals(await toAddress(l1Token), ethers.ZeroAddress) &&
      hexStringEquals(await toAddress(l2Token), predeploys.LegacyERC20ETH)
    )
  }

  populateTransaction = {
    approve: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        overrides?: Overrides
      }
    ): Promise<never> => {
      throw new Error(`approvals not necessary for ETH bridge`)
    },

    deposit: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        l2GasLimit?: NumberLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      if (!(await this.supportsTokenPair(l1Token, l2Token))) {
        throw new Error(`token pair not supported by bridge`)
      }

      if (opts?.recipient === undefined) {
        return this.l1Bridge.depositETH.populateTransaction(
          opts?.l2GasLimit || 200_000, // Default to 200k gas limit.
          '0x', // No data.
          {
            ...omit(opts?.overrides || {}, 'value'),
            value: amount,
          }
        )
      } else {
        return this.l1Bridge.depositETHTo.populateTransaction(
          toAddress(opts.recipient),
          opts?.l2GasLimit || 200_000, // Default to 200k gas limit.
          '0x', // No data.
          {
            ...omit(opts?.overrides || {}, 'value'),
            value: amount,
          }
        )
      }
    },

    withdraw: async (
      l1Token: AddressLike,
      l2Token: AddressLike,
      amount: NumberLike,
      opts?: {
        recipient?: AddressLike
        overrides?: Overrides
      }
    ): Promise<TransactionRequest> => {
      if (!(await this.supportsTokenPair(l1Token, l2Token))) {
        throw new Error(`token pair not supported by bridge`)
      }

      if (opts?.recipient === undefined) {
        return this.l2Bridge.withdraw.populateTransaction(
          toAddress(l2Token),
          amount,
          0, // L1 gas not required.
          '0x', // No data.
          {
            ...omit(opts?.overrides || {}, 'value'),
            value: this.messenger.bedrock ? amount : 0,
          }
        )
      } else {
        return this.l2Bridge.withdrawTo.populateTransaction(
          toAddress(l2Token),
          toAddress(opts.recipient),
          amount,
          0, // L1 gas not required.
          '0x', // No data.
          {
            ...omit(opts?.overrides || {}, 'value'),
            value: this.messenger.bedrock ? amount : 0,
          }
        )
      }
    },
  }
}
