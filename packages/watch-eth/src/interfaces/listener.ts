/**
 * Generic listener for Ethereum events and objects
 */
export interface EthereumListener<T> {
  handle(t: T): Promise<void>
}
