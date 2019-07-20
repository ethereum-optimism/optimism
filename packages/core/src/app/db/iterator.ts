/**
 * Modified from bcoin's bdb (https://github.com/bcoin-org/bdb) (MIT LICENSE).
 * Credit to the original author, Christopher Jeffrey (https://github.com/chjj).
 */

/* External Imports */
import { AbstractIterator } from 'abstract-leveldown'
import BigNum = require('bn.js')

/* Internal Imports */
import {
  Iterator,
  IteratorOptions,
  K,
  V,
  KV,
  DB,
  RangeEntry,
} from '../../types'

const defaultIteratorOptions: IteratorOptions = {
  reverse: false,
  limit: -1,
  keys: true,
  values: true,
  keyAsBuffer: true,
  valueAsBuffer: true,
  prefix: Buffer.from(''),
}

/**
 * Wrapper for an abstract-leveldown compliant iterator.
 */
export class BaseIterator implements Iterator {
  private readonly options: IteratorOptions
  private readonly prefix: Buffer
  private iterator: AbstractIterator<K, V>
  private finished: boolean

  constructor(readonly db: DB, options: IteratorOptions = {}) {
    this.prefix = options.prefix || defaultIteratorOptions.prefix

    /**
     * Option values for iterators cannot be `null` or `undefined`.
     * We need to be careful not to set these values
     * accidentally or risk an error in the underlying DB.
     */

    if (options.gte !== undefined) {
      options.gte = this.addPrefix(options.gte)
    }
    if (options.lte !== undefined) {
      options.lte = this.addPrefix(options.lte)
    }
    if (options.gt !== undefined) {
      options.gt = this.addPrefix(options.gt)
    }
    if (options.lt !== undefined) {
      options.lt = this.addPrefix(options.lt)
    }

    this.options = {
      ...defaultIteratorOptions,
      ...options,
    }
  }

  /**
   * Seeks to the next key:value pair in the
   * iterator and returns it.
   * @returns the next value in the iterator.
   */
  public async next(): Promise<KV> {
    this.start()
    const { key, value } = await new Promise<KV>((resolve, reject) => {
      this.iterator.next((err, k, v) => {
        if (err) {
          reject(err)
          return
        }
        resolve({ key: k, value: v })
      })
    })

    if (key === undefined && value === undefined) {
      this.cleanup()
    }

    return { key: this.removePrefix(key), value }
  }

  /**
   * Seeks to a target key.
   * @param target Key to seek to.
   */
  public async seek(target: K): Promise<void> {
    this.start()
    this.iterator.seek(this.addPrefix(target))
  }

  /**
   * Executes a function for each key:value pair
   * remaining in the iterator. Starts seeking
   * from the iterator's cursor.
   * @param cb Function to be called for each key:value pair.
   */
  public async each(cb: (key: Buffer, value: Buffer) => any): Promise<void> {
    while (!this.finished) {
      const { key, value } = await this.next()

      if (this.finished) {
        return this.end()
      }

      let result: any
      try {
        result = cb(key, value)

        if (result instanceof Promise) {
          result = await result
        }
      } catch (err) {
        await this.end()
        throw err
      }

      if (result === false) {
        return this.end()
      }
    }

    return this.end()
  }

  /**
   * @returns the items in the iterator.
   */
  public async items(): Promise<KV[]> {
    const items: KV[] = []
    await this.each((key, value) => {
      return items.push({
        key,
        value,
      })
    })
    return items
  }

  /**
   * @returns the keys in the iterator.
   */
  public async keys(): Promise<K[]> {
    const items = await this.items()
    return items.map((item) => {
      return item.key
    })
  }

  /**
   * @returns the values in the iterator.
   */
  public async values(): Promise<V[]> {
    const items = await this.items()
    return items.map((item) => {
      return item.value
    })
  }

  /**
   * Ends iteration and frees up resources.
   */
  public async end(): Promise<void> {
    if (!this.iterator) {
      try {
        this.start()
      } catch (err) {
        throw err
      }
    }

    this.cleanup()
    await new Promise<void>((resolve, reject) => {
      this.iterator.end((err) => {
        if (err) {
          reject(err)
          return
        }
        resolve()
      })
    })
  }

  /**
   * Starts iteration by creating a snapshot.
   */
  private start(): void {
    if (this.iterator !== undefined) {
      return
    }

    this.iterator = this.db.db.iterator(this.options)
  }

  /**
   * Cleans up the iterator.
   */
  private cleanup(): void {
    this.finished = true
  }

  /**
   * Adds a prefix to a value.
   * @param value Value to add the prefix to.
   * @returns the value with the prefix added.
   */
  private addPrefix(value: Buffer): Buffer {
    return value ? Buffer.concat([this.prefix, value]) : value
  }

  /**
   * Removes a prefix from a value.
   * @param value Value to remove prefix from.
   * @returns the value with the prefix removed.
   */
  private removePrefix(value: Buffer): Buffer {
    return value ? value.slice(this.prefix.length) : value
  }
}

/**
 * A special purpose iterator which includes a nextRange() function that returns RangeEntrys instead of simple KVs.
 * This is used by the RangeBucket class.
 */
export class BaseRangeIterator extends BaseIterator {
  /**
   * Constructs a RangeIterator with a particular `resultToRange()` function that will transform
   * the it.next() result into a RangeEntry.
   */
  constructor(
    db: DB,
    options: IteratorOptions = {},
    readonly resultToRange: (result: KV) => RangeEntry
  ) {
    super(db, options)
  }

  /**
   * Advances the iterator to the next key and converts its result into a RangeEntry.
   * @returns the RangeEntry at the next key.
   */
  public async nextRange(): Promise<RangeEntry> {
    const res: KV = await this.next()
    if (typeof res.key === 'undefined') {
      return
    }
    return this.resultToRange(res)
  }
}
