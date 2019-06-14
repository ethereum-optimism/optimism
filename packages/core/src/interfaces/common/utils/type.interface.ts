export type Type<T> = new (...args: any[]) => T

export interface AbiEncodable {
  encoded: string
}
