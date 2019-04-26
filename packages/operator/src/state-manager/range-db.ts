/* External Imports */
import level from 'level'
import BigNum = require('bn.js')

/* Internal Imports */
import { itNext, itEnd } from '../utils'

/* Logging */
import debug from 'debug'
const log = debug('test:info:range-db')

type Endianness = 'le' | 'be'

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
    await this.db.put(keyWithPrefix, Buffer.concat([this.bnToKey(start), value]))
  }

  public async get(start: BigNum, end: BigNum) {
    log('Getting range:', start, end)
    const it = this.db.iterator({
      gt: this.bnToKey(start),
      keyAsBuffer: true
    })
    const ranges = []
    let result = await itNext(it)
    while (
      this.isCorrectPrefix(result.key) &&
      Buffer.compare(result.key, this.bnToKey(end)) < 0
    ) {
      ranges.push({
        start: result.value.slice(0, this.keyLength),
        end: result.key,
        value: result.value.slice(this.keyLength)
      })
      result = await itNext(it)
    }
    await itEnd(it)
    return ranges
  }

  public async getContiguous(start: BigNum, end: BigNum) {
    log('Getting range:', start, end)
  }
}
