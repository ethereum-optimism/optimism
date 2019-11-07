/* Internal Imports */
import { K } from './db'

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

export interface KeyType {
  min: string | number | Buffer
  max: string | number | Buffer
  dynamic: boolean
  size(value?: any): number
  read(key: K, offset: number): any
  write(key: K, value: any, offset: number): any
}
