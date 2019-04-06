import uuid = require('uuid')

/**
 * Gets names of all methods on an object.
 * @param obj Object to check.
 * @returns all method names on that object.
 */
const getAllMethodNames = (obj: any): string[] => {
  const methods = new Set()

  // tslint:disable-next-line
  while ((obj = Reflect.getPrototypeOf(obj))) {
    const keys = Reflect.ownKeys(obj)
    keys.forEach((k) => methods.add(k))
  }

  return Array.from(methods)
}

/**
 * Gets all method and property names on an object.
 * @param obj Object to check.
 * @returns all method and property names on that object.
 */
const getAllMethodAndPropertyNames = (obj: any): string[] => {
  const methods = getAllMethodNames(obj)
  const properties = Object.getOwnPropertyNames(obj)
  return Array.from(new Set(methods.concat(properties)))
}

/**
 * Represents a basic process with start/stop functionality.
 */
export class Process {
  private ready = false
  public readonly pid = uuid.v4()

  /**
   * @returns `true` if the process is ready, `false` otherwise.
   */
  public isReady(): boolean {
    return this.ready
  }

  /**
   * Starts the process.
   */
  public async start(): Promise<void> {
    if (this.ready) {
      return
    }

    await this.onStart()
    this.ready = true
  }

  /**
   * Stops the process.
   */
  public async stop(): Promise<void> {
    if (!this.ready) {
      return
    }

    await this.onStop()
    this.ready = false
  }

  /**
   * Runs when the process is started.
   */
  protected async onStart(): Promise<void> {
    return
  }

  /**
   * Runs when the process is stopped.
   */
  protected async onStop(): Promise<void> {
    return
  }

  /**
   * Asserts that the process is ready and
   * throws otherwise.
   */
  protected assertReady(): void {
    if (!this.isReady()) {
      throw new Error('Process is not ready.')
    }
  }
}

/**
 * Process that proxies a class.
 */
export class ProxyProcess<TBase> extends Process {
  protected instance: TBase = {} as any

  constructor() {
    super()

    /**
     * Checks if a specific property should be accessible
     * externally before the underlying object is ready.
     * @param prop Property to check.
     * @returns `true` if the property is accessible, `false` otherwise.
     */
    const isAccessible = (prop: any): boolean => {
      return getAllMethodAndPropertyNames(this).includes(prop)
    }

    return new Proxy(this.instance as any, {
      get: (_, prop) => {
        if (isAccessible(prop)) {
          return this[prop]
        }
        this.assertReady()
        return this.instance[prop]
      },
      set: (_, prop, value): boolean => {
        if (isAccessible(prop)) {
          this[prop] = value
          return true
        }
        this.assertReady()
        this.instance[prop] = value
        return true
      },
    })
  }
}
