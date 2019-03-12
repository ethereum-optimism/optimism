/**
 * Base class for JSON-RPC subdispatchers that handle requests.
 */
export abstract class BaseSubdispatcher {
  /**
   * Returns the JSON-RPC prefix of this subdispatcher.
   * @returns the prefix.
   */
  public abstract readonly prefix: string

  /**
   * Returns an object with pointers to methods.
   * @return names and pointers to handlers.
   */
  public abstract readonly methods: { [key: string]: (...args: any) => any }

  /**
   * Returns all JSON-RPC methods of this subdispatcher.
   * @returns prefixed names and pointers to handlers.
   */
  public getAllMethods(): { [key: string]: (...args: any) => any } {
    const methods: { [key: string]: (...args: any) => any } = {}
    for (const method of Object.keys(this.methods)) {
      methods[this.prefix + method] = this.methods[method]
    }
    return methods
  }
}
