/* External Imports */
import { Service, OnStart } from '@nestd/core'
import axios, { AxiosInstance, AxiosResponse } from 'axios'
import uuidv4 from 'uuid'
import { Transaction, sleep } from '@pigi/utils'

/* Services */
import { LoggerService } from '../logger.service'
import { EventService } from '../event.service'
import { ContractService } from '../eth/contract.service'
import { ConfigService } from '../config.service'

/* Internal Imports */
import { TransactionProof } from '../../models/chain'
import { EthInfo } from '../../models/operator'
import { JSONRPCParam, JSONRPCResponse, JSONRPCResult } from '../../models/rpc'
import { CONFIG } from '../../constants'

interface OperatorOptions {
  operatorPingInterval: number
}

@Service()
export class OperatorService implements OnStart {
  private readonly name = 'operator'
  private readonly prefix = 'pgop_'
  private connected = false
  private pinging = false
  private endpoint?: string
  private http?: AxiosInstance

  constructor(
    private readonly logger: LoggerService,
    private readonly events: EventService,
    private readonly config: ConfigService,
    private readonly contract: ContractService
  ) {}

  public async onStart(): Promise<void> {
    this.events.on('contract.initialized', () => {
      this.initConnection()
    })
    this.startPingInterval()
  }

  /**
   * @returns `true` if we're connected to the operator, `false` otherwise.
   */
  public isConnected(): boolean {
    return this.connected
  }

  /**
   * Returns the next plasma block, according the operator.
   * @return Next plasma block number.
   */
  public async getNextBlock(): Promise<number> {
    const block = await this.handle('getBlockNumber')
    return block as number
  }

  /**
   * Returns information about the smart contract.
   * @return Smart contract info.
   */
  public async getEthInfo(): Promise<EthInfo> {
    return (await this.handle('getEthInfo')) as EthInfo
  }

  /**
   * Returns transaction received by a given address
   * between two given blocks.
   * @param address Address to query.
   * @param startBlock Block to query from.
   * @param endBlock Block to query to.
   * @return List of encoded transactions.
   */
  public async getReceivedTransactions(
    address: string,
    startBlock: number,
    endBlock: number
  ): Promise<Transaction[]> {
    return (await this.handle('getTransactions', [
      address,
      startBlock,
      endBlock,
    ])) as Transaction[]
  }

  /**
   * Gets a transaction proof for a transaction.
   * @param encoded The encoded transaction.
   * @return Proof information for the transaction.
   */
  public async getTransactionProof(txhash: string): Promise<TransactionProof> {
    return (await this.handle('getProof', [txhash])) as TransactionProof
  }

  /**
   * Sends a signed transaction to the operator.
   * @param transaction The encoded transaction.
   * @returns The transaction receipt.
   */
  public async sendTransaction(transaction: Transaction): Promise<string> {
    return (await this.handle('addTransaction', [
      transaction.encoded,
    ])) as string
  }

  /**
   * Attempts to have the operator submit a new block.
   * Probably won't work if the operator is properly
   * configured but used for testing.
   * @returns A promise that resolves once the request goes through.
   */
  public async submitBlock(): Promise<void> {
    await this.handle('newBlock')
  }

  /**
   * @returns operator options.
   */
  private options(): OperatorOptions {
    return this.config.get(CONFIG.OPERATOR_OPTIONS)
  }

  /**
   * Sends a JSON-RPC command as a HTTP POST request.
   * @param method Name of the method to call.
   * @param params Any extra parameters.
   * @returns The result of the operation or an error.
   */
  private async handle(
    method: string,
    params: JSONRPCParam[] = []
  ): Promise<JSONRPCResult | JSONRPCResult[]> {
    if (this.http === undefined) {
      throw new Error('Cannot make request because endpoint has not been set.')
    }

    let response: AxiosResponse
    try {
      response = await this.http.post('/', {
        id: uuidv4(),
        jsonrpc: '2.0',
        method: this.prefix + method,
        params,
      })
    } catch (err) {
      this.logger.error(this.name, 'Operator response failed.', err)
      throw err
    }

    const data: JSONRPCResponse = JSON.parse(response.data)

    if (data.error) {
      throw data.error
    }
    if (data.result === undefined) {
      throw new Error('No result in JSON-RPC response from operator.')
    }

    return data.result
  }

  /**
   * Initializes the connection to the operator.
   */
  private async initConnection(): Promise<void> {
    const endpoint = this.contract.operatorEndpoint
    const baseURL = endpoint.startsWith('http')
      ? endpoint
      : `https://${endpoint}`

    this.endpoint = endpoint
    this.http = axios.create({ baseURL })
  }

  /**
   * Regularly pings the operator to check if it's online.
   */
  private async startPingInterval(): Promise<void> {
    if (this.pinging) {
      return
    }

    this.pinging = true
    this.pingInterval()
  }

  /**
   * Regularly pings the operator to check if it's online.
   */
  private async pingInterval(): Promise<void> {
    try {
      if (this.endpoint !== undefined) {
        await this.getEthInfo()
        if (!this.connected) {
          this.logger.log(this.name, 'Successfully connected to operator.')
        }
        this.connected = true
      }
    } catch (err) {
      this.connected = false
      this.logger.error(
        this.name,
        `Cannot connect to operator. Attempting to reconnect...`,
        err
      )
    } finally {
      await sleep(this.options().operatorPingInterval)
      this.pingInterval()
    }
  }
}
