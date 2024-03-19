import {
  BaseServiceV2,
  StandardOptions,
  ExpressRouter,
  Gauge,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import { getOEContract, DEFAULT_L2_CONTRACT_ADDRESSES } from '@eth-optimism/sdk'
import { getChainId, sleep } from '@eth-optimism/core-utils'
import { Provider } from '@ethersproject/abstract-provider'
import { ethers } from 'ethers'
import dateformat from 'dateformat'

import { version } from '../../package.json'
import { DEFAULT_STARTING_BLOCK_NUMBERS } from './constants'

type Options = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  optimismPortalAddress: string
  l2ToL1MessagePasserAddress: string
  startBlockNumber: number
  eventBlockRange: number
  sleepTimeMs: number
}

type Metrics = {
  highestBlockNumber: Gauge
  withdrawalsValidated: Gauge
  isDetectingForgeries: Gauge
  nodeConnectionFailures: Gauge
}

type State = {
  portal: ethers.Contract
  messenger: ethers.Contract
  highestUncheckedBlockNumber: number
  faultProofWindow: number
  forgeryDetected: boolean
}

export class WithdrawalMonitor extends BaseServiceV2<Options, Metrics, State> {
  constructor(options?: Partial<Options & StandardOptions>) {
    super({
      version,
      name: 'two-step-monitor',
      loop: true,
      options: {
        loopIntervalMs: 1000,
        ...options,
      },
      optionsSpec: {
        l1RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L1',
        },
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L2',
        },
        optimismPortalAddress: {
          validator: validators.address,
          default: null,
          desc: 'Address of the OptimismPortal proxy contract on L1',
          public: true,
        },
        l2ToL1MessagePasserAddress: {
          validator: validators.address,
          default: DEFAULT_L2_CONTRACT_ADDRESSES.BedrockMessagePasser as string,
          desc: 'Address of the L2ToL1MessagePasser contract on L2',
          public: true,
        },
        startBlockNumber: {
          validator: validators.num,
          default: -1,
          desc: 'L1 block number to start checking from',
          public: true,
        },
        eventBlockRange: {
          validator: validators.num,
          default: 2000,
          desc: 'Number of blocks to query for events over per loop',
          public: true,
        },
        sleepTimeMs: {
          validator: validators.num,
          default: 15000,
          desc: 'Time in ms to sleep when waiting for a node',
          public: true,
        },
      },
      metricsSpec: {
        highestBlockNumber: {
          type: Gauge,
          desc: 'Highest block number (checked and known)',
          labels: ['type'],
        },
        withdrawalsValidated: {
          type: Gauge,
          desc: 'Latest L1 Block (checked and known)',
          labels: ['type'],
        },
        isDetectingForgeries: {
          type: Gauge,
          desc: '0 if state is ok. 1 or more if forged withdrawals are detected.',
        },
        nodeConnectionFailures: {
          type: Gauge,
          desc: 'Number of times node connection has failed',
          labels: ['layer', 'section'],
        },
      },
    })
  }

  async init(): Promise<void> {
    // Connect to L1.
    await waitForProvider(this.options.l1RpcProvider, {
      logger: this.logger,
      name: 'L1',
    })

    // Connect to L2.
    await waitForProvider(this.options.l2RpcProvider, {
      logger: this.logger,
      name: 'L2',
    })

    // Need L2 chain ID to resolve contract addresses.
    const l2ChainId = await getChainId(this.options.l2RpcProvider)

    // Create the OptimismPortal contract instance. If the optimismPortal option is not provided
    // then the SDK will attempt to resolve the address automatically based on the L2 chain ID. If
    // the SDK isn't aware of the L2 chain ID then it will throw an error that makes it clear the
    // user needs to provide this value explicitly.
    this.state.portal = getOEContract('OptimismPortal', l2ChainId, {
      signerOrProvider: this.options.l1RpcProvider,
      address: this.options.optimismPortalAddress,
    })

    // Create the L2ToL1MessagePasser contract instance. If the l2ToL1MessagePasser option is not
    // provided then we'll use the default address which typically should be correct. It's very
    // unlikely that any user would change this address so this should work in 99% of cases. If we
    // really wanted to be extra safe we could do some sanity checks to make sure the contract has
    // the interface we need but doesn't seem important for now.
    this.state.messenger = getOEContract('L2ToL1MessagePasser', l2ChainId, {
      signerOrProvider: this.options.l2RpcProvider,
      address: this.options.l2ToL1MessagePasserAddress,
    })

    // Previous versions of wd-mon would try to pick the starting block number automatically but
    // this had the possibility of missing certain withdrawals if the service was restarted at the
    // wrong time. Given the added complexity of finding a starting point automatically after FPAC,
    // it's much easier to simply start a fixed block number than trying to do something fancy. Use
    // the default configured in this service or use zero if no default is defined.
    this.state.highestUncheckedBlockNumber = this.options.startBlockNumber
    if (this.options.startBlockNumber === -1) {
      this.state.highestUncheckedBlockNumber =
        DEFAULT_STARTING_BLOCK_NUMBERS[l2ChainId] || 0
    }

    // Default state is that forgeries have not been detected.
    this.state.forgeryDetected = false
  }

  // K8s healthcheck
  async routes(router: ExpressRouter): Promise<void> {
    router.get('/healthz', async (req, res) => {
      return res.status(200).json({
        ok: !this.state.forgeryDetected,
      })
    })
  }

  async main(): Promise<void> {
    // Get the latest L1 block number.
    let latestL1BlockNumber: number
    try {
      latestL1BlockNumber = await this.options.l1RpcProvider.getBlockNumber()
    } catch (err) {
      // Log the issue so we can debug it.
      this.logger.error(`got error when connecting to node`, {
        error: err,
        node: 'l1',
        section: 'getBlockNumber',
      })

      // Increment the metric so we can detect the issue.
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'getBlockNumber',
      })

      // Sleep for a little to give intermittent errors a chance to recover.
      return sleep(this.options.sleepTimeMs)
    }

    // Update highest block number metrics so we can keep track of how the service is doing.
    this.metrics.highestBlockNumber.set({ type: 'known' }, latestL1BlockNumber)
    this.metrics.highestBlockNumber.set(
      { type: 'checked' },
      this.state.highestUncheckedBlockNumber
    )

    // Check if the RPC provider is behind us for some reason. Can happen occasionally,
    // particularly if connected to an RPC provider that load balances over multiple nodes that
    // might not be perfectly in sync.
    if (latestL1BlockNumber <= this.state.highestUncheckedBlockNumber) {
      // Sleep for a little to give the RPC a chance to catch up.
      return sleep(this.options.sleepTimeMs)
    }

    // Generally better to use a relatively small block range because it means this service can be
    // used alongside many different types of L1 nodes. For instance, Geth will typically only
    // support a block range of 2000 blocks out of the box.
    const toBlockNumber = Math.min(
      this.state.highestUncheckedBlockNumber + this.options.eventBlockRange,
      latestL1BlockNumber
    )

    // Useful to log this stuff just in case we get stuck or something.
    this.logger.info(`checking recent blocks`, {
      fromBlockNumber: this.state.highestUncheckedBlockNumber,
      toBlockNumber,
    })

    // Query for WithdrawalProven events within the specified block range.
    let events: ethers.Event[]
    try {
      events = await this.state.portal.queryFilter(
        this.state.portal.filters.WithdrawalProven(),
        this.state.highestUncheckedBlockNumber,
        toBlockNumber
      )
    } catch (err) {
      // Log the issue so we can debug it.
      this.logger.error(`got error when connecting to node`, {
        error: err,
        node: 'l1',
        section: 'querying for WithdrawalProven events',
      })

      // Increment the metric so we can detect the issue.
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'querying for WithdrawalProven events',
      })

      // Sleep for a little to give intermittent errors a chance to recover.
      return sleep(this.options.sleepTimeMs)
    }

    // Go over all the events and check if the withdrawal hash actually exists on L2.
    for (const event of events) {
      // Could consider using multicall here but this is efficient enough for now.
      const hash = event.args.withdrawalHash
      const exists = await this.state.messenger.sentMessages(hash)

      // Hopefully the withdrawal exists!
      if (exists) {
        // Unlike below we don't grab the timestamp here because it adds an unnecessary request.
        this.logger.info(`valid withdrawal`, {
          withdrawalHash: event.args.withdrawalHash,
        })

        // Bump the withdrawals metric so we can keep track.
        this.metrics.withdrawalsValidated.inc()
      } else {
        // Grab and format the timestamp so it's clear how much time is left.
        const block = await event.getBlock()
        const ts = `${dateformat(
          new Date(block.timestamp * 1000),
          'mmmm dS, yyyy, h:MM:ss TT',
          true
        )} UTC`

        // Uh oh!
        this.logger.error(`withdrawalHash not seen on L2`, {
          withdrawalHash: event.args.withdrawalHash,
          provenAt: ts,
        })

        // Change to forgery state.
        this.state.forgeryDetected = true
        this.metrics.isDetectingForgeries.set(1)

        // Return early so that we never increment the highest unchecked block number and therefore
        // will continue to loop on this forgery indefinitely. We probably want to change this
        // behavior at some point so that we keep scanning for additional forgeries since the
        // existence of one forgery likely implies the existence of many others.
        return sleep(this.options.sleepTimeMs)
      }
    }

    // Increment the highest unchecked block number for the next loop.
    this.state.highestUncheckedBlockNumber = toBlockNumber

    // If we got through the above without throwing an error, we should be fine to reset. Only case
    // where this is relevant is if something is detected as a forgery accidentally and the error
    // doesn't happen again on the next loop.
    this.state.forgeryDetected = false
    this.metrics.isDetectingForgeries.set(0)
  }
}

if (require.main === module) {
  const service = new WithdrawalMonitor()
  service.run()
}
