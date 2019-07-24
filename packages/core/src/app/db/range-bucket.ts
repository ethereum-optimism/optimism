/**
 * Modified from bcoin's bdb (https://github.com/bcoin-org/bdb) (MIT LICENSE).
 * Credit to the original author, Christopher Jeffrey (https://github.com/chjj).
 */

/* Internal Imports */
import {
  Bucket,
  Batch,
  DB,
  IteratorOptions,
  KV,
  RangeBucket,
  RangeEntry,
  PUT_BATCH_TYPE,
  RangeIterator,
} from '../../types'
import {
  intersects,
  BigNumber,
  BIG_ENDIAN,
  Endianness,
  ZERO,
  BaseRangeIterator,
} from '../../app'

/* Logging */
import debug from 'debug'
const log = debug('info:range-db')

/**
 * Simple bucket implementation that forwards all
 * calls up to the database but appends a prefix.
 */
export class BaseRangeBucket implements RangeBucket {
  /**
   * Creates the RangeBucket.
   * @param db Pointer to the Level instance to be used.
   * @param prefix A Buffer which is prepended to each range key.
   * @param keyLength The number of bytes which should be used for the range keys.
   * @param endianness The endianness of the range keys.
   */
  constructor(
    readonly db: DB,
    readonly prefix: Buffer,
    readonly keyLength: number = 16,
    readonly endianness: Endianness = BIG_ENDIAN
  ) {}

  /**
   * Concatenates some value to this bucket's prefix.
   * @param value Value to concatenate.
   * @returns the value concatenated to the prefix.
   */
  private addPrefix(value: Buffer): Buffer {
    return value !== undefined
      ? Buffer.concat([this.prefix, value])
      : this.prefix
  }

