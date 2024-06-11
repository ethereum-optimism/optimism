import {
  BaseServiceV2,
  StandardOptions,
  ExpressRouter,
  Gauge,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import {
  BedrockOutputData,
  getChainId,
  sleep,
  toRpcHexString,
} from '@eth-optimism/core-utils'
import { config } from 'dotenv'
import {
  CONTRACT_ADDRESSES,
  CrossChainMessenger,
  getOEContract,
  L2ChainID,
  OEL1ContractsLike,
} from '@eth-optimism/sdk'
import { Provider } from '@ethersproject/abstract-provider'
import { Contract, ethers } from 'ethers'
import dateformat from 'dateformat'

import { version } from '../../package.json'
import { findFirstUnfinalizedOutputIndex, findOutputForIndex } from './helpers'

type Options = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  startOutputIndex: number
  optimismPortalAddress?: string
}

type Metrics = {
  highestOutputIndex: Gauge
  isCurrentlyMismatched: Gauge
  nodeConnectionFailures: Gauge
}

type State = {
  faultProofWindow: number
  outputOracle: Contract
  messenger: CrossChainMessenger
  currentOutputIndex: number
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
        startOutputIndex: {
          validator: validators.num,
          default: -1,
          desc: 'The L2 height to start from',
          public: true,
        },
        optimismPortalAddress: {
          validator: validators.str,
          default: ethers.constants.AddressZero,
          desc: '[Custom OP Chains] Deployed OptimismPortal contract address. Used to retrieve necessary info for output verification ',
          public: true,
        },
      },
      metricsSpec: {
        highestOutputIndex: {
          type: Gauge,
          desc: 'Highest output indices (checked and known)',
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
   * - OptimismPortal (used to also fetch L2OutputOracle address variable). This is the preferred address
   * since in early versions of bedrock, OptimismPortal holds the FINALIZATION_WINDOW variable instead of L2OutputOracle.
   * The retrieved L2OutputOracle address from OptimismPortal is used to query for output roots.
   *
   * @param l2ChainId op chain id
   * @returns OEL1ContractsLike set of L1 contracts with only the required addresses set
   */
  async getOEL1Contracts(l2ChainId: number): Promise<OEL1ContractsLike> {
    // CrossChainMessenger requires all address to be defined. Default to `AddressZero` to ignore unused contracts
    let contracts: OEL1ContractsLike = {
      OptimismPortal: ethers.constants.AddressZero,
      L2OutputOracle: ethers.constants.AddressZero,
      // Unused contracts
      AddressManager: ethers.constants.AddressZero,
      BondManager: ethers.constants.AddressZero,
      CanonicalTransactionChain: ethers.constants.AddressZero,
      L1CrossDomainMessenger: ethers.constants.AddressZero,
      L1StandardBridge: ethers.constants.AddressZero,
      StateCommitmentChain: ethers.constants.AddressZero,
    }

    const knownChainId = L2ChainID[l2ChainId] !== undefined
    if (knownChainId) {
      this.logger.info(`Recognized L2 chain id ${L2ChainID[l2ChainId]}`)

      // fallback to the predefined defaults for this chain id
      contracts = CONTRACT_ADDRESSES[l2ChainId].l1
    }

    this.logger.info('checking contract address options...')
    const portalAddress = this.options.optimismPortalAddress
    if (!knownChainId && portalAddress === ethers.constants.AddressZero) {
      this.logger.error('OptimismPortal contract unspecified')
      throw new Error(
        '--optimismportalcontractaddress needs to set for custom op chains'
      )
    }

    if (portalAddress !== ethers.constants.AddressZero) {
      this.logger.info('set OptimismPortal contract override')
      contracts.OptimismPortal = portalAddress

      this.logger.info('fetching L2OutputOracle contract from OptimismPortal')
      const portalContract = getOEContract('OptimismPortal', l2ChainId, {
        address: portalAddress,
        signerOrProvider: this.options.l1RpcProvider,
      })
      contracts.L2OutputOracle = await portalContract.l2Oracle()
    }

    // ... for a known chain ids without an override, the L2OutputOracle will already
    // be set via the hardcoded default
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
      bedrock: true,
      contracts: { l1: await this.getOEL1Contracts(l2ChainId) },
    })

    // Not diverged by default.
    this.state.diverged = false

    // We use this a lot, a bit cleaner to pull out to the top level of the state object.
    this.state.faultProofWindow =
      await this.state.messenger.getChallengePeriodSeconds()
    this.logger.info(
      `fault proof window is ${this.state.faultProofWindow} seconds`
    )

    this.state.outputOracle = this.state.messenger.contracts.l1.L2OutputOracle

    // Figure out where to start syncing from.
    if (this.options.startOutputIndex === -1) {
      this.logger.info('finding appropriate starting unfinalized output')
      const firstUnfinalized = await findFirstUnfinalizedOutputIndex(
        this.state.outputOracle,
        this.state.faultProofWindow,
        this.logger
      )

      // We may not have an unfinalized outputs in the case where no outputs have been submitted
      // for the entire duration of the FAULTPROOFWINDOW. We generally do not expect this to happen on mainnet,
      // but it happens often on testnets because the FAULTPROOFWINDOW is very short.
      if (firstUnfinalized === undefined) {
        this.logger.info(
          'no unfinalized outputes found. skipping all outputes.'
        )
        const totalOutputes = await this.state.outputOracle.nextOutputIndex()
        this.state.currentOutputIndex = totalOutputes.toNumber() - 1
      } else {
        this.state.currentOutputIndex = firstUnfinalized
      }
    } else {
      this.state.currentOutputIndex = this.options.startOutputIndex
    }

    this.logger.info('starting output', {
      outputIndex: this.state.currentOutputIndex,
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

    let latestOutputIndex: number
    try {
      const totalOutputes = await this.state.outputOracle.nextOutputIndex()
      latestOutputIndex = totalOutputes.toNumber() - 1
    } catch (err) {
      this.logger.error('failed to query total # of outputes', {
        error: err,
        node: 'l1',
        section: 'nextOutputIndex',
      })
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'nextOutputIndex',
      })
      await sleep(15000)
      return
    }

    if (this.state.currentOutputIndex > latestOutputIndex) {
      this.logger.info('output index is ahead of the oracle. waiting...', {
        outputIndex: this.state.currentOutputIndex,
        latestOutputIndex,
      })
      await sleep(15000)
      return
    }

    this.metrics.highestOutputIndex.set({ type: 'known' }, latestOutputIndex)
    this.logger.info('checking output', {
      outputIndex: this.state.currentOutputIndex,
      latestOutputIndex,
    })

    let outputData: BedrockOutputData
    try {
      outputData = await findOutputForIndex(
        this.state.outputOracle,
        this.state.currentOutputIndex,
        this.logger
      )
    } catch (err) {
      this.logger.error('failed to fetch output associated with output', {
        error: err,
        node: 'l1',
        section: 'findOutputForIndex',
        outputIndex: this.state.currentOutputIndex,
      })
      this.metrics.nodeConnectionFailures.inc({
        layer: 'l1',
        section: 'findOutputForIndex',
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

    const outputBlockNumber = outputData.l2BlockNumber
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
      ).send('eth_getBlockByNumber', [toRpcHexString(outputBlockNumber), false])
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

    if (outputRoot !== outputData.outputRoot) {
      this.state.diverged = true
      this.metrics.isCurrentlyMismatched.set(1)
      this.logger.error('state root mismatch', {
        blockNumber: outputBlock.number,
        expectedStateRoot: outputData.outputRoot,
        actualStateRoot: outputRoot,
        finalizationTime: dateformat(
          new Date(
            (ethers.BigNumber.from(outputBlock.timestamp).toNumber() +
              this.state.faultProofWindow) *
              1000
          ),
          'mmmm dS, yyyy, h:MM:ss TT'
        ),
      })
      return
    }

    const elapsedMs = Date.now() - startMs

    // Mark the current output index as checked
    this.logger.info('checked output ok', {
      outputIndex: this.state.currentOutputIndex,
      timeMs: elapsedMs,
    })
    this.metrics.highestOutputIndex.set(
      { type: 'checked' },
      this.state.currentOutputIndex
    )

    // If we got through the above without throwing an error, we should be
    // fine to reset and move onto the next output
    this.state.diverged = false
    this.state.currentOutputIndex++
    this.metrics.isCurrentlyMismatched.set(0)
  }
}

if (require.main === module) {
  config()
  const service = new FaultDetector()
  service.run()
}
