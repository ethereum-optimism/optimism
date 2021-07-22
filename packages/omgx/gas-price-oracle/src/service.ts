/* Imports: External */
import { Contract, Wallet, BigNumber, providers } from 'ethers'
import fetch from 'node-fetch'

/* Imports: Internal */
import { sleep } from '@eth-optimism/core-utils'
import { BaseService } from '@eth-optimism/common-ts'
import { loadContract } from '@eth-optimism/contracts'

interface GasPriceOracleOptions {
  // Providers for interacting with L1 and L2.
  l1RpcProvider: providers.JsonRpcProvider
  l2RpcProvider: providers.JsonRpcProvider

  // Address of the gasPrice contract
  gasPriceOracleAddress: string,

  // Wallet
  deployerWallet: Wallet,
  sequencerWallet: Wallet,
  proposerWallet: Wallet,
  relayerWallet: Wallet,
  fastRelayerWallet: Wallet,

  // Floor pice
  gasFloorPrice: number

  // Roof price
  gasRoofPrice: number

  // Min percent change
  gasPriceMinPercentChange: number

  // Interval in seconds to wait between loops
  pollingInterval: number

  // Etherscan API
  etherscanAPI: string
}

const optionSettings = {}

export class GasPriceOracleService extends BaseService<GasPriceOracleOptions> {
  constructor(options: GasPriceOracleOptions) {
    super('GasPriceOracle', options, optionSettings)
  }

  private state: {
    OVM_GasPriceOracle: Contract
    L1ETHBalance: BigNumber
    L1ETHCostFee: BigNumber
    L2ETHCollectFee: BigNumber
    lastQueriedL1Block: number
    lastQueriedL2Block: number
    avgL2GasLimitPerBlock: BigNumber
    numberOfBlocksInterval: number
    etherscanURL: string
  }

