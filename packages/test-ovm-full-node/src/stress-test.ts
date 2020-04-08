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
  requestsPerMilli: number
  responseMetrics: Metrics
  confirmMetrics: Metrics
}

export abstract class FullNodeStressTest {
  private requestNumber

  protected constructor(
    protected readonly numberOfRequests,
    protected readonly nodeUrl
  ) {

  }

  public async run(): Promise<TestResult> {
    await this.deployContract()

    this.requestNumber = 0
    const promises: Array<Promise<RequestResult>> = []
    for (let i = 0; i < this.numberOfRequests; i++) {
      promises.push(this.processSingleRequest())
    }

    const startTime: number = Date.now()
    const requestResults: RequestResult[] = await Promise.all(promises)
    const totalTimeMillis: number = Date.now() - startTime

    // Do some maths
    const results: TestResult = {
      requestCount: this.numberOfRequests,
      requestResults: [],
      totalTimeMillis,
      requestsPerMilli: totalTimeMillis / this.numberOfRequests,
      responseMetrics: this.getMetrics(requestResults.map(x => x.responseDurationMillis)),
      confirmMetrics: this.getMetrics(requestResults.map(x => x.confirmationDurationMillis))
    }

    log.info(`Test results: ${JSON.stringify(results)}`)
    return results
  }

  protected abstract deployContract(): Promise<void>

  protected abstract getSignedTransaction(): Promise<string>

  private async processSingleRequest(): Promise<RequestResult> {
    const request = await this.getSignedTransaction()
    const provider: JsonRpcProvider = new JsonRpcProvider(this.nodeUrl)
    const index: number = this.requestNumber++

    const startTime: number = Date.now()

    const response = await provider.sendTransaction(request)

    const responseTime: number = Date.now()

    // await provider.waitForTransaction(response.hash)

    const confirmTime: number = Date.now()

    return {
      index,
      request,
      response: JSON.stringify(response),
      responseDurationMillis: responseTime - startTime,
      confirmationDurationMillis: confirmTime - startTime,
    }
  }

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

  private static getWorstPercentileMean(data: number[], percentile: number): number {
    const percentileData: number[] = data.sort((a, b) => a - b)
      .slice(Math.floor(data.length * (100 - percentile) / 100))
    return percentileData.reduce((a,b) => a + b, 0) / percentileData.length
  }
}