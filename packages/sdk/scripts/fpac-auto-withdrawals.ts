import {
  BaseServiceV2,
  StandardOptions,
  Counter,
  validators,
  Gauge,
} from '@eth-optimism/common-ts'
import { ethers } from 'ethers'
import { Provider } from '@ethersproject/abstract-provider'
import { getChainId } from '@eth-optimism/core-utils'

import { CrossChainMessenger, MessageStatus, TokenBridgeMessage } from '../src'

type WithdrawerOptions = {
  startingBlockNumber: number
  addrOptimismPortal: string
  addrDisputeGameFactory: string
  addrL1CrossDomainMessenger: string
  addrL1StandardBridge: string
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  key: string
}

type WithdrawerMetrics = {
  withdrawalsActive: Gauge
  withdrawalsCreated: Counter
  withdrawalsProven: Counter
  withdrawalsFinalized: Counter
  withdrawalsReproven: Counter
  highestSyncedBlock: Gauge
  highestKnownBlock: Gauge
}

type WithdrawerState = {
  messenger: CrossChainMessenger
  withdrawals: TokenBridgeMessage[]
  finalized: TokenBridgeMessage[]
  highestSyncedBlock: number
}

export class WithdrawerService extends BaseServiceV2<
  WithdrawerOptions,
  WithdrawerMetrics,
  WithdrawerState
