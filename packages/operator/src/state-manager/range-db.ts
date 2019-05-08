/* External Imports */
import level from 'level'
import BigNum = require('bn.js')

/* Internal Imports */
import { itNext, itEnd } from '../utils'

/* Logging */
import debug from 'debug'
const log = debug('info:range-db')

type Endianness = 'le' | 'be'

const lte = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) <= 0
}

export class RangeDB {
  constructor (readonly db: level, readonly prefix: Buffer, readonly keyLength: number=16, readonly endianness: Endianness='be') {
  }

  private addPrefix(target: Buffer) {
    return Buffer.concat([this.prefix, target])
  }

  private bnToKey(bigNum: BigNum) {
    const buf = bigNum.toBuffer(this.endianness, this.keyLength)
    return Buffer.concat([this.prefix, buf])
  }

  private getStartFromValue(value: Buffer) {
    return value.slice(0, this.keyLength)
  }

  private getDataFromValue(value: Buffer) {
    return value.slice(this.keyLength)
  }

  private isCorrectPrefix(key: Buffer): boolean {
    if (typeof key === 'undefined') {
      return false
    }
    return Buffer.compare(this.prefix, key.slice(0, this.prefix.length)) === 0
  }

  public async put(start: BigNum, end: BigNum, value: Buffer) {
    log('Putting range:', start.toString('hex'), end.toString('hex'), value)
    const endBuffer = end.toBuffer(this.endianness, this.keyLength)
    const keyWithPrefix = this.addPrefix(endBuffer)
    const valueThing = Buffer.concat([start.toBuffer(this.endianness, this.keyLength), value])
    await this.db.put(keyWithPrefix, valueThing)
  }

  private resultToRange(result) {
    // Helper function which gets the start and end position from a DB seek result
    return {
      start: new BigNum(this.getStartFromValue(result.value)),
      end: new BigNum(result.key.slice(this.prefix.length)),
      value: this.getDataFromValue(result.value)
    }
  }

  public async get(start: BigNum, end: BigNum) {
    log('Getting range:', start, end)
    const it = this.db.iterator({
      gt: this.bnToKey(start),
      keyAsBuffer: true
    })
    const ranges = []
    let result = await itNext(it)
    // First make sure that the resulting value has the correct prefix.
    if (!this.isCorrectPrefix(result.key)) {
      // If not return because this means there are no values yet put in this RangeDB
      await itEnd(it)
      return []
    }
    // Next check if the resulting value surrounds our query. If so we want to return just that value.
    if (lte(this.getStartFromValue(result.value), this.bnToKey(start)) && // result start <= query start
        lte(this.bnToKey(end), result.key)                                // query end <= result end
    ) {
      // Add the first result to our return values if it surrounds our query and return the ranges
      ranges.push(this.resultToRange(result))
      await itEnd(it)
      return ranges
    }
    log('these are the ranges', ranges)
    while (
      this.isCorrectPrefix(result.key) &&
      Buffer.compare(result.key, this.bnToKey(end)) <= 0
    ) {
      ranges.push(this.resultToRange(result))
      result = await itNext(it)
    }
    await itEnd(it)
    return ranges
  }
}
