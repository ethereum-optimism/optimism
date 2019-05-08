/* External Imports */
import level from 'level'
import BigNum = require('bn.js')

/* Internal Imports */
import { itNext, itEnd } from '../utils'

/* Logging */
import debug from 'debug'
const log = debug('info:range-db')

type Endianness = 'le' | 'be'

interface Range {
  start: BigNum,
  end: BigNum,
  value: Buffer
}

const lte = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) <= 0
}

const lt = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) < 0
}

export class RangeDB {
  constructor (readonly db: level, readonly prefix: Buffer, readonly keyLength: number=16, readonly endianness: Endianness='be') {
  }

  private addPrefix(target: Buffer): Buffer {
    return Buffer.concat([this.prefix, target])
  }

  private addStartToValue(start: BigNum, value: Buffer): Buffer {
    return Buffer.concat([start.toBuffer(this.endianness, this.keyLength), value])
  }

  private bnToKey(bigNum: BigNum): Buffer {
    const buf = bigNum.toBuffer(this.endianness, this.keyLength)
    return Buffer.concat([this.prefix, buf])
  }

  private getStartFromValue(value: Buffer): Buffer {
    return value.slice(0, this.keyLength)
  }

  private getDataFromValue(value: Buffer): Buffer {
    return value.slice(this.keyLength)
  }

  private isCorrectPrefix(key: Buffer): boolean {
    if (typeof key === 'undefined') {
      return false
    }
    return Buffer.compare(this.prefix, key.slice(0, this.prefix.length)) === 0
  }

  private validateRange(start: BigNum, end: BigNum): void {
    // Make sure start is less than end
    if (!start.lt(end)) {
      throw new Error('Start not less than end')
    }
  }

  private intersects(start1: Buffer, end1: Buffer, start2: Buffer, end2: Buffer): boolean {
    // max(start1, start2) < min(end1, end2)
    const maxStart = (lte(start1, start2)) ? start2 : start1
    const minEnd = (lte(end1, end2)) ? end1 : end2
    return lt(maxStart, minEnd)
  }

  public async put(start: BigNum, end: BigNum, value: Buffer): Promise<void> {
    this.validateRange(start, end)
    log('Putting range: [', start.toString(16), ',', end.toString(16), ') with value:', value)

    // Step #1: get all overlapping ranges and queue them for deletion
    //
    const {ranges, batchOps} = await this.delBatchOps(start, end)

    // Step #2: Add back ranges which are split
    //
    // If the start position is not equal to the first range's start...
    if (ranges.length > 0 && !ranges[0].start.eq(start)) {
      // Reduce the first affected range's end position. Eg: ##### becomes ###$$
      batchOps.push({
        type: 'put',
        key: this.bnToKey(start),
        value: this.addStartToValue(ranges[0].start, ranges[0].value)
      })
    }
    // If the end position is not equal to the last range's end...
    if (ranges.length > 0 && !ranges[ranges.length - 1].end.eq(end)) {
      // Increase the last affected range's start position. Eg: ##### becomes $$###
      batchOps.push({
        type: 'put',
        key: this.bnToKey(ranges[ranges.length - 1].end),
        value: this.addStartToValue(end, ranges[0].value)
      })
    }

    // Step #3: Add our new range
    //
    batchOps.push({
      type: 'put',
      key: this.bnToKey(end),
      value: this.addStartToValue(start, value)
    })

    // Step #4: Execute the batch!
    //
    await this.db.batch(batchOps)
  }

  private resultToRange(result): Range {
    // Helper function which gets the start and end position from a DB seek result
    return {
      start: new BigNum(this.getStartFromValue(result.value)),
      end: new BigNum(result.key.slice(this.prefix.length)),
      value: this.getDataFromValue(result.value)
    }
  }

  private async delBatchOps(start: BigNum, end: BigNum): Promise<any> {
    this.validateRange(start, end)
    const ranges = await this.get(start, end)
    const batchOps = []
    for (const range of ranges) {
      batchOps.push({
        type: 'del',
        key: this.bnToKey(range.end)
      })
    }
    return {ranges, batchOps}
  }

  public async del(start: BigNum, end: BigNum, returnOperations: boolean = false): Promise<Range[]> {
    // Delete all overlapping ranges and return the values which have been deleted
    const {ranges, batchOps} = await this.delBatchOps(start, end)
    await this.db.batch(batchOps)
    return ranges
  }

  public async get(start: BigNum, end: BigNum): Promise<Range[]> {
    this.validateRange(start, end)
    log('Getting range: [', start.toString(16), ',', end.toString(16), ')')
    // Seek to the beginning
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
    const queryStart = this.bnToKey(start)
    const queryEnd = this.bnToKey(end)
    let resultStart = this.addPrefix(this.getStartFromValue(result.value))
    let resultEnd = result.key
    while (this.intersects(queryStart, queryEnd, resultStart, resultEnd)) {
      // If the query & result intersect, add it to our ranges array
      ranges.push(this.resultToRange(result))
      // Get the next result
      result = await itNext(it)
      // Make sure the result returned a value
      if (typeof(result.key) === 'undefined') {
        break
      }
      // Format the result start & end as buffers with the correct prefix
      resultStart = this.addPrefix(this.getStartFromValue(result.value))
      resultEnd = result.key
    }
    // End the iteration
    await itEnd(it)
    // Return the ranges
    return ranges
  }
}