> {
  constructor(options?: Partial<WithdrawerOptions & StandardOptions>) {
    super({
      version: '0.0.1',
      name: 'fpac-auto-withdrawals',
      loop: true,
      options: {
        loopIntervalMs: 60_000,
        ...options,
      },
      optionsSpec: {
        startingBlockNumber: {
          validator: validators.num,
          desc: 'Block number to start syncing from',
          default: 0,
          public: true,
        },
        addrOptimismPortal: {
          validator: validators.str,
          desc: 'Address of the OptimismPortal proxy contract',
          public: true,
        },
        addrDisputeGameFactory: {
          validator: validators.str,
          desc: 'Address of the DisputeGameFactory proxy contract',
          public: true,
        },
        addrL1CrossDomainMessenger: {
          validator: validators.str,
          desc: 'Address of the L1CrossDomainMessenger proxy contract',
          public: true,
        },
        addrL1StandardBridge: {
          validator: validators.str,
          desc: 'Address of the L1StandardBridge proxy contract',
          public: true,
        },
        l1RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for L1 network to connect to',
        },
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for L2 network to connect to',
        },
        key: {
          validator: validators.str,
          desc: 'Private key to use for signing transactions',
        },
      },
      metricsSpec: {
        withdrawalsActive: {
          type: Gauge,
          desc: 'Number of active withdrawals',
        },
        withdrawalsCreated: {
          type: Counter,
          desc: 'Number of withdrawals created',
        },
        withdrawalsProven: {
          type: Counter,
          desc: 'Number of withdrawals proven',
        },
        withdrawalsFinalized: {
          type: Counter,
          desc: 'Number of withdrawals finalized',
        },
        withdrawalsReproven: {
          type: Counter,
          desc: 'Number of withdrawals reproven',
        },
        highestSyncedBlock: {
          type: Gauge,
          desc: 'Highest block number synced',
        },
        highestKnownBlock: {
          type: Gauge,
          desc: 'Highest block number known',
        },
      },
    })
  }

  protected async init(): Promise<void> {
    this.state.highestSyncedBlock = this.options.startingBlockNumber
    this.state.withdrawals = []
    this.state.finalized = []
    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: new ethers.Wallet(
        this.options.key,
        this.options.l1RpcProvider
      ),
      l2SignerOrProvider: new ethers.Wallet(
        this.options.key,
        this.options.l2RpcProvider
      ),
      l1ChainId: await getChainId(this.options.l1RpcProvider),
      l2ChainId: await getChainId(this.options.l2RpcProvider),
      contracts: {
        l1: {
          OptimismPortal: this.options.addrOptimismPortal,
          OptimismPortal2: this.options.addrOptimismPortal,
          DisputeGameFactory: this.options.addrDisputeGameFactory,
          L1CrossDomainMessenger: this.options.addrL1CrossDomainMessenger,
          L1StandardBridge: this.options.addrL1StandardBridge,

          // Rest need to be filled out but can be empty.
          AddressManager: ethers.constants.AddressZero,
          StateCommitmentChain: ethers.constants.AddressZero,
          CanonicalTransactionChain: ethers.constants.AddressZero,
          BondManager: ethers.constants.AddressZero,
          L2OutputOracle: ethers.constants.AddressZero,
        },
      },
    })
  }

  protected async main(): Promise<void> {
    // Update highest known block.
    const latestBlockNumber = await this.options.l2RpcProvider.getBlockNumber()
    this.metrics.highestKnownBlock.set(latestBlockNumber)

    // Sync withdrawals to tip.
    while (this.state.highestSyncedBlock < latestBlockNumber) {
      // Sync withdrawals in chunks of 2000 blocks.
      const targetBlock = Math.min(
        this.state.highestSyncedBlock + 2000,
        latestBlockNumber
      )
      this.logger.info('Syncing block range', {
        fromBlock: this.state.highestSyncedBlock,
        toBlock: targetBlock,
        target: latestBlockNumber,
      })

      // Grab all withdrawals for the block range.
      let withdrawals = await this.state.messenger.getWithdrawalsByAddress(
        await this.state.messenger.l2Signer.getAddress(),
        {
          fromBlock: this.state.highestSyncedBlock,
          toBlock: targetBlock,
        }
      )

      // Remove any duplicates for overlapping block ranges.
      withdrawals = withdrawals.filter((withdrawal) => {
        return !this.state.withdrawals.some((other) => {
          return other.transactionHash === withdrawal.transactionHash
        })
      })

      // Insert the withdrawals.
      this.state.withdrawals.push(...withdrawals)
      this.logger.info('Block range synced', { count: withdrawals.length })

      // Update the highest synced block.
      this.state.highestSyncedBlock = targetBlock
      this.metrics.highestSyncedBlock.set(this.state.highestSyncedBlock)
    }

    // Create a withdrawal.
    this.logger.info('Creating a new withdrawal')
    const wd = await this.state.messenger.withdrawETH(1)
    this.logger.info('Withdrawal created', { hash: wd.hash })
    this.metrics.withdrawalsCreated.inc()

    // Check if any withdrawals are ready for relay.
    for (const withdrawal of this.state.withdrawals) {
      // Don't process finalized withdrawals.
      if (this.state.finalized.includes(withdrawal)) {
        continue
      }

      try {
        // Grab the status of the withdrawal.
        const status = await this.state.messenger.getMessageStatus(withdrawal)

        // If the withdrawal is ready to prove, prove it.
        if (status === MessageStatus.READY_TO_PROVE) {
          this.logger.info('Proving a withdrawal', {
            hash: withdrawal.transactionHash,
          })

          const receipt = await this.state.messenger.proveMessage(withdrawal)
          await this.state.messenger.waitForMessageStatus(
            withdrawal,
            MessageStatus.IN_CHALLENGE_PERIOD,
            { timeoutMs: 120_000 }
          )

          this.logger.info('Withdrawal proven', { hash: receipt.hash })
          this.metrics.withdrawalsProven.inc()
        }

        // If the withdrawal is ready for relay, finalize it.
        if (status === MessageStatus.READY_FOR_RELAY) {
          this.logger.info('Finalizing a withdrawal', {
            hash: withdrawal.transactionHash,
          })

          const receipt = await this.state.messenger.finalizeMessage(withdrawal)
          await this.state.messenger.waitForMessageStatus(
            withdrawal,
            MessageStatus.RELAYED,
            { timeoutMs: 120_000 }
          )

          this.logger.info('Withdrawal finalized', { hash: receipt.hash })
          this.state.finalized.push(withdrawal)
          this.metrics.withdrawalsFinalized.inc()
        }

        // If the withdrawal was already relayed, remove it from the list.
        if (status === MessageStatus.RELAYED) {
          this.logger.info('Withdrawal already relayed', {
            hash: withdrawal.transactionHash,
          })

          this.state.finalized.push(withdrawal)
          this.metrics.withdrawalsFinalized.inc()
        }
      } catch (err) {
        // If the withdrawal was invalidated, reprove it.
        if (
          err.message.includes(
            'withdrawal proposal was invalidated, must reprove'
          )
        ) {
          this.logger.info('Reproving a withdrawal', {
            hash: withdrawal.transactionHash,
          })

          const receipt = await this.state.messenger.proveMessage(withdrawal)
          this.logger.info('Withdrawal reproven', { hash: receipt.hash })
          this.metrics.withdrawalsReproven.inc()
        } else {
          throw err
        }
      }
    }

    // Update active withdrawal count.
    this.metrics.withdrawalsActive.set(this.state.withdrawals.length)
  }
}

if (require.main === module) {
  const service = new WithdrawerService()
  service.run()
}
