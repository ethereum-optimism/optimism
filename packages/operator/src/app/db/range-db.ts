/* External Imports */
import level from 'level'
import BigNum = require('bn.js')

/* Internal Imports */
import { itNext, itEnd } from '../utils'
import { RangeStore, Range, Endianness } from '../../interfaces/db/range-db.interface'

/* Logging */
import debug from 'debug'
const log = debug('info:range-db')

/**
 * Checks if buf1 is less than or equal to buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 <= buf2
 */
const lte = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) <= 0
}

/**
 * Checks if buf1 is strictly less than buf2
 * @param buf1 the first Buffer
 * @param buf2 the second Buffer
 * @returns boolean result of evaluating buf1 < buf2
 */
const lt = (buf1: Buffer, buf2: Buffer): boolean => {
  return Buffer.compare(buf1, buf2) < 0
}

/**
 * A RangeStore which uses Level as a backend.
 */
export class LevelRangeStore implements RangeStore {

  /**
   * Creates the LevelRangeStore.
   * @param db Pointer to the Level instance to be used.
   * @param prefix A Buffer which is prepended to each range key.
   * @param keyLength The number of bytes which should be used for the range keys.
   * @param endianness The endianness of the range keys.
   */
  constructor (
    readonly db: level,
    readonly prefix: Buffer,
    readonly keyLength: number=16,
    readonly endianness: Endianness='be'
  ) {}

  /**
   * Adds this RangeStore's prefix to the target Buffer
   * @param target A Buffer which will have the prefix prepended to it.
   * @returns resulting Buffer `prefix+target`.
   */
  private addPrefix(target: Buffer): Buffer {
    return Buffer.concat([this.prefix, target])
  }

  /**
   * Adds the start position of a range to a Buffer value.
   * This is used to generate the value we store in the DB
   * because each range is stored internally as `end->start+data`.
   * @param start A BigNumber representing the start position.
   * @param value A Buffer value, likely to be stored.
   * @returns resulting concatenation of the start key & input value
   */
  private addStartToValue(start: BigNum, value: Buffer): Buffer {
    return Buffer.concat([start.toBuffer(this.endianness, this.keyLength), value])
  }

  /**
   * Extracts the start out of a Buffer which contains `start+data`
   * @param value A buffer which contains a start & the actual data
   * @returns the start as a Buffer
   */
  private getStartFromValue(value: Buffer): Buffer {
    return value.slice(0, this.keyLength)
  }

  /**
   * Extracts the data out of a Buffer which contains `start+data`
   * @param value A buffer which contains a start & the actual data
   * @returns the start as a Buffer
   */
  private getDataFromValue(value: Buffer): Buffer {
    return value.slice(this.keyLength)
  }

  /**
   * Turns a BigNumber into a Buffer which can be used as a range key.
   * In particular this means prepending the buffer with our prefix & padding zeros
   * to make sure the start/end has the proper length.
   * @param start A BigNumber representing the start position.
   * @param value A Buffer value, likely to be stored.
   * @returns resulting concatenation of the start key & input value
   */
  private bnToKey(bigNum: BigNum): Buffer {
    const buf = bigNum.toBuffer(this.endianness, this.keyLength)
    return Buffer.concat([this.prefix, buf])
  }

  /**
   * Checks if the Buffer value contains the correct prefix.
   * @param key A buffer key to our database.
   * @returns true if the key starts with our prefix, false otherwise.
   */
  private isCorrectPrefix(key: Buffer): boolean {
    if (typeof key === 'undefined') {
      return false
    }
    return Buffer.compare(this.prefix, key.slice(0, this.prefix.length)) === 0
  }

  /**
   * Validates the range input to make sure that start < end.
   * @param start The start of the range.
   * @param end The end of the range.
   * @returns true if start > end, false otherwise.
   */
  private validateRange(start: BigNum, end: BigNum): void {
    // Make sure start is less than end
    if (!start.lt(end)) {
      throw new Error('Start not less than end')
    }
  }

  /**
   * Checks if two ranges intersect, eg. [1,10) & [8,11) would return true.
   * @param start1 The start of the first range.
   * @param end1 The end of the first range.
   * @param start2 The start of the second range.
   * @param end2 The end of the second range.
   * @returns true if max(start1, start2) < min(end1, end2), false otherwise.
   */
  private intersects(start1: Buffer, end1: Buffer, start2: Buffer, end2: Buffer): boolean {
    const maxStart = (lte(start1, start2)) ? start2 : start1
    const minEnd = (lte(end1, end2)) ? end1 : end2
    return lt(maxStart, minEnd)
  }

  /**
   * Transforms a result of the DB query (key, value) into a range object.
   * @param result The resulting value which has been extracted from our DB.
   * @returns a range object with {start, end, value}
   */
  private resultToRange(result): Range {
    // Helper function which gets the start and end position from a DB seek result
    return {
      start: new BigNum(this.getStartFromValue(result.value)),
      end: new BigNum(result.key.slice(this.prefix.length)),
      value: this.getDataFromValue(result.value)
    }
  }

  /**
   * Iterates through the DB to find all overlapping ranges & constructs an array of
   * batch operations to delete them. This is used in `del()` and `put()`
   * @param start The start of the range which we want deletion batch operations for.
   * @param end The end of the range which we want deletion batch operations for.
   * @returns an object which contains both the ranges we queried & the batch deletion operations.
   */
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

  /**
   * Puts a new range in the DB. Note that it maps the values to a range.
   * Sometimes putting a new range will split old ranges, or delete them entirely.
   * For example: put(0,5,'$') might result in `$$$$$`, then put(1,4,'#') would result in `$###$`.
   * @param start The start of the range which we are putting values into.
   * @param end The end of the range which we are putting values into.
   * @param value The value which we will be putting in these ranges.
   */
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

  /**
   * Deletes all ranges which intersect with [start,end)
   * @param start The start of the range we are deleting.
   * @param end The end of the range we are deleting.
   * @returns all of the ranges which have been deleted.
   */
  public async del(start: BigNum, end: BigNum): Promise<Range[]> {
    // Delete all overlapping ranges and return the values which have been deleted
    const {ranges, batchOps} = await this.delBatchOps(start, end)
    await this.db.batch(batchOps)
    return ranges
  }

  /**
   * Gets all ranges which intersect with [start,end)
   * @param start The start of the range we are getting.
   * @param end The end of the range we are getting.
   * @returns all of the ranges which have been gotten.
   */
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
