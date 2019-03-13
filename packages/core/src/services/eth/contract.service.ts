/* External Imports */
import { Service, OnStart } from '@nestd/core'
import BigNum from 'bn.js'
import * as compiledContracts from '@pigi/contracts'
import Web3 = require('web3')
import Contract from 'web3/eth/contract'
import { EventLog } from 'web3/types'
import * as web3Utils from 'web3-utils'

/* Services */
import { WalletService } from '../wallet/wallet.service'
import { EventService } from '../event.service'
import { LoggerService } from '../logger.service'
import { ConfigService } from '../config.service'

/* Internal Imports */
import { Deposit } from '../../models/chain'
import { EthereumEvent, EthereumTransactionReceipt } from '../../models/eth'
import { ChainCreatedEvent } from '../../models/events'
import { CONFIG } from '../../constants'

/* Compiled Contracts */
const plasmaChainCompiled = compiledContracts.plasmaChainCompiled
const erc20Compiled = compiledContracts.erc20Compiled
const registryCompiled = compiledContracts.plasmaRegistryCompiled

interface ContractOptions {
  registryAddress: string
  plasmaChainName: string
}

@Service()
export class ContractService implements OnStart {
  private contract: Contract
  private registry: Contract
  private endpoint?: string
  private web3: Web3
  private readonly name = 'contract'

  constructor(
    private readonly events: EventService,
    private readonly logger: LoggerService,
    private readonly config: ConfigService,
    private readonly wallet: WalletService
  ) {}

  public async onStart(): Promise<void> {
    this.initContractInfo()

    this.web3 = new Web3(
      new Web3.providers.HttpProvider(this.ethereumEndpoint())
    )
    this.contract = new this.web3.eth.Contract(plasmaChainCompiled.abi)
    this.registry = new this.web3.eth.Contract(
      registryCompiled.abi,
      this.options().registryAddress
    )
  }

  /**
   * @returns Address of the connected contract.
   */
  get address(): string | null {
    return this.contract.options.address
  }

  /**
   * @returns the ABI of the contract.
   */
  get abi(): any[] {
    return plasmaChainCompiled.abi
  }

  /**
   * @returns `true` if the contract has an address, `false` otherwise.
   */
  get hasAddress(): boolean {
    return this.address !== null
  }

  /**
   * @returns `true` if the contract is ready to be used, `false` otherwise.
   */
  get ready(): boolean {
    return this.hasAddress && this.endpoint !== undefined
  }

  /**
   * @returns Address of the connected contract.
   */
  get operatorEndpoint(): string {
    return this.endpoint
  }

  /**
   * @returns Plasma Chain contract name.
   */
  get plasmaChainName(): string {
    return this.options().plasmaChainName
  }

  /**
   * Waits for the contract to have an address.
   * @returns the address.
   */
  public async waitForAddress(): Promise<string> {
    if (this.hasAddress) {
      return this.address
    } else {
      return new Promise<string>((resolve) => {
        this.events.on('contract.Initialized', async () => {
          resolve(this.address)
        })
      })
    }
  }

  /**
   * Queries a given block.
   * @param block Number of the block to query.
   * @returns Root hash of the block with that number.
   */
  public async getBlock(block: number): Promise<string> {
    return this.contract.methods.blockHashes(block).call()
  }

  /**
   * @returns Number of the block that will be submitted next.
   */
  public async getNextBlock(): Promise<number> {
    return this.contract.methods.nextPlasmaBlockNumber().call()
  }

  /**
   * @returns Number of the last submitted block.
   */
  public async getCurrentBlock(): Promise<number> {
    const nextBlockNumber = await this.getNextBlock()
    return nextBlockNumber - 1
  }

  /**
   * @returns Address of the current operator.
   */
  public async getOperator(): Promise<string> {
    return this.contract.methods.operator().call()
  }

  /**
   * Returns the address for a given token ID.
   * @param token The token ID.
   * @returns Address of the contract for that token.
   */
  public async getTokenAddress(token: string): Promise<string> {
    if (web3Utils.isAddress(token)) {
      return token
    }

    // tslint:disable-next-line:no-string-literal
    return this.contract.methods['listings__contractAddress'](
      token.toString()
    ).call()
  }

