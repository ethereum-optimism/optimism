import {
  BaseDB,
  DBStatus,
  K,
  V,
  Batch,
  IteratorOptions,
  Bucket,
  KeyValueStore,
  Iterator,
} from '../../../interfaces'
import { BaseBucket } from './base-bucket'

export class EphemDB implements BaseDB {
  private _status: DBStatus = 'new'
  private kvs: Map<K, V>

  get status(): DBStatus {
    return this._status
  }

  /**
   * Opens the store.
   */
  public async open(): Promise<void> {
    this._status = 'open'
  }

  /**
   * Closes the store.
   */
  public async close(): Promise<void> {
    this._status = 'closed'
  }

  public async get(key: K): Promise<V> {
    return this.kvs.get(key)
  }

  public async put(key: K, value: V): Promise<void> {
    this.kvs.set(key, value)
  }

  public async del(key: K): Promise<void> {
    this.kvs.delete(key)
  }

  public async has(key: K): Promise<boolean> {
    return this.kvs.has(key)
  }

  public async batch(operations: ReadonlyArray<Batch>): Promise<void> {
    for (const op of operations) {
      if (op.type === 'put') {
        this.put(op.key, op.value)
      } else {
        this.del(op.key)
      }
    }
  }

  public iterator(options?: IteratorOptions): EphemDBIterator {
    return new EphemDBIterator(options)
  }

  public bucket(prefix: K): BaseBucket {
    return new BaseBucket(this, prefix)
  }
}