  protected async _init(): Promise<void> {
    this.logger.info('Initializing gas price oracle', {
      gasPriceOracleAddress: this.options.gasPriceOracleAddress,
      deployerAddress: this.options.deployerWallet.address,
      sequencerWallet: this.options.sequencerWallet.address,
      proposerWallet: this.options.proposerWallet.address,
      relayerWallet: this.options.relayerWallet.address,
      fastRelayerWallet: this.options.fastRelayerWallet.address,
      gasFloorPrice: this.options.gasFloorPrice,
      pollingInterval: this.options.pollingInterval,
    })

    this.state = { } as any;

    this.logger.info('Connecting to OVM_GasPriceOracle...')
    this.state.OVM_GasPriceOracle = loadContract(
      'OVM_GasPriceOracle',
      this.options.gasPriceOracleAddress,
      this.options.l2RpcProvider,
    ).connect(this.options.deployerWallet)
    this.logger.info('Connected to OVM_GasPriceOracle', {
      address: this.state.OVM_GasPriceOracle.address,
    })

    this.state.L1ETHBalance = BigNumber.from('0')
    this.state.L1ETHCostFee = BigNumber.from('0')
    this.state.L2ETHCollectFee = BigNumber.from('0')

    this.state.lastQueriedL1Block = await this.options.l1RpcProvider.getBlockNumber()
    this.state.lastQueriedL2Block = await this.options.l2RpcProvider.getBlockNumber()

    this.state.avgL2GasLimitPerBlock = BigNumber.from('0')
    this.state.numberOfBlocksInterval = 0

    this.state.etherscanURL = `https://api-rinkeby.etherscan.io/api?module=account&action=txlist&apikey=${this.options.etherscanAPI}`
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      await sleep(this.options.pollingInterval)
      await this._getL1Balance()
      await this._getL2GasCost()
      await this._updateGasPrice()
    }
  }

  private async _getL1Balance(): Promise<void> {
    const balances = await Promise.all([
      this.options.l1RpcProvider.getBalance(this.options.sequencerWallet.address),
      this.options.l1RpcProvider.getBalance(this.options.proposerWallet.address),
      this.options.l1RpcProvider.getBalance(this.options.relayerWallet.address),
      this.options.l1RpcProvider.getBalance(this.options.fastRelayerWallet.address),
    ])
    const L1ETHBalanceLatest = balances.reduce(
      (acc,cur) => { return acc.add(cur) }, BigNumber.from('0')
    )

    const latestQueriedL1Block = await this.options.l1RpcProvider.getBlockNumber()

    // Check if anyone transferred ETH to accounts
    const txResponse = await Promise.all([
      fetch(`${this.state.etherscanURL}&startblock=${this.state.lastQueriedL1Block}&endblock=${latestQueriedL1Block}&address=${this.options.sequencerWallet.address}`),
      fetch(`${this.state.etherscanURL}&startblock=${this.state.lastQueriedL1Block}&endblock=${latestQueriedL1Block}&address=${this.options.proposerWallet.address}`),
      fetch(`${this.state.etherscanURL}&startblock=${this.state.lastQueriedL1Block}&endblock=${latestQueriedL1Block}&address=${this.options.relayerWallet.address}`),
      fetch(`${this.state.etherscanURL}&startblock=${this.state.lastQueriedL1Block}&endblock=${latestQueriedL1Block}&address=${this.options.fastRelayerWallet.address}`),
    ])
    const txHistory = await Promise.all(txResponse.map(response => response.json()))
    const addedETHBalance = txHistory.reduce((acc, cur) => {
      if (Array.isArray(cur.result)) {
        const addedValued = cur.result.filter(i => i.value !== '0').reduce((iacc, icur) => {
          return iacc.add(BigNumber.from(icur.value))
        }, BigNumber.from('0'))
        return acc.add(addedValued)
      }
      return acc
    }, BigNumber.from('0'))

    if (!this.state.L1ETHBalance.eq(BigNumber.from('0'))) {
      this.state.L1ETHCostFee = this.state.L1ETHCostFee.add(
        this.state.L1ETHBalance.add(addedETHBalance).sub(L1ETHBalanceLatest)
      )
    }
    this.state.L1ETHBalance = L1ETHBalanceLatest
    this.state.lastQueriedL1Block = latestQueriedL1Block

    this.logger.info("Got L1 ETH balances", {
      L1ETHBalance: this.state.L1ETHBalance.toString(),
      L1ETHCostFee: this.state.L1ETHCostFee.toString(),
      latestQueriedL1Block: this.state.lastQueriedL1Block,
    })
  }

  private async _getL2GasCost(): Promise<void> {
    const latestQueriedL2Block = await this.options.l2RpcProvider.getBlockNumber()
    const numberOfBlocksInterval = latestQueriedL2Block > this.state.lastQueriedL2Block ?
      latestQueriedL2Block - this.state.lastQueriedL2Block : 1

    const txs = await Promise.all(
      latestQueriedL2Block === this.state.lastQueriedL2Block ?
      [this.options.l2RpcProvider.getBlockWithTransactions(this.state.lastQueriedL2Block)] :
      [...Array(latestQueriedL2Block - this.state.lastQueriedL2Block)]
      .map((_, i) => this.options.l2RpcProvider.getBlockWithTransactions(this.state.lastQueriedL2Block + i + 1))
    )
    const collectGasLimitAndFee = txs.reduce((acc, cur) => {
      return [
        acc[0].add(cur.transactions[0].gasLimit),
        acc[1].add(cur.transactions[0].gasLimit.mul(cur.transactions[0].gasPrice))
      ]
    }, [BigNumber.from('0'), BigNumber.from('0')])

    this.state.L2ETHCollectFee = latestQueriedL2Block !== this.state.lastQueriedL2Block ?
      this.state.L2ETHCollectFee.add(collectGasLimitAndFee[1]) : this.state.L2ETHCollectFee
    this.state.lastQueriedL2Block = latestQueriedL2Block
    this.state.avgL2GasLimitPerBlock = collectGasLimitAndFee[0].div(numberOfBlocksInterval)
    this.state.numberOfBlocksInterval = numberOfBlocksInterval

    this.logger.info("Got L2 Gas Cost", {
      L2ETHCollectFee: this.state.L2ETHCollectFee.toString(),
      lastQueriedL2Block: this.state.lastQueriedL2Block,
      avgL2GasUsagePerBlock: this.state.avgL2GasLimitPerBlock.toString(),
      numberOfBlocksInterval: this.state.numberOfBlocksInterval,
    })
  }

  private async _updateGasPrice(): Promise<void> {
    const gasPrice = await this.state.OVM_GasPriceOracle.gasPrice()
    const gasPriceInt = parseInt(gasPrice.toString())
    this.logger.info("Got L2 gas price", { gasPrice: gasPriceInt })

    let targetGasPrice = this.options.gasFloorPrice

    if (this.state.L1ETHCostFee.gt(this.state.L2ETHCollectFee)) {
      const estimatedGas = BigNumber.from(this.state.numberOfBlocksInterval).mul(this.state.avgL2GasLimitPerBlock)
      const estimatedGasPrice = this.state.L1ETHCostFee.sub(this.state.L2ETHCollectFee).div(estimatedGas)

      if (estimatedGasPrice.gt(BigNumber.from(this.options.gasRoofPrice))) {
        targetGasPrice = this.options.gasRoofPrice
      } else if (estimatedGasPrice.gt(BigNumber.from(this.options.gasFloorPrice))) {
        targetGasPrice = parseInt(estimatedGasPrice.toString())
      }
    }

    if (gasPriceInt !== targetGasPrice && (
      targetGasPrice > (1 + this.options.gasPriceMinPercentChange) * gasPriceInt ||
      targetGasPrice < (1 - this.options.gasPriceMinPercentChange) * gasPriceInt)
    ) {
      this.logger.debug("Updating L2 gas price...")
      const tx = await this.state.OVM_GasPriceOracle.setGasPrice(targetGasPrice, { gasPrice: 0 })
      await tx.wait()
      this.logger.info("Updated L2 gas price", { gasPrice: targetGasPrice })
    } else {
      this.logger.info("No need to update L2 gas price", { gasPrice: gasPriceInt, targetGasPrice })
    }
  }
}