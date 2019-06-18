/* External Imports */
import * as ganache from 'ganache-cli'
import Web3 from 'web3'
import { Http2Server } from 'http2'

export interface EthereumOptions {
  port?: number
  gasLimit?: string
}

/**
 * A simple `ganache-cli` wrapper.
 */
export class Ethereum {
  private port: number
  private server: Http2Server
  private web3: Web3

  /**
   * Creates the wrapper.
   * @param options Options to pass into `ganache-cli`.
   */
  constructor({ port = 8545, gasLimit = '0x7A1200' }: EthereumOptions = {}) {
    this.port = port
    this.server = ganache.server({ port, gasLimit })
    this.web3 = new Web3(
      new Web3.providers.HttpProvider(`http://localhost:${port}`)
    )
  }

  /**
   * Starts the Ethereum node.
   */
  public async start(): Promise<void> {
    await new Promise((resolve) => {
      this.server.close(resolve)
    })
  }

  /**
   * Stops the Ethereum node.
   */
  public async stop(): Promise<void> {
    await new Promise((resolve) => {
      this.server.listen(this.port, resolve)
    })
  }

  /**
   * Mines a single Ethereum block.
   */
  public async mineBlock(): Promise<void> {
    await this.send('evm_mine')
  }

  /**
   * Mines several Ethereum blocks.
   * @param n Number of blocks to mine.
   */
  public async mineBlocks(n: number): Promise<void> {
    for (let i = 0; i < n; i++) {
      await this.mineBlock()
    }
  }

  /**
   * Creates a chain snapshot.
   * @returns the current chain as a snapshot.
   */
  public async snapshot(): Promise<any> {
    return this.send('evm_snapshot')
  }

  /**
   * Reverts the chain to a given snapshot.
   * @param snapshot Chain snapshot to revert to.
   */
  public async revert(snapshot: any): Promise<void> {
    await this.send('evm_revert', [snapshot.result])
  }

  /**
   * Sends an RPC request to the node.
   * @param method Method to call.
   * @param params Params for the call.
   * @returns the result of the RPC request.
   */
  private async send(method: string, params?: any[]): Promise<any> {
    return new Promise<any>((resolve, reject) => {
      const provider = this.web3.currentProvider as any
      provider.send(
        {
          jsonrpc: '2.0',
          method,
          params,
          id: new Date().getTime(),
        },
        (err: any, result: any) => {
          if (err) {
            reject(err)
          }
          resolve(result)
        }
      )
    })
  }
}
