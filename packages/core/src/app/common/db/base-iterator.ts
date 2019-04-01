import { Iterator, IteratorOptions, BaseDB, K, V } from '../../../interfaces'

const defaultIteratorOptions: IteratorOptions = {
  reverse: false,
  limit: -1,
  keys: true,
  values: true,
  keyAsBuffer: true,
  valueAsBuffer: true,
  prefix: new Buffer(''),
}

export class BaseIterator implements Iterator {
  private readonly options: IteratorOptions
  private iterator: Iterator
  private prefix: Buffer
  private finished: boolean

  constructor(readonly db: BaseDB, options: IteratorOptions) {
    this.prefix = options.prefix

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

  public async next(): Promise<{ key: K; value: V }> {
    const { key, value } = await this.iterator.next()

    if (key === undefined && value === undefined) {
      this.cleanup()
    }

    return { key: this.removePrefix(key), value }
  }

  public async seek(target: K): Promise<void> {
    this.start()
    this.iterator.seek(this.addPrefix(target))
  }

  public async each(cb: (key: Buffer, value: Buffer) => any): Promise<void> {
    while (!this.finished) {
      const { key, value } = await this.next()

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

  public async keys(): Promise<K[]> {
    const items: Buffer[] = []
    await this.each((key, _) => {
      return items.push(key)
    })
    return items
  }

  public async values(): Promise<V[]> {
    const items: Buffer[] = []
    await this.each((_, value) => {
      return items.push(value)
    })
    return items
  }

  public async end(): Promise<void> {
    if (!this.iterator) {
      try {
        this.start()
      } catch (err) {
        throw err
      }
    }

    this.cleanup()
    await this.iterator.end()
  }

  private start(): void {
    if (this.iterator !== undefined) {
      return
    }

    this.iterator = this.db.iterator(this.options)
  }

  private cleanup(): void {
    this.finished = true
  }

  private addPrefix(value: Buffer): Buffer {
    return value ? Buffer.concat([this.prefix, value]) : value
  }

  private removePrefix(value: Buffer): Buffer {
    return value ? value.slice(this.prefix.length) : value
  }
}
