import {
  BaseServiceV2,
  StandardOptions,
  ExpressRouter,
  Gauge,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import { getChainId, sleep, toRpcHexString } from '@eth-optimism/core-utils'
import { config } from 'dotenv'
import {
  CONTRACT_ADDRESSES,
  CrossChainMessenger,
  getOEContract,
  L2ChainID,
  OEL1ContractsLike,
} from '@eth-optimism/sdk'
import { Provider } from '@ethersproject/abstract-provider'
import { ethers, Transaction } from 'ethers'
import dateformat from 'dateformat'

import { version } from '../package.json'
import {
  findFirstUnfinalizedStateBatchIndex,
  findEventForStateBatch,
  PartialEvent,
  OutputOracle,
  updateOracleCache,
} from './helpers'

type Options = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  startBatchIndex: number
  bedrock: boolean
  optimismPortalAddress?: string
  stateCommitmentChainAddress?: string
}

type Metrics = {
  highestBatchIndex: Gauge
  isCurrentlyMismatched: Gauge
  nodeConnectionFailures: Gauge
}

type State = {
  fpw: number
  oo: OutputOracle<any>
  messenger: CrossChainMessenger
  currentBatchIndex: number
  diverged: boolean
}

export class FaultDetector extends BaseServiceV2<Options, Metrics, State> {
  constructor(options?: Partial<Options & StandardOptions>) {
    super({
      version,
      name: 'fault-detector',
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
        startBatchIndex: {
          validator: validators.num,
          default: -1,
          desc: 'Batch index to start checking from. For bedrock chains, this is the L2 height to start from',
          public: true,
        },
        bedrock: {
          validator: validators.bool,
          default: true,
          desc: 'Whether or not the service is running against a Bedrock chain',
          public: true,
        },
        optimismPortalAddress: {
          validator: validators.str,
          default: ethers.constants.AddressZero,
          desc: '[Custom Bedrock Chains] Deployed OptimismPortal contract address. Used to retrieve necessary info for ouput verification ',
          public: true,
        },
        stateCommitmentChainAddress: {
          validator: validators.str,
          default: ethers.constants.AddressZero,
          desc: '[Custom Legacy Chains] Deployed StateCommitmentChain contract address. Used to fetch necessary info for output verification.',
          public: true,
        },
      },
      metricsSpec: {
        highestBatchIndex: {
          type: Gauge,
          desc: 'Highest batch indices (checked and known)',
          labels: ['type'],
        },
        isCurrentlyMismatched: {
          type: Gauge,
          desc: '0 if state is ok, 1 if state is mismatched',
        },
        nodeConnectionFailures: {
          type: Gauge,
          desc: 'Number of times node connection has failed',
          labels: ['layer', 'section'],
        },
      },
    })
  }

  /**
   * Provides the required set of addresses used by the fault detector. For recognized op-chains, this
   * will fallback to the pre-defined set of addresses from options, otherwise aborting if unset.
   *
   * Required Contracts
   * - Bedrock: OptimismPortal (used to also fetch L2OutputOracle address variable). This is the preferred address
   * since in early versions of bedrock, OptimismPortal holds the FINALIZATION_WINDOW variable instead of L2OutputOracle.
   * The retrieved L2OutputOracle address from OptimismPortal is used to query for output roots.
   * - Legacy: StateCommitmentChain to query for output roots.
   *
   * @param l2ChainId op chain id
   * @returns OEL1ContractsLike set of L1 contracts with only the required addresses set
   */
  async getOEL1Contracts(l2ChainId: number): Promise<OEL1ContractsLike> {
    // CrossChainMessenger requires all address to be defined. Default to `AddressZero` to ignore unused contracts
    let contracts: OEL1ContractsLike = {
      AddressManager: ethers.constants.AddressZero,
      L1CrossDomainMessenger: ethers.constants.AddressZero,
      L1StandardBridge: ethers.constants.AddressZero,
      StateCommitmentChain: ethers.constants.AddressZero,
      CanonicalTransactionChain: ethers.constants.AddressZero,
      BondManager: ethers.constants.AddressZero,
      OptimismPortal: ethers.constants.AddressZero,
      L2OutputOracle: ethers.constants.AddressZero,
    }

    const chainType = this.options.bedrock ? 'bedrock' : 'legacy'
    this.logger.info(`Setting contracts for OP chain type: ${chainType}`)

    const knownChainId = L2ChainID[l2ChainId] !== undefined
    if (knownChainId) {
      this.logger.info(`Recognized L2 chain id ${L2ChainID[l2ChainId]}`)

      // fallback to the predefined defaults for this chain id
      contracts = CONTRACT_ADDRESSES[l2ChainId].l1
    }

    this.logger.info('checking contract address options...')
    if (this.options.bedrock) {
      const address = this.options.optimismPortalAddress
      if (!knownChainId && address === ethers.constants.AddressZero) {
        this.logger.error('OptimismPortal contract unspecified')
        throw new Error(
          '--optimismportalcontractaddress needs to set for custom bedrock op chains'
        )
      }

      if (address !== ethers.constants.AddressZero) {
        this.logger.info('set OptimismPortal contract override')
        contracts.OptimismPortal = address

        this.logger.info('fetching L2OutputOracle contract from OptimismPortal')
        const opts = { address, signerOrProvider: this.options.l1RpcProvider }
        const portalContract = getOEContract('OptimismPortal', l2ChainId, opts)
        contracts.L2OutputOracle = await portalContract.L2_ORACLE()
      }

      // ... for a known chain ids without an override, the L2OutputOracle will already
      // be set via the hardcoded default
    } else {
      const address = this.options.stateCommitmentChainAddress
      if (!knownChainId && address === ethers.constants.AddressZero) {
        this.logger.error('StateCommitmentChain contract unspecified')
        throw new Error(
          '--statecommitmentchainaddress needs to set for custom legacy op chains'
        )
      }

      if (address !== ethers.constants.AddressZero) {
        this.logger.info('set StateCommitmentChain contract override')
        contracts.StateCommitmentChain = address
      }
    }

    return contracts
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

    const l1ChainId = await getChainId(this.options.l1RpcProvider)
    const l2ChainId = await getChainId(this.options.l2RpcProvider)
    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: this.options.l1RpcProvider,
      l2SignerOrProvider: this.options.l2RpcProvider,
      l1ChainId,
      l2ChainId,
      bedrock: this.options.bedrock,
      contracts: { l1: await this.getOEL1Contracts(l2ChainId) },
    })

    // Not diverged by default.
    this.state.diverged = false

    // We use this a lot, a bit cleaner to pull out to the top level of the state object.
    this.state.fpw = await this.state.messenger.getChallengePeriodSeconds()
    this.logger.info(`fault proof window is ${this.state.fpw} seconds`)

    if (this.options.bedrock) {
      const oo = this.state.messenger.contracts.l1.L2OutputOracle
      this.state.oo = {
        contract: oo,
        filter: oo.filters.OutputProposed(),
        getTotalElements: async () => oo.nextOutputIndex(),
        getEventIndex: (args) => args.l2OutputIndex,
      }
    } else {
      const oo = this.state.messenger.contracts.l1.StateCommitmentChain
      this.state.oo = {
        contract: oo,
        filter: oo.filters.StateBatchAppended(),
        getTotalElements: async () => oo.getTotalBatches(),
        getEventIndex: (args) => args._batchIndex,
      }
    }

    // Populate the event cache.
    this.logger.info('warming event cache, this might take a while...')
    await updateOracleCache(this.state.oo, this.logger)

    // Figure out where to start syncing from.
    if (this.options.startBatchIndex === -1) {
      this.logger.info('finding appropriate starting unfinalized batch')
      const firstUnfinalized = await findFirstUnfinalizedStateBatchIndex(
        this.state.oo,
        this.state.fpw,
        this.logger
      )

      // We may not have an unfinalized batches in the case where no batches have been submitted
      // for the entire duration of the FPW. We generally do not expect this to happen on mainnet,
      // but it happens often on testnets because the FPW is very short.
      if (firstUnfinalized === undefined) {
        this.logger.info('no unfinalized batches found. skipping all batches.')
        const totalBatches = await this.state.oo.getTotalElements()
        this.state.currentBatchIndex = totalBatches.toNumber() - 1
      } else {
        this.state.currentBatchIndex = firstUnfinalized
      }
    } else {
      this.state.currentBatchIndex = this.options.startBatchIndex
    }

    this.logger.info('starting batch', {
      batchIndex: this.state.currentBatchIndex,
    })

    // Set the initial metrics.
    this.metrics.isCurrentlyMismatched.set(0)
  }

  async routes(router: ExpressRouter): Promise<void> {
    router.get('/status', async (req, res) => {
      return res.status(200).json({
        ok: !this.state.diverged,
      })
    })
  }

  async main(): Promise<void> {
    const startMs = Date.now()

    let latestBatchIndex: number
    try {
      const totalBatches = await this.state.oo.getTotalElements()
      latestBatchIndex = totalBatches.toNumber() - 1
    } catch (err) {
      this.logger.error('failed to query total # of batches', {
        error: err,
        node: 'l1',
        section: 'getTotalElements',
      })
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'getTotalElements',
      })
      await sleep(15000)
      return
    }

    if (this.state.currentBatchIndex > latestBatchIndex) {
      this.logger.info('batch index is ahead of the oracle. waiting...', {
        batchIndex: this.state.currentBatchIndex,
        latestBatchIndex,
      })
      await sleep(15000)
      return
    }

    this.metrics.highestBatchIndex.set({ type: 'known' }, latestBatchIndex)
    this.logger.info('checking batch', {
      batchIndex: this.state.currentBatchIndex,
      latestBatchIndex,
    })

    let event: PartialEvent
    try {
      event = await findEventForStateBatch(
        this.state.oo,
        this.state.currentBatchIndex,
        this.logger
      )
    } catch (err) {
      this.logger.error('failed to fetch event associated with batch', {
        error: err,
        node: 'l1',
        section: 'findEventForStateBatch',
        batchIndex: this.state.currentBatchIndex,
      })
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'findEventForStateBatch',
      })
      await sleep(15000)
      return
    }

    let latestBlock: number
    try {
      latestBlock = await this.options.l2RpcProvider.getBlockNumber()
    } catch (err) {
      this.logger.error('failed to query L2 block height', {
        error: err,
        node: 'l2',
        section: 'getBlockNumber',
      })
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l2',
        section: 'getBlockNumber',
      })
      await sleep(15000)
      return
    }

    if (this.options.bedrock) {
      const outputBlockNumber = event.args.l2BlockNumber.toNumber()
      if (latestBlock < outputBlockNumber) {
        this.logger.info('L2 node is behind, waiting for sync...', {
          l2BlockHeight: latestBlock,
          outputBlock: outputBlockNumber,
        })
        return
      }

      let outputBlock: any
      try {
        outputBlock = await (
          this.options.l2RpcProvider as ethers.providers.JsonRpcProvider
        ).send('eth_getBlockByNumber', [
          toRpcHexString(outputBlockNumber),
          false,
        ])
      } catch (err) {
        this.logger.error('failed to fetch output block', {
          error: err,
          node: 'l2',
          section: 'getBlock',
          block: outputBlockNumber,
        })
        this.metrics.nodeConnectionFailures.inc({
          layer: 'l2',
          section: 'getBlock',
        })
        await sleep(15000)
        return
      }

      let messagePasserProofResponse: any
      try {
        messagePasserProofResponse = await (
          this.options.l2RpcProvider as ethers.providers.JsonRpcProvider
        ).send('eth_getProof', [
          this.state.messenger.contracts.l2.BedrockMessagePasser.address,
          [],
          toRpcHexString(outputBlockNumber),
        ])
      } catch (err) {
        this.logger.error('failed to fetch message passer proof', {
          error: err,
          node: 'l2',
          section: 'getProof',
          block: outputBlockNumber,
        })
        this.metrics.nodeConnectionFailures.inc({
          layer: 'l2',
          section: 'getProof',
        })
        await sleep(15000)
        return
      }

      const outputRoot = ethers.utils.solidityKeccak256(
        ['uint256', 'bytes32', 'bytes32', 'bytes32'],
        [
          0,
          outputBlock.stateRoot,
          messagePasserProofResponse.storageHash,
          outputBlock.hash,
        ]
      )

      if (outputRoot !== event.args.outputRoot) {
        this.state.diverged = true
        this.metrics.isCurrentlyMismatched.set(1)
        this.logger.error('state root mismatch', {
          blockNumber: outputBlock.number,
          expectedStateRoot: event.args.outputRoot,
          actualStateRoot: outputRoot,
          finalizationTime: dateformat(
            new Date(
              (ethers.BigNumber.from(outputBlock.timestamp).toNumber() +
                this.state.fpw) *
                1000
            ),
            'mmmm dS, yyyy, h:MM:ss TT'
          ),
        })
        return
      }
    } else {
      let batchTransaction: Transaction
      try {
        batchTransaction = await this.options.l1RpcProvider.getTransaction(
          event.transactionHash
        )
      } catch (err) {
        this.logger.error('failed to acquire batch transaction', {
          error: err,
          node: 'l1',
          section: 'getTransaction',
        })
        this.metrics.nodeConnectionFailures.inc({
          layer: 'l1',
          section: 'getTransaction',
        })
        await sleep(15000)
        return
      }

      const [stateRoots] = this.state.oo.contract.interface.decodeFunctionData(
        'appendStateBatch',
        batchTransaction.data
      )

      const batchStart = event.args._prevTotalElements.toNumber() + 1
      const batchSize = event.args._batchSize.toNumber()
      const batchEnd = batchStart + batchSize

      if (latestBlock < batchEnd) {
        this.logger.info('L2 node is behind. waiting for sync...', {
          batchBlockStart: batchStart,
          batchBlockEnd: batchEnd,
          l2BlockHeight: latestBlock,
        })
        return
      }

      // `getBlockRange` has a limit of 1000 blocks, so we have to break this request out into
      // multiple requests of maximum 1000 blocks in the case that batchSize > 1000.
      let blocks: any[] = []
      for (let i = 0; i < batchSize; i += 1000) {
        let newBlocks: any[]
        try {
          newBlocks = await (
            this.options.l2RpcProvider as ethers.providers.JsonRpcProvider
          ).send('eth_getBlockRange', [
            toRpcHexString(batchStart + i),
            toRpcHexString(batchStart + i + Math.min(batchSize - i, 1000) - 1),
            false,
          ])
        } catch (err) {
          this.logger.error('failed to query for blocks in batch', {
            error: err,
            node: 'l2',
            section: 'getBlockRange',
          })
          this.metrics.nodeConnectionFailures.inc({
            layer: 'l2',
            section: 'getBlockRange',
          })
          await sleep(15000)
          return
        }

        blocks = blocks.concat(newBlocks)
      }

      for (const [i, stateRoot] of stateRoots.entries()) {
        if (blocks[i].stateRoot !== stateRoot) {
          this.state.diverged = true
          this.metrics.isCurrentlyMismatched.set(1)
          this.logger.error('state root mismatch', {
            blockNumber: blocks[i].number,
            expectedStateRoot: blocks[i].stateRoot,
            actualStateRoot: stateRoot,
            finalizationTime: dateformat(
              new Date(
                (ethers.BigNumber.from(blocks[i].timestamp).toNumber() +
                  this.state.fpw) *
                  1000
              ),
              'mmmm dS, yyyy, h:MM:ss TT'
            ),
          })
          return
        }
      }
    }

    const elapsedMs = Date.now() - startMs

    // Mark the current batch index as checked
    this.logger.info('checked batch ok', {
      batchIndex: this.state.currentBatchIndex,
      timeMs: elapsedMs,
    })
    this.metrics.highestBatchIndex.set(
      { type: 'checked' },
      this.state.currentBatchIndex
    )

    // If we got through the above without throwing an error, we should be
    // fine to reset and move onto the next batch
    this.state.diverged = false
    this.state.currentBatchIndex++
    this.metrics.isCurrentlyMismatched.set(0)
  }
}

if (require.main === module) {
  config()
  const service = new FaultDetector()
  service.run()
}