  /**
   * Lists a token with the given address
   * so that it can be deposited.
   * @param tokenAddress Address of the token.
   * @param sender Address of the account sending the listToken transaction.
   * @returns The Ethereum transaction result.
   */
  public async listToken(
    tokenAddress: string,
    sender: string
  ): Promise<EthereumTransactionReceipt> {
    sender = sender || (await this.wallet.getAccounts())[0]
    await this.wallet.addAccountToWallet(sender)

    const tx = this.contract.methods.listToken(tokenAddress, 0)
    const gas = await tx.estimateGas({ from: sender })
    return tx.send({ from: sender, gas })
  }

  /**
   * Gets the current challenge period.
   * Challenge period is returned in number of blocks.
   * @returns Current challenge period.
   */
  public async getChallengePeriod(): Promise<number> {
    // tslint:disable-next-line:no-string-literal
    return this.contract.methods['CHALLENGE_PERIOD']().call()
  }

  /**
   * Gets the token ID for a specific token.
   * Token IDs are unique to each plasma chain.
   * TODO: Add link that explains how token IDs work.
   * @param tokenAddress Token contract address.
   * @returns ID of that token.
   */
  public async getTokenId(tokenAddress: string): Promise<string> {
    return this.contract.methods.listed(tokenAddress).call()
  }

  /**
   * Returns past events for the contract
   * @param event The name of the event.
   * @param filter The filter object.
   * @returns past events with the given filter.
   */
  public async getPastEvents(
    event: string,
    filter: {} = {}
  ): Promise<EthereumEvent[]> {
    const events: EventLog[] = await this.contract.getPastEvents(event, filter)
    return events.map((ethereumEvent) => {
      return EthereumEvent.from(ethereumEvent)
    })
  }

  /**
   * Checks whether a specific deposit actually exists.
   * @param deposit Deposit to check.
   * @returns `true` if the deposit exists, `false` otherwise.
   */
  public async depositValid(deposit: Deposit): Promise<boolean> {
    // Find past deposit events.
    const depositEvents = await this.getPastEvents('DepositEvent', {
      filter: {
        depositer: deposit.owner,
        // block: deposit.block
      },
      fromBlock: 0,
    })

    // Convert the events to deposit objects.
    const deposits = depositEvents.map(Deposit.from)

    // Check that one of the events matches this deposit.
    return deposits.some(deposit.equals)
  }

  /**
   * Submits a deposit for a user.
   * This method will pipe the `deposit` call to the correct
   * ERC20 or ETH call.
   * @param token Token to deposit, specified by ID.
   * @param amount Amount to deposit.
   * @param owner Address of the user to deposit for.
   * @returns Deposit transaction receipt.
   */
  public async deposit(
    token: BigNum,
    amount: BigNum,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    await this.wallet.addAccountToWallet(owner)

    amount = new BigNum(amount, 'hex')
    if (token.toString() === '0') {
      return this.depositETH(amount, owner)
    } else {
      return this.depositERC20(token, amount, owner)
    }
  }

  /**
   * Starts an exit for a user.
   * Exits can only be started on *transfers*, meaning you
   * need to specify the block in which the transfer was received.
   * TODO: Add link that explains this in more detail.
   * @param block Block in which the transfer was received.
   * @param token Token to be exited.
   * @param start Start of the range received in the transfer.
   * @param end End of the range received in the transfer.
   * @param owner Adress to exit from.
   * @returns Exit transaction receipt.
   */
  public async startExit(
    block: BigNum,
    token: BigNum,
    start: BigNum,
    end: BigNum,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    await this.wallet.addAccountToWallet(owner)

    return this.contract.methods
      .beginExit(token, block, start, end)
      .send({ from: owner, gas: 200000 })
  }

