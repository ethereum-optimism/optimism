/* External Imports */
import {getLogger, Logger, sleep} from '@eth-optimism/core-utils'
import {JsonRpcProvider} from 'ethers/providers'

const log: Logger = getLogger('stress-test')

export interface Metrics {
  bestMillis: number,
  worstMillis: number
  meanDurationMillis: number
  medianDurationMillis: number
  worstTenPercentMillis: number
  worstFivePercentMillis: number
  worstOnePercentMillis: number
}

export interface RequestResult {
  index: number
  request: string
  response: string
  responseDurationMillis: number
  confirmationDurationMillis: number
}

export interface TestResult {
  requestResults: RequestResult[]
  requestCount: number
  totalTimeMillis: number
  requestsPerSecond: number
  responseMetrics: Metrics
  confirmMetrics: Metrics
}

/**
 * Base class handling generic stress test functionality,
 * leaving what is being tested to the implementor.
 */
export abstract class FullNodeStressTest {
  private requestNumber

  protected constructor(
    protected readonly numberOfRequests,
    protected readonly nodeUrl
  ) {}

  /**
   * Runs a stress test with the configured number of requests and node URL using
   * the contract deployed by the implementing class and the signed transactions
   * created by the implementing class.
   */
  public async run(): Promise<TestResult> {
    await this.deployContract()

    this.requestNumber = 0

    const signedTransactions: string[] = []
    for (let i = 0; i < this.numberOfRequests; i++) {
      signedTransactions.push(await this.getSignedTransaction())
    }

    const promises: Array<Promise<RequestResult>> = []
    for (let i = 0; i < this.numberOfRequests; i++) {
      promises.push(this.processSingleRequest(signedTransactions[i]))
    }

    const startTime: number = Date.now()
    const requestResults: RequestResult[] = await Promise.all(promises)
    const totalTimeMillis: number = Date.now() - startTime

    const results: TestResult = {
      requestCount: this.numberOfRequests,
      requestResults: [], // not included by default because it's way too verbose
      totalTimeMillis,
      requestsPerSecond: this.numberOfRequests / totalTimeMillis * 1_000,
      responseMetrics: this.getMetrics(requestResults.map(x => x.responseDurationMillis)),
      confirmMetrics: this.getMetrics(requestResults.map(x => x.confirmationDurationMillis))
    }

    log.info(`Test results: ${JSON.stringify(results)}`)
    return results
  }

  /**
   * Deploys the contract to be used for this stress test.
   */
  protected abstract deployContract(): Promise<void>

  /**
   * Gets unique signed transactions to be used for the stress test.
   * Note: to make sure there aren't any parallel execution issues,
   * a different wallet should be used for each.
   */
  protected abstract getSignedTransaction(): Promise<string>

  /**
   * Handles executing a single request and capturing metrics around it.
   *
   * @param signedTransaction The signed transaction to execute.
   * @returns The RequestResult with the metrics for this transaction.
   */
  private async processSingleRequest(signedTransaction: string): Promise<RequestResult> {
    const provider: JsonRpcProvider = new JsonRpcProvider(this.nodeUrl)
    const index: number = this.requestNumber++

    const startTime: number = Date.now()
    const response = await provider.sendTransaction(signedTransaction)
    const responseTime: number = Date.now()

    await provider.waitForTransaction(response.hash)
    const confirmTime: number = Date.now()

    return {
      index,
      request: signedTransaction,
      response: JSON.stringify(response),
      responseDurationMillis: responseTime - startTime,
      confirmationDurationMillis: confirmTime - startTime,
    }
  }

  /**
   * Gets the metrics for a specific set of data, including best, worst, mean, etc.
   *
   * @param data The array of numerical data to operate on
   * @returns The Metrics for the dataset.
   */
  private getMetrics(data: number[]): Metrics {
    const sortedData = data.sort((a,b) => a - b)
    return {
      bestMillis: sortedData[0],
      worstMillis: sortedData[sortedData.length -1],
      meanDurationMillis: sortedData.reduce((a, b) => a + b, 0) / this.numberOfRequests,
      medianDurationMillis: sortedData[Math.floor(this.numberOfRequests / 2)],
      worstTenPercentMillis: FullNodeStressTest.getWorstPercentileMean(sortedData, 10),
      worstFivePercentMillis: FullNodeStressTest.getWorstPercentileMean(sortedData, 5),
      worstOnePercentMillis: FullNodeStressTest.getWorstPercentileMean(sortedData, 1)
    }
  }

  /**
   * Utility function to get the mean for a percentile range below a specific percentile.
   *
   * @param data The data to use for the calculations.
   * @param percentile The percentile defining the lowest N % of data to be included in the mean calc.
   * @returns The mean.
   */
  private static getWorstPercentileMean(data: number[], percentile: number): number {
    const percentileData: number[] = data.sort((a, b) => a - b)
      .slice(Math.floor(data.length * (100 - percentile) / 100))
    return percentileData.reduce((a,b) => a + b, 0) / percentileData.length
  }
}