/**
 * Modified from bcoin's bdb (https://github.com/bcoin-org/bdb) (MIT LICENSE).
 * Credit to the original author, Christopher Jeffrey (https://github.com/chjj).
 */

/* Internal Imports */
import {
  RangeBucket,
  Batch,
  Bucket,
  DB,
  Iterator,
  IteratorOptions,
  K,
  V,
} from '../types'

/**
 * Simple bucket implementation that forwards all
 * calls up to the database but appends a prefix.
 */
export class BaseBucket implements Bucket {
  constructor(readonly db: DB, readonly prefix: Buffer) {}

  /**
   * Queries the value at a given key.
   * @param key Key to query.
   * @returns the value at that key.
   */
  public async get(key: K): Promise<V> {
    return this.db.get(this.addPrefix(key))
  }

  /**
   * Sets the value at a given key.
   * @param key Key to set.
   * @param value Value to set to.
   */
  public async put(key: K, value: V): Promise<void> {
    return this.db.put(this.addPrefix(key), value)
  }

  /**
   * Deletes a given key.
   * @param key Key to delete.
   */
  public async del(key: K): Promise<void> {
    return this.db.del(this.addPrefix(key))
  }

  /**
   * Checks whether a given key is set.
   * @param key Key to query.
   * @returns `true` if the key is set, `false` otherwise.
   */
  public async has(key: K): Promise<boolean> {
    return this.db.has(this.addPrefix(key))
  }

  /**
   * Performs a series of operations in batch.
   * @param operations Operations to perform.
   */
  public async batch(operations: ReadonlyArray<Batch>): Promise<void> {
    return this.db.batch(
      operations.map((op) => {
        return {
          ...op,
          key: this.addPrefix(op.key),
        }
      })
    )
  }

  /**
   * Creates an iterator with some options.
   * @param options Parameters for the iterator.
   * @returns the iterator instance.
   */
  public iterator(options?: IteratorOptions): Iterator {
    return this.db.iterator({
      ...options,
      prefix: this.addPrefix(options.prefix),
    })
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
}