  /**
   * Finalizes an exit for a user.
   * @param exitId ID of the exit to finalize.
   * @param exitableEnd Weird quirk in how we handle exits. For more
   * information, see:
   * https://github.com/plasma-group/plasma-contracts/issues/44.
   * @param owner Address that owns this exit.
   * @returns Finalization transaction receipt.
   */
  public async finalizeExit(
    exitId: string,
    exitableEnd: BigNum,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    await this.wallet.addAccountToWallet(owner)

    return this.contract.methods
      .finalizeExit(exitId, exitableEnd)
      .send({ from: owner, gas: 100000 })
  }

  /**
   * Submits a block with the given hash.
   * Will only work if the operator's account is unlocked and
   * available to the node.
   * @param hash Hash of the block to submit.
   * @returns Block submission transaction receipt.
   */
  public async submitBlock(hash: string): Promise<EthereumTransactionReceipt> {
    const operator = await this.getOperator()
    await this.wallet.addAccountToWallet(operator)

    return this.contract.methods.submitBlock(hash).send({ from: operator })
  }

  /**
   * @returns any contract options.
   */
  private options(): ContractOptions {
    return this.config.get(CONFIG.CONTRACT_OPTIONS)
  }

  /**
   * @returns the current Ethereum endpoint.
   */
  private ethereumEndpoint(): string {
    return this.config.get(CONFIG.ETHEREUM_ENDPOINT)
  }

  /**
   * Deposits an amount of ETH for a user.
   * @param amount Amount to deposit.
   * @param owner Address of the user to deposit for.
   * @returns the deposit transaction receipt.
   */
  private async depositETH(
    amount: BigNum,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    return this.contract.methods
      .depositETH()
      .send({ from: owner, value: amount.toString(10), gas: 150000 })
  }

  /**
   * Deposits an amount of an ERC20 for a user.
   * @param token Token to deposit.
   * @param amount Amount to deposit.
   * @param owner Address of the user to deposit for.
   * @returns the deposit transaction receipt.
   */
  private async depositERC20(
    token: BigNum,
    amount: BigNum,
    owner: string
  ): Promise<EthereumTransactionReceipt> {
    const tokenAddress = await this.getTokenAddress(token.toString(10))
    const tokenContract = new this.web3.eth.Contract(
      erc20Compiled.abi,
      tokenAddress
    )
    await tokenContract.methods.approve(this.address, amount).send({
      from: owner,
      gas: 6000000, // TODO: Figure out how much this should be.
    })
    return this.contract.methods.depositERC20(tokenAddress, amount).send({
      from: owner,
      gas: 6000000, // TODO: Figure out how much this should be.
    })
  }

  /**
   * Initializes the contract address and operator endpoint.
   * Queries information from the registry.
   */
  private async initContractInfo() {
    if (!this.plasmaChainName) {
      throw new Error('ERROR: Plasma chain name not provided.')
    }

    const plasmaChainName = web3Utils
      .asciiToHex(this.plasmaChainName)
      .padEnd(66, '0')
    const operator = await this.registry.methods
      .plasmaChainNames(plasmaChainName)
      .call()
    const events = await this.registry.getPastEvents('NewPlasmaChain', {
      filter: { OperatorAddress: operator },
      fromBlock: 0,
    })

    // Parse the events into something useable.
    const parsed = events.map(ChainCreatedEvent.from)

    // Find a matching event.
    const event = parsed.find((parsedEvent: ChainCreatedEvent) => {
      return parsedEvent.plasmaChainName === plasmaChainName
    })

    if (!event) {
      throw new Error('ERROR: Plasma chain name not found in registry.')
    }

    // Set the appropriate instance variables.
    this.contract.options.address = event.plasmaChainAddress
    this.endpoint = event.operatorEndpoint

    // Let other services know that the contract is ready.
    this.events.event(this.name, 'Initialized')

    this.logger.log(
      this.name,
      `Connected to plasma chain: ${this.plasmaChainName}`
    )
    this.logger.log(this.name, `Contract address set: ${this.address}`)
    this.logger.log(this.name, `Operator endpoint set: ${this.endpoint}`)
  }
}
