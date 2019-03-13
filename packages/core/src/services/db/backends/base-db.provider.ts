export type DBValue = string | object | number | boolean

export type DBResult = DBValue | DBValue[]

export interface DBObject {
  key: string
  value: DBValue
}

export interface DBOptions {
  [key: string]: any

  namespace: string
  id?: string
}

export interface BaseDBProvider {
  /**
   * Returns the value stored at the given key.
   * @param key Key to query.
   * @param fallback A fallback value if the key doesn't exist.
   * @returns the stored value or the fallback.
   */
  get<T>(key: string, fallback?: T): Promise<T | DBResult>

  /**
   * Sets a given key with the value.
   * @param key Key to set.
   * @param value Value to store.
   */
  set(key: string, value: DBValue): Promise<void>

  /**
   * Deletes a given key from storage.
   * @param key Key to delete.
   */
  delete(key: string): Promise<void>

  /**
   * Checks if a key exists in storage.
   * @param key Key to check.
   * @returns `true` if the key exists, `false` otherwise.
   */
  exists(key: string): Promise<boolean>

  /**
   * Finds the next key after a given key.
   * @param key The key to start searching from.
   * @returns the next key with the same prefix.
   */
  findNextKey(key: string): Promise<string>

  /**
   * Puts a series of objects into the database in bulk.
   * Should be more efficient than simply calling `set` repeatedly.
   * @param objects A series of objects to put into the database.
   */
  bulkPut(objects: DBObject[]): Promise<void>

  /**
   * Pushes to an array stored at a key in the database.
   * @param key The key at which the array is stored.
   * @param value Value to add to the array.
   */
  push<T>(key: string, value: T): Promise<void>
}
