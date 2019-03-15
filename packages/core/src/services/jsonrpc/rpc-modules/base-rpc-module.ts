/**
 * Gets the names of methods on an object.
 * Returns everything except for the constructor.
 * @param obj Object to get methods of.
 * @returns the methods on that object.
 */
const getMethodNames = (obj: any): string[] => {
  return Object.getOwnPropertyNames(obj.prototype).filter(
    (method) => method !== 'constructor'
  )
}

/**
 * Base class for JSON-RPC subdispatchers that handle requests.
 */
export abstract class BaseRpcModule {
  /**
   * Returns the JSON-RPC prefix of this subdispatcher.
   * @returns the prefix.
   */
  public abstract readonly prefix: string

  /**
   * Returns all JSON-RPC methods of this subdispatcher.
   * @returns prefixed names and pointers to handlers.
   */
  public getAllMethods(): { [key: string]: (...args: any) => any } {
    const methods: { [key: string]: (...args: any) => any } = {}
    for (const method of getMethodNames(this)) {
      methods[this.prefix + method] = this[method].bind(this)
    }
    return methods
  }
}
