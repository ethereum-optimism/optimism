import BigNum = require('bn.js')
import _ = require('lodash')
import { isAddress, sha3 } from 'web3-utils'
import { EventLog } from 'web3-core/types'

interface RawEventData {
  [key: string]: string | number
}

interface EventData {
  [key: string]: string | BigNum
}

/**
 * Parses an Ethereum event.
 * Converts number-like strings into BigNums.
 * @param event An Ethereum event.
 * @returns the parsed event values.
 */
const parseEventValues = (event: EventLog): EventData => {
  const values = _.cloneDeep(event.returnValues as RawEventData)
  const parsed: EventData = {}
  for (const key of Object.keys(values)) {
    const value = values[key]
    if (
      typeof value !== 'string' ||
      (!isNaN(Number(value)) && !isAddress(value))
    ) {
      parsed[key] = new BigNum(value, 10)
    }
  }
  return parsed
}

/**
 * Checks whether an object is an EventLog.
 * @param data Object to check.
 * @returns `true` if it's an EventLog, `false` otherwise.
 */
export const isEventLog = (data: any): data is EventLog => {
  return (
    data.blockNumber !== undefined &&
    data.returnValues !== undefined &&
    data.transactionHash !== undefined &&
    data.logIndex !== undefined
  )
}

interface EthereumEventArgs {
  raw: RawEventData
  data: EventData
  block: BigNum
  hash: string
}

/**
 * Represents an Ethereum event log object.
 */
export class EthereumEvent {
  /**
   * Creates an EthereumEvent from an EthereumEvent.
   * @param event The EthereumEvent to cast.
   * @returns the ExitStartedEvent object.
   */
  public static fromEventLog(event: EventLog): EthereumEvent {
    return new EthereumEvent({
      block: new BigNum(event.blockNumber, 10),
      data: parseEventValues(event),
      hash: sha3(event.transactionHash + event.logIndex),
      raw: event.returnValues as RawEventData,
    })
  }

  /**
   * Creates an EthereumEvent from some arguments.
   * @param args The arguments to cast.
   * @returns the EthereumEvent object.
   */
  public static from(args: EventLog): EthereumEvent {
    if (isEventLog(args)) {
      return EthereumEvent.fromEventLog(args)
    }

    throw new Error('Cannot cast to EthereumEvent.')
  }

  public raw: RawEventData
  public data: EventData
  public block: BigNum
  public hash: string

  constructor(event: EthereumEventArgs) {
    this.raw = event.raw
    this.data = event.data
    this.block = event.block
    this.hash = event.hash
  }
}