  /**
   * Adds the start position of a range to a Buffer value.
   * This is used to generate the value we store in the DB
   * because each range is stored internally as `end->start+data`.
   * @param start A BigNumber representing the start position.
   * @param value A Buffer value, likely to be stored.
   * @returns resulting concatenation of the start key & input value
   */
  private addStartToValue(start: BigNumber, value: Buffer): Buffer {
    return Buffer.concat([
      start.toBuffer(this.endianness, this.keyLength),
      value,
    ])
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
  private bnToKey(bigNum: BigNumber): Buffer {
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
   * Validates the range input to make sure that start < end and start >= 0.
   * @param start The start of the range.
   * @param end The end of the range.
   * @returns true if start > end, false otherwise.
   */
  private validateRange(start: BigNumber, end: BigNumber): void {
    // Make sure start is less than end
    if (!start.lt(end)) {
      throw new Error('Start not less than end')
    }
    // Make sure start is greater than or equal to zero
    if (!start.gte(ZERO)) {
      throw new Error('Start less than zero')
    }
  }

  /**
   * Transforms a result of the DB query (key, value) into a range object.
   * @param result The resulting value which has been extracted from our DB.
   * @returns a range object with {start, end, value}
   */
  private resultToRange(result: KV): RangeEntry {
    // Helper function which gets the start and end position from a DB seek result
    return {
      start: new BigNumber(this.getStartFromValue(result.value)),
      end: new BigNumber(result.key.slice(this.prefix.length)),
      value: this.getDataFromValue(result.value),
    }
  }

  /**
   * Iterates through the DB to find all overlapping ranges & constructs an array of
   * batch operations to delete them. This is used in `del()` and `put()`
   * @param start The start of the range which we want deletion batch operations for.
   * @param end The end of the range which we want deletion batch operations for.
   * @returns an object which contains both the ranges we queried & the batch deletion operations.
   */
  private async delBatchOps(
    start: BigNumber,
    end: BigNumber
  ): Promise<{ ranges: RangeEntry[]; batchOps: Batch[] }> {
    this.validateRange(start, end)
    const ranges = await this.get(start, end)
    const batchOps = []
    for (const range of ranges) {
      batchOps.push({
        type: 'del',
        key: this.bnToKey(range.end),
      })
    }
    return { ranges, batchOps }
  }

  /**
   * Puts a new range in the DB. Note that it maps the values to a range.
   * Sometimes putting a new range will split old ranges, or delete them entirely.
   * For example: put(0,5,'$') might result in `$$$$$`, then put(1,4,'#') would result in `$###$`.
   * @param start The start of the range which we are putting values into.
   * @param end The end of the range which we are putting values into.
   * @param value The value which we will be putting in these ranges.
   */
  public async put(
    start: BigNumber,
    end: BigNumber,
    value: Buffer
  ): Promise<void> {
    this.validateRange(start, end)
    log(
      'Putting range: [',
      start.toString(16),
      ',',
      end.toString(16),
      ') with value:',
      value
    )

    // Step #1: get all overlapping ranges and queue them for deletion
    //
    const { ranges, batchOps } = await this.delBatchOps(start, end)

    // Step #2: Add back ranges which are split
    //
    // If the start position is greater than the first range's start...
    if (ranges.length > 0 && start.gt(ranges[0].start)) {
      // Reduce the first affected range's end position. Eg: ##### becomes ###$$
      batchOps.push({
        type: PUT_BATCH_TYPE,
        key: this.bnToKey(start),
        value: this.addStartToValue(ranges[0].start, ranges[0].value),
      })
    }
    // If the end position less than the last range's end...
    if (ranges.length > 0 && ranges[ranges.length - 1].end.gt(end)) {
      batchOps.push({
        type: PUT_BATCH_TYPE,
        key: this.bnToKey(ranges[ranges.length - 1].end),
        value: this.addStartToValue(end, ranges[ranges.length - 1].value),
      })
    }

    // Step #3: Add our new range
    //
    batchOps.push({
      type: PUT_BATCH_TYPE,
      key: this.bnToKey(end),
      value: this.addStartToValue(start, value),
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
  public async del(start: BigNumber, end: BigNumber): Promise<RangeEntry[]> {
    // Delete all overlapping ranges and return the values which have been deleted
    const { ranges, batchOps } = await this.delBatchOps(start, end)
    await this.db.batch(batchOps)
    return ranges
  }

  public async hasDataInRange(
    start: BigNumber,
    end: BigNumber
  ): Promise<boolean> {
    // TODO: can eagerly return when true, but this is good enough or now
    return (await this.get(start, end)).length > 0
  }

  /**
   * Gets all ranges which intersect with [start,end)
   * @param start The start of the range we are getting.
   * @param end The end of the range we are getting.
   * @returns all of the ranges which have been gotten.
   */
  public async get(start: BigNumber, end: BigNumber): Promise<RangeEntry[]> {
    this.validateRange(start, end)
    log('Getting range: [', start.toString(16), ',', end.toString(16), ')')
    // Seek to the beginning
    const it = this.db.iterator({
      gt: this.bnToKey(start),
      keyAsBuffer: true,
    })
    const ranges = []
    let result = await it.next()
    // First make sure that the resulting value has the correct prefix.
    if (!this.isCorrectPrefix(result.key)) {
      // If not return because this means there are no values yet put in this RangeDB
      await it.end()
      return []
    }
    const queryStart = this.bnToKey(start)
    const queryEnd = this.bnToKey(end)
    let resultStart = this.addPrefix(this.getStartFromValue(result.value))
    let resultEnd = result.key
    while (
      intersects(queryStart, queryEnd, resultStart, resultEnd) &&
      this.isCorrectPrefix(result.key)
    ) {
      // If the query & result intersect, add it to our ranges array
      ranges.push(this.resultToRange(result))
      // Get the next result
      result = await it.next()
      // Make sure the result returned a value
      if (typeof result.key === 'undefined') {
        break
      }
      // Format the result start & end as buffers with the correct prefix
      resultStart = this.addPrefix(this.getStartFromValue(result.value))
      resultEnd = result.key
    }
    // End the iteration
    await it.end()
    // Return the ranges
    return ranges
  }

  /**
   * Creates an iterator with some options.
   * @param options Parameters for the iterator.
   * @returns the iterator instance.
   */
  public iterator(options: IteratorOptions = {}): RangeIterator {
    const rangeIterator = new BaseRangeIterator(
      this.db,
      {
        ...options,
        prefix: this.addPrefix(options.prefix),
      },
      (res: KV) => this.resultToRange(res)
    )
    return rangeIterator
  }

  /**
   * Creates a prefixed bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the bucket.
   * @returns the bucket instance.
   */
  public bucket(prefix: Buffer): Bucket {
    return this.db.bucket(this.addPrefix(prefix))
  }

  /**
   * Creates a prefixed range bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the bucket.
   * @returns the range bucket instance.
   */
  public rangeBucket(prefix: Buffer): RangeBucket {
    return this.db.rangeBucket(this.addPrefix(prefix))
  }
}
