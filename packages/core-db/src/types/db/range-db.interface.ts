/* External Imports */
import { BigNumber, Endianness } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  Bucket,
  IteratorOptions,
  KeyValueStore,
  RangeIterator,
  V,
} from './db.interface'

/**
 * Represents a range of values. Note start & end are big numbers!
 */
export interface RangeEntry {
  start: BigNumber
  end: BigNumber
  value: V
}

/**
 * Represents a key value store which uses ranges as keys
 */
export interface RangeStore {
  readonly db: KeyValueStore
  readonly prefix: Buffer // TODO: Use core's standard prefix management
  readonly keyLength: number // The number of bytes which should be used for the range keys
  readonly endianness: Endianness // The endianness of the range keys

  /**
   * Determines if there is any data with a Range overlapping the specified Range.
   * @param start the start of the Range
   * @param end the end of the Range
   * @returns true if there is data in the Range, false otherwise
   */
  hasDataInRange(start: BigNumber, end: BigNumber): Promise<boolean>

  /**
   * Queries for all values which are stored over the particular range.
   * @param start the start of the range to query.
   * @param end the start of the range to query.
   * @returns an array of values which are stored at intersecting ranges.
   */
  get(start: BigNumber, end: BigNumber): Promise<RangeEntry[]>

  /**
   * Sets a range to be equal to a particular value
   * @param start the start of the range which we will store the value at.
   * @param end the end of the range which we will store the value at.
   * @param value the value which will be stored
   */
  put(start: BigNumber, end: BigNumber, value: Buffer): Promise<void>

  /**
   * Deletes all values stored over a given range.
   * @param start the start of the range which will be deleted.
   * @param end the end of the range which will be deleted.
   * @returns all of the ranges which have been deleted.
   */
  del(start: BigNumber, end: BigNumber): Promise<RangeEntry[]>

  /**
   * Creates an iterator with some options.
   * @param options Parameters for the iterator.
   * @returns the iterator instance.
   */
  iterator(options?: IteratorOptions): RangeIterator
}

export interface RangeBucket extends RangeStore {
  /**
   * Creates a prefixed bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the bucket.
   * @returns the bucket instance.
   */
  bucket(prefix: Buffer): Bucket

  /**
   * Creates a prefixed range bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the bucket.
   * @returns the range bucket instance.
   */
  rangeBucket(prefix: Buffer): RangeBucket
}
