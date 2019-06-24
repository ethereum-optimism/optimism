/* External Imports */
import {
  AbstractIteratorOptions,
  AbstractLevelDOWN,
  AbstractOpenOptions,
} from 'abstract-leveldown'

/* Internal Imports */
import { RangeBucket } from '../../types'

export type K = NonNullable<Buffer>
export type V = NonNullable<Buffer>
export interface KV {
  key: K
  value: V
}

export type Batch = PutBatch | DelBatch

export interface PutBatch {
  readonly type: 'put'
  readonly key: K
  readonly value: V
}

export interface DelBatch {
  readonly type: 'del'
  readonly key: K
}

/**
 * KeyValueStore represents a basic collection of key:value pairs.
 */
export interface KeyValueStore {
  /**
   * Queries the value at a given key.
   * @param key Key to query.
   * @returns the value at that key.
   */
  get(key: K): Promise<V>

  /**
   * Sets the value at a given key.
   * @param key Key to set.
   * @param value Value to set to.
   */
  put(key: K, value: V): Promise<void>

  /**
   * Deletes a given key.
   * @param key Key to delete.
   */
  del(key: K): Promise<void>

  /**
   * Checks whether a given key is set.
   * @param key Key to query.
   * @returns `true` if the key is set, `false` otherwise.
   */
  has(key: K): Promise<boolean>

  /**
   * Performs a series of operations in batch.
   * @param operations Operations to perform.
   */
  batch(operations: ReadonlyArray<Batch>): Promise<void>

  /**
   * Creates an iterator with some options.
   * @param options Parameters for the iterator.
   * @returns the iterator instance.
   */
  iterator(options?: IteratorOptions): Iterator

  /**
   * Creates a prefixed bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the bucket.
   * @returns the bucket instance.
   */
  bucket(prefix: K): Bucket

  /**
   * Creates a prefixed range bucket underneath
   * this bucket.
   * @param prefix Prefix to use for the range bucket.
   * @returns the range bucket instance.
   */
  rangeBucket(prefix: K): RangeBucket
}

/**
 * Represents a key:value store.
 */
export interface DB extends KeyValueStore {
  readonly db: AbstractLevelDOWN

  /**
   * Opens the store.
   * @param [options] Database options.
   */
  open(options?: AbstractOpenOptions): Promise<void>

  /**
   * Closes the store.
   */
  close(): Promise<void>
}

/**
 * Bucket are effectively databases that only perform operations
 * on keys that share a common `prefix`.
 */
export interface Bucket extends KeyValueStore {
  readonly db: DB
  readonly prefix: K
}

export interface IteratorOptions extends AbstractIteratorOptions {
  gte?: K
  lte?: K
  gt?: K
  lt?: K
  reverse?: boolean
  limit?: number
  keys?: boolean
  values?: boolean
  keyAsBuffer?: boolean
  valueAsBuffer?: boolean
  prefix?: Buffer
}

/**
 * Iterators traverse over ranges of keys.
 * Iterators operate on a *snapshot* of the store
 * and not on the store itself. As a result,
 * the iterator is not impacted by writes
 * made after the iterator was created.
 */
export interface Iterator {
  readonly db: DB

  /**
   * Advances the iterator to the next key.
   * @returns the entry at the next key.
   */
  next(): Promise<{ key: K; value: V }>

  /**
   * Seeks the iterator to the target key.
   * @param target Key to seek to.
   */
  seek(target: K): Promise<void>

  /**
   * Executes a function for each key:value
   * pair in the iterator.
   * @param cb Function to be executed.
   */
  each(cb: (key: Buffer, value: Buffer) => any): Promise<void>

  /**
   * @returns the items in the iterator.
   */
  items(): Promise<KV[]>

  /**
   * @returns all keys in the iterator.
   */
  keys(): Promise<K[]>

  /**
   * @returns all values in the iterator.
   */
  values(): Promise<V[]>

  /**
   * Ends iteration and frees resources.
   */
  end(): Promise<void>
}
