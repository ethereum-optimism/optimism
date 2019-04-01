import {
  Bucket,
  BaseDB,
  Batch,
  IteratorOptions,
  Iterator,
  K,
  V,
} from '../../../interfaces'

/**
 * Simple bucket implementation that forwards all
 * calls up to the database but appends a prefix.
 */
export class BaseBucket implements Bucket {
  constructor(readonly db: BaseDB, readonly prefix: Buffer) {}

  public async get(key: K): Promise<V> {
    return this.db.get(this.addPrefix(key))
  }

  public async put(key: K, value: V): Promise<void> {
    return this.db.put(this.addPrefix(key), value)
  }

  public async del(key: K): Promise<void> {
    return this.db.del(this.addPrefix(key))
  }

  public async has(key: K): Promise<boolean> {
    return this.db.has(this.addPrefix(key))
  }

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

  public iterator(options?: IteratorOptions): Iterator {
    return this.db.iterator({
      ...options,
      prefix: this.addPrefix(options.prefix),
    })
  }

  public bucket(prefix: Buffer): Bucket {
    return this.db.bucket(this.addPrefix(prefix))
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
