export type K = NonNullable<Buffer>
export type V = NonNullable<Buffer>

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
 * Bucket represents a basic collection of key:value pairs.
 * Bucket are effectively databases that only perform operations
 * on keys that share a common `prefix`.
 */
export interface Bucket {
  readonly prefix: K

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
  has(key: Buffer): Promise<Boolean>

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
}

export interface DBOptions {
  createIfMissing?: boolean
  errorIfExists?: boolean
}

/**
 * Represents a key:value store.
 */
export interface BaseDB extends Bucket {
  readonly status: 'new' | 'opening' | 'open' | 'closing' | 'closed'

  /**
   * Opens the store.
   * @param [options] Database options.
   */
  open(options?: DBOptions): Promise<void>

  /**
   * Closes the store.
   */
  close(): Promise<void>
}

export interface IteratorOptions {
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
}

/**
 * Iterators traverse over ranges of keys.
 * Iterators operate on a *snapshot* of the store
 * and not on the store itself. As a result,
 * the iterator is not impacted by writes
 * made after the iterator was created.
 */
export interface Iterator {
  db: BaseDB

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
   * Ends iteration and frees resources.
   */
  end(): Promise<void>

  /**
   * Executes a function for each key:value
   * pair in the iterator.
   * @param cb Function to be executed.
   */
  each(cb: (...args: any[]) => any): Promise<void>

  /**
   * @returns all keys in the iterator.
   */
  keys(): Promise<K[]>

  /**
   * @returns all values in the iterator.
   */
  values(): Promise<V[]>
}

/**
 * Keys are formatting helpers for inserting into the database.
 */
export interface Key {
  /**
   * Encode a value based on this key.
   * @param args Arguments to encode.
   * @returns the encoded key.
   */
  encode(...args: any[]): K

  /**
   * Decodes a key into its components.
   * @param key Key to decode.
   * @returns the decoded key as an array.
   */
  decode(key: K): any[]

  /**
   * Returns the minimum value of the key.
   * @param args Parts of the key to set.
   * @returns the minimum value.
   */
  min(...args: any[]): K

  /**
   * Returns the maximum value of the key.
   * @param args Parts of the key to set.
   * @returns the maximum value.
   */
  max(...args: any[]): K
}
