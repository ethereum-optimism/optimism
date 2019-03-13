/* Internal Imports */
import { BaseDBProvider, DBObject, DBResult, DBValue } from './base-provider'
import { jsonify, stringify } from '../../../utils'

/**
 * A simple in-memory key-value store.
 */
export class EphemDBProvider implements BaseDBProvider {
  private db = new Map<string, string>()

  public async start(): Promise<void> {
    return
  }

  public async get<T>(key: string, fallback?: T): Promise<T | DBResult> {
    const result = this.db.get(key)
    if (!result) {
      if (fallback !== undefined) {
        return fallback
      } else {
        throw new Error('Key not found in database')
      }
    }

    return jsonify(result)
  }

  public async set(key: string, value: DBValue): Promise<void> {
    const stringified = stringify(value)
    this.db.set(key, stringified)
  }

  public async delete(key: string): Promise<void> {
    this.db.delete(key)
  }

  public async exists(key: string): Promise<boolean> {
    return this.db.has(key)
  }

  public async findNextKey(key: string): Promise<string> {
    const prefix = key.split(':')[0]
    const keys = Array.from(this.db.keys())

    const nextKey = keys
      .filter((k) => {
        return k.startsWith(prefix)
      })
      .sort()
      .find((k) => {
        return k > key
      })

    if (!nextKey) {
      throw new Error('Could not find next key in database.')
    }

    return nextKey
  }

  public async bulkPut(objects: DBObject[]): Promise<void> {
    for (const object of objects) {
      await this.set(object.key, object.value)
    }
  }

  public async push<T>(key: string, value: T): Promise<void> {
    const current = (await this.get(key, [])) as T[]
    current.push(value)
    await this.set(key, current)
  }
}
