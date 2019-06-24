/* External Imports */
import { AbstractOpenOptions } from 'abstract-leveldown'
import BigNum = require('bn.js')
import level from 'level'

/* Internal Imports */
import { KeyValueStore, V, Bucket } from './db.interface'

/**
 * Represents a range of values. Note start & end are big numbers!
 */
export interface RangeEntry {
  start: BigNum
  end: BigNum
  value: V
}

export type Endianness = 'le' | 'be'

/**
 * Represents a key value store which uses ranges as keys
 */
export interface RangeStore {
  readonly db: KeyValueStore | level // TODO: Remove direct use of level & replace with generic KeyValueStore
  readonly prefix: Buffer // TODO: Use core's standard prefix management
  readonly keyLength: number // The number of bytes which should be used for the range keys
  readonly endianness: Endianness // The endianness of the range keys

  /**
   * Queries for all values which are stored over the particular range.
   * @param start the start of the range to query.
   * @param end the start of the range to query.
   * @returns an array of values which are stored at intersecting ranges.
   */
  get(start: BigNum, end: BigNum): Promise<RangeEntry[]>

  /**
   * Sets a range to be equal to a particular value
   * @param start the start of the range which we will store the value at.
   * @param end the end of the range which we will store the value at.
   * @param value the value which will be stored
   */
  put(start: BigNum, end: BigNum, value: Buffer): Promise<void>

  /**
   * Deletes all values stored over a given range.
   * @param start the start of the range which will be deleted.
   * @param end the end of the range which will be deleted.
   * @returns all of the ranges which have been deleted.
   */
  del(start: BigNum, end: BigNum): Promise<RangeEntry[]>
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
