import {
  BaseServiceV2,
  StandardOptions,
  ExpressRouter,
  Gauge,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import {
  getOEContract,
  DEFAULT_L2_CONTRACT_ADDRESSES,
  makeStateTrieProof,
  toJsonRpcProvider,
} from '@eth-optimism/sdk'
import { getChainId, sleep, toRpcHexString } from '@eth-optimism/core-utils'
import { Provider } from '@ethersproject/abstract-provider'
import { Contract, ethers } from 'ethers'
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
  highestCheckedBlockNumber: Gauge
  highestKnownBlockNumber: Gauge
  withdrawalsValidated: Gauge
  invalidProposalWithdrawals: Gauge
  invalidProofWithdrawals: Gauge
  isDetectingForgeries: Gauge
  nodeConnectionFailures: Gauge
}

type State = {
  portal: ethers.Contract
  messenger: ethers.Contract
  highestUncheckedBlockNumber: number
  faultProofWindow: number
  forgeryDetected: boolean
  invalidProposalWithdrawals: Array<Contract> //Withdrawals against invalid proposals.
  invalidProofWithdrawals: Array<Contract> //Withdrawals against invalid proof.
}

enum GameStatus {
  // The game is currently in progress, and has not been resolved.
  IN_PROGRESS,
  // The game has concluded, and the `rootClaim` was challenged successfully.
  CHALLENGER_WINS,
  // The game has concluded, and the `rootClaim` could not be contested.
  DEFENDER_WINS,
}

export class FaultProofWithdrawalMonitor extends BaseServiceV2<
  Options,
  Metrics,
  State
> {
  /**
   * Contract objects attached to their respective providers and addresses.
   */
  public l2ChainId: number

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
        highestCheckedBlockNumber: {
          type: Gauge,
          desc: 'Highest L1 block number that we have searched.',
          labels: ['type'],
        },
        highestKnownBlockNumber: {
          type: Gauge,
          desc: 'Highest L1 block number that we have seen.',
          labels: ['type'],
        },
        invalidProposalWithdrawals: {
          type: Gauge,
          desc: 'Number of withdrawals against invalid proposals.',
          labels: ['type'],
        },
        invalidProofWithdrawals: {
          type: Gauge,
          desc: 'Number of withdrawals with invalid proofs.',
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
    this.l2ChainId = l2ChainId

    // Create the OptimismPortal contract instance. If the optimismPortal option is not provided
    // then the SDK will attempt to resolve the address automatically based on the L2 chain ID. If
    // the SDK isn't aware of the L2 chain ID then it will throw an error that makes it clear the
    // user needs to provide this value explicitly.
    this.state.portal = getOEContract('OptimismPortal2', l2ChainId, {
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
    this.state.invalidProposalWithdrawals = []
    this.state.invalidProofWithdrawals = []
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
    this.metrics.isDetectingForgeries.set(Number(this.state.forgeryDetected))
    this.metrics.invalidProposalWithdrawals.set(
      this.state.invalidProposalWithdrawals.length
    )
    this.metrics.invalidProofWithdrawals.set(
      this.state.invalidProofWithdrawals.length
    )

    for (const disputeGame of this.state.invalidProposalWithdrawals) {
      const disputeGameAddress = disputeGame.address
      const isGameBlacklisted =
        this.state.portal.dispudeGameBlacklist(disputeGameAddress)
      if (isGameBlacklisted) {
        if (isGameBlacklisted) {
          const index =
            this.state.invalidProposalWithdrawals.indexOf(disputeGame)
          if (index !== -1) {
            this.state.invalidProposalWithdrawals.splice(index, 1)
          }
        }
      } else {
        const status = disputeGame.status()
        if (status === GameStatus.CHALLENGER_WINS) {
          const index =
            this.state.invalidProposalWithdrawals.indexOf(disputeGame)
          if (index !== -1) {
            this.state.invalidProposalWithdrawals.splice(index, 1)
          }
        } else if (status === GameStatus.DEFENDER_WINS) {
          this.state.forgeryDetected = true
          this.metrics.isDetectingForgeries.set(
            Number(this.state.forgeryDetected)
          )
        }
      }
    }
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
    this.metrics.highestKnownBlockNumber.set(latestL1BlockNumber)
    this.metrics.highestCheckedBlockNumber.set(
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

      const disputeGame = await this.getDisputeGameFromEvent(event)
      const rootClaim = await disputeGame.rootClaim()
      const l2BlockNumber = await disputeGame.l2BlockNumber()
      const isValidRoot = await this.isValidOutputRoot(rootClaim, l2BlockNumber)
      if (isValidRoot) {
        // Check if the withdrawal exists on L2.
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
          this.state.invalidProofWithdrawals.push(disputeGame)
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
          this.metrics.isDetectingForgeries.set(
            Number(this.state.forgeryDetected)
          )
        }
      } else {
        this.state.invalidProposalWithdrawals.push(disputeGame)
        this.logger.info(`invalid proposal`, {
          withdrawalHash: event.args.withdrawalHash,
        })
      }
    }

    // Increment the highest unchecked block number for the next loop.
    this.state.highestUncheckedBlockNumber = toBlockNumber
  }

  async getDisputeGameFromEvent(event: ethers.Event): Promise<ethers.Contract> {
    const disputeGameAddress = await this.getDisputeGameAddress(event)
    const disputeGame = await this.getDisputeGame(disputeGameAddress)
    return disputeGame
  }

  async getDisputeGameAddress(event: ethers.Event): Promise<string> {
    // Get the transaction informations from the event
    const transactionHash = event.transactionHash
    const tx = await this.options.l1RpcProvider.getTransaction(transactionHash)
    const sender = tx.from
    const withdrawalHash = event.args.withdrawalHash

    // Get the dispute game relative to this withdrawal from the portal
    const provenWithdrawals = await this.state.portal.provenWithdrawals(
      withdrawalHash,
      sender
    )
    const disputeGameProxyAddress = provenWithdrawals['disputeGameProxy']
    return disputeGameProxyAddress
  }

  async getDisputeGame(
    disputeGameProxyAddress: string
  ): Promise<ethers.Contract> {
    const FaultDisputeGame = getOEContract('FaultDisputeGame', this.l2ChainId, {
      signerOrProvider: this.options.l1RpcProvider,
      address: disputeGameProxyAddress,
    })

    return FaultDisputeGame
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
    try {
      // Make sure this is a JSON RPC provider.
      const provider = toJsonRpcProvider(this.options.l2RpcProvider)

      // Grab the block and storage proof at the same time.
      const [block, proof] = await Promise.all([
        provider.send('eth_getBlockByNumber', [
          toRpcHexString(l2BlockNumber),
          false,
        ]),
        makeStateTrieProof(
          provider,
          l2BlockNumber,
          this.state.messenger.address,
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
      return valid
    } catch (err) {
      // Assume the game is invalid but don't add it to the cache just in case we had a temp error.
      return false
    }
  }
}

if (require.main === module) {
  const service = new FaultProofWithdrawalMonitor()
  service.run()
}
