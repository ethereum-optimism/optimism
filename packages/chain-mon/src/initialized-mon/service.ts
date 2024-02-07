import {
  BaseServiceV2,
  StandardOptions,
  Gauge,
  Counter,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import { getChainId, compareAddrs } from '@eth-optimism/core-utils'
import { Provider, TransactionResponse } from '@ethersproject/abstract-provider'
import mainnetConfig from '@eth-optimism/contracts-bedrock/deploy-config/mainnet.json'
import sepoliaConfig from '@eth-optimism/contracts-bedrock/deploy-config/sepolia.json'

import { version } from '../../package.json'

const networks = {
  1: {
    name: 'mainnet',
    l1StartingBlockTag: mainnetConfig.l1StartingBlockTag,
    contracts: [
      {
        label: 'Multisig',
        address: '0x9ba6e03d8b90de867373db8cf1a58d2f7f006b3a',
      },
      {
        label: 'AddressManager',
        address: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
      },
      {
        label: 'L1CrossDomainMessengerProxy',
        address: '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
      },
      {
        label: 'L1ERC721BridgeProxy',
        address: '0x5a7749f83b81B301cAb5f48EB8516B986DAef23D',
      },
      {
        label: 'L1StandardBridgeProxy',
        address: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
      },
      {
        label: 'L2OutputOracleProxy',
        address: '0xdfe97868233d1aa22e815a266982f2cf17685a27',
      },
      {
        label: 'OptimismMintableERC20FactoryProxy	',
        address: '0x75505a97BD334E7BD3C476893285569C4136Fa0F',
      },
      {
        label: 'OptimismPortalProxy',
        address: '0xbEb5Fc579115071764c7423A4f12eDde41f106Ed',
      },
      {
        label: 'ProxyAdmin',
        address: '0x543bA4AADBAb8f9025686Bd03993043599c6fB04',
      },
      {
        label: 'SystemConfigProxy',
        address: '0x229047fed2591dbec1eF1118d64F7aF3dB9EB290',
      },
    ],
  },
  10: {
    name: 'op-mainnet',
    l1StartingBlockTag: null,
    contracts: [
      {
        label: 'L2ToL1MessagePasser',
        address: '0x4200000000000000000000000000000000000016',
      },
      {
        label: 'L2CrossDomainMessenger',
        address: '0x4200000000000000000000000000000000000007',
      },
      {
        label: 'L2StandardBridge',
        address: '0x4200000000000000000000000000000000000010',
      },
      {
        label: 'L2ERC721Bridge',
        address: '0x4200000000000000000000000000000000000014',
      },
      {
        label: 'SequencerFeeWallet',
        address: '0x4200000000000000000000000000000000000011',
      },
      {
        label: 'OptimismMintableERC20Factory',
        address: '0x4200000000000000000000000000000000000012',
      },
      {
        label: 'OptimismMintableERC721Factory',
        address: '0x4200000000000000000000000000000000000017',
      },
      {
        label: 'L1BlockAttributes',
        address: '0x4200000000000000000000000000000000000015',
      },
      {
        label: 'GasPriceOracle',
        address: '0x420000000000000000000000000000000000000F',
      },
      {
        label: 'L1MessageSender',
        address: '0x4200000000000000000000000000000000000001',
      },
      {
        label: 'DeployerWhitelist',
        address: '0x4200000000000000000000000000000000000002',
      },
      {
        label: 'LegacyERC20ETH',
        address: '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
      },
      {
        label: 'L1BlockNumber',
        address: '0x4200000000000000000000000000000000000013',
      },
      {
        label: 'LegacyMessagePasser',
        address: '0x4200000000000000000000000000000000000000',
      },
      {
        label: 'ProxyAdmin',
        address: '0x4200000000000000000000000000000000000018',
      },
      {
        label: 'BaseFeeVault',
        address: '0x4200000000000000000000000000000000000019',
      },
      {
        label: 'L1FeeVault',
        address: '0x420000000000000000000000000000000000001A',
      },
      {
        label: 'GovernanceToken',
        address: '0x4200000000000000000000000000000000000042',
      },
      {
        label: 'SchemaRegistry',
        address: '0x4200000000000000000000000000000000000020',
      },
      {
        label: 'EAS',
        address: '0x4200000000000000000000000000000000000021',
      },
    ],
  },
  11155111: {
    name: 'sepolia',
    l1StartingBlockTag: sepoliaConfig.l1StartingBlockTag,
    contracts: [
      {
        label: 'Multisig',
        address: '0xdee57160aafcf04c34c887b5962d0a69676d3c8b',
      },
      {
        label: 'AddressManager',
        address: '0x9bFE9c5609311DF1c011c47642253B78a4f33F4B',
      },
      {
        label: 'L1CrossDomainMessengerProxy',
        address: '0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef',
      },
      {
        label: 'L1ERC721BridgeProxy',
        address: '0xd83e03D576d23C9AEab8cC44Fa98d058D2176D1f',
      },
      {
        label: 'L1StandardBridgeProxy',
        address: '0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1',
      },
      {
        label: 'L2OutputOracleProxy',
        address: '0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F',
      },
      {
        label: 'OptimismMintableERC20FactoryProxy	',
        address: '0x868D59fF9710159C2B330Cc0fBDF57144dD7A13b',
      },
      {
        label: 'OptimismPortalProxy',
        address: '0x16Fc5058F25648194471939df75CF27A2fdC48BC',
      },
      {
        label: 'ProxyAdmin',
        address: '0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc',
      },
      {
        label: 'SystemConfigProxy',
        address: '0x034edD2A225f7f429A63E0f1D2084B9E0A93b538',
      },
    ],
  },
  11155420: {
    name: 'op-sepolia',
    l1StartingBlockTag: null,
    contracts: [
      {
        label: 'L2ToL1MessagePasser',
        address: '0x4200000000000000000000000000000000000016',
      },
      {
        label: 'L2CrossDomainMessenger',
        address: '0x4200000000000000000000000000000000000007',
      },
      {
        label: 'L2StandardBridge',
        address: '0x4200000000000000000000000000000000000010',
      },
      {
        label: 'L2ERC721Bridge',
        address: '0x4200000000000000000000000000000000000014',
      },
      {
        label: 'SequencerFeeWallet',
        address: '0x4200000000000000000000000000000000000011',
      },
      {
        label: 'OptimismMintableERC20Factory',
        address: '0x4200000000000000000000000000000000000012',
      },
      {
        label: 'OptimismMintableERC721Factory',
        address: '0x4200000000000000000000000000000000000017',
      },
      {
        label: 'L1BlockAttributes',
        address: '0x4200000000000000000000000000000000000015',
      },
      {
        label: 'GasPriceOracle',
        address: '0x420000000000000000000000000000000000000F',
      },
      {
        label: 'L1MessageSender',
        address: '0x4200000000000000000000000000000000000001',
      },
      {
        label: 'DeployerWhitelist',
        address: '0x4200000000000000000000000000000000000002',
      },
      {
        label: 'LegacyERC20ETH',
        address: '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
      },
      {
        label: 'L1BlockNumber',
        address: '0x4200000000000000000000000000000000000013',
      },
      {
        label: 'LegacyMessagePasser',
        address: '0x4200000000000000000000000000000000000000',
      },
      {
        label: 'ProxyAdmin',
        address: '0x4200000000000000000000000000000000000018',
      },
      {
        label: 'BaseFeeVault',
        address: '0x4200000000000000000000000000000000000019',
      },
      {
        label: 'L1FeeVault',
        address: '0x420000000000000000000000000000000000001A',
      },
      {
        label: 'GovernanceToken',
        address: '0x4200000000000000000000000000000000000042',
      },
      {
        label: 'SchemaRegistry',
        address: '0x4200000000000000000000000000000000000020',
      },
      {
        label: 'EAS',
        address: '0x4200000000000000000000000000000000000021',
      },
    ],
  },
}

type InitializedMonOptions = {
  rpc: Provider
  startBlockNumber: number
}

type InitializedMonMetrics = {
  initializedCalls: Counter
  unexpectedRpcErrors: Counter
}

type InitializedMonState = {
  chainId: number
  highestUncheckedBlockNumber: number
}

export class InitializedMonService extends BaseServiceV2<
  InitializedMonOptions,
  InitializedMonMetrics,
  InitializedMonState
> {
  constructor(options?: Partial<InitializedMonOptions & StandardOptions>) {
    super({
      version,
      name: 'initialized-mon',
      loop: true,
      options: {
        loopIntervalMs: 1000,
        ...options,
      },
      optionsSpec: {
        rpc: {
          validator: validators.provider,
          desc: 'Provider for network to monitor balances on',
        },
        startBlockNumber: {
          validator: validators.num,
          default: -1,
          desc: 'L1 block number to start checking from',
          public: true,
        },
      },
      metricsSpec: {
        initializedCalls: {
          type: Gauge,
          desc: 'Successful transactions to tracked contracts emitting initialized event',
          labels: ['label', 'address'],
        },
        unexpectedRpcErrors: {
          type: Counter,
          desc: 'Number of unexpected RPC errors',
          labels: ['section', 'name'],
        },
      },
    })
  }

  protected async init(): Promise<void> {
    // Connect to L1.
    await waitForProvider(this.options.rpc, {
      logger: this.logger,
      name: 'L1',
    })

    this.state.chainId = await getChainId(this.options.rpc)

    const l1StartingBlockTag = networks[this.state.chainId].l1StartingBlockTag

    if (this.options.startBlockNumber === -1) {
      const block_number = l1StartingBlockTag != null ? (await this.options.rpc.getBlock(l1StartingBlockTag)).number : 0
      this.state.highestUncheckedBlockNumber = block_number
    } else {
      this.state.highestUncheckedBlockNumber = this.options.startBlockNumber
    }
  }

  protected async main(): Promise<void> {
    if (
      (await this.options.rpc.getBlockNumber()) <
      this.state.highestUncheckedBlockNumber
    ) {
      this.logger.info('Waiting for new blocks')
      return
    }

    const network = networks[this.state.chainId]
    const contracts = network.contracts

    const block = await this.options.rpc.getBlock(
      this.state.highestUncheckedBlockNumber
    )
    this.logger.info('Checking block', {
      number: block.number,
    })

    const transactions: TransactionResponse[] = []
    for (const txHash of block.transactions) {
      const t = await this.options.rpc.getTransaction(txHash)
      transactions.push(t)
    }

    for (const transaction of transactions) {
      for (const contract of contracts) {
        const to = transaction.to != null ? transaction.to : transaction["creates"]
        if (compareAddrs(contract.address, to)) {
          try {
            const transactionReceipt = await transaction.wait()
            for (const log of transactionReceipt.logs) {
              // keccak256("Initialized(suint8)") = 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498
              if (log.topics.includes('0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498')) {
                this.metrics.initializedCalls.inc({
                  label: contract.label,
                  address: contract.address,
                })
                this.logger.info('initialized event', {
                  label: contract.label,
                  address: contract.address,
                })
              }
            }
          } catch (err) {
            // If error is due to transaction failing, ignore transaction
            if (err.message.length >= 18 && err.message.slice(0, 18) === 'transaction failed') {
              break
            }
            // Otherwise, we have an unexpected RPC error
            this.logger.info(`got unexpected RPC error`, {
              section: 'creations',
              name: 'NULL',
              err,
            })

            this.metrics.unexpectedRpcErrors.inc({
              section: 'creations',
              name: 'NULL',
            })

            return
          }
        }
      }
    }
    this.logger.info('Checked block', {
      number: this.state.highestUncheckedBlockNumber,
    })
    this.state.highestUncheckedBlockNumber++
  }
}

if (require.main === module) {
  const service = new InitializedMonService()
  service.run()
}
