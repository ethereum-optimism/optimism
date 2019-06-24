/**
 * Modified from bcoin's bdb (https://github.com/bcoin-org/bdb) (MIT LICENSE).
 * Credit to the original author, Christopher Jeffrey (https://github.com/chjj).
 */

/* Internal Imports */
import { Key, KeyType } from '../../../types'
import { types } from './types'
import { makeID, assert } from './utils'

type KeyTypeName = keyof typeof types

/**
 * Simple key implementation.
 */
export class BaseKey implements Key {
  private id: number
  private ops: KeyType[] = []
  private size = 0
  private index = -1

  constructor(id: string | number, ops: KeyTypeName[] = []) {
    this.id = makeID(id)

    for (let i = 0; i < ops.length; i++) {
      const name = ops[i]

      if (!(name in types)) {
        throw new Error(`Invalid type name: ${name}.`)
      }

      const op = types[name]

      if (op.dynamic) {
        if (this.index === -1) {
          this.index = i
        }
      } else {
        this.size += (op as any).size()
      }

      this.ops.push(op)
    }
  }

  /**
   * Gets the size of a key.
   * @param args Arguments to the key.
   * @returns the size of the key.
   */
  public getSize(args: any[]): number {
    assert(args.length === this.ops.length)

    let size = 1 + this.size

    if (this.index === -1) {
      return size
    }

    for (let i = this.index; i < args.length; i++) {
      const op = this.ops[i]
      const arg = args[i]
      if (op.dynamic) {
        size += op.size(arg)
      }
    }

    return size
  }

  /**
   * Encodes a key.
   * @param args Arguments to encode.
   * @returns the encoded key.
   */
  public encode(args: any[] = []): Buffer {
    assert(args.length === this.ops.length)

    const size = this.getSize(args)
    const key = Buffer.allocUnsafe(size)

    key[0] = this.id

    let offset = 1

    for (let i = 0; i < this.ops.length; i++) {
      const op = this.ops[i]
      const arg = args[i]
      offset += op.write(key, arg, offset)
    }

    return key
  }

  /**
   * Decodes a key to its component arguments.
   * @param key Key to decode.
   * @returns the components.
   */
  public decode(key: Buffer): any | any[] {
    if (this.ops.length === 0) {
      return key
    }

    if (key.length === 0 || key[0] !== this.id) {
      throw new Error('Key prefix mismatch.')
    }

    const args = []

    let offset = 1

    for (const op of this.ops) {
      const arg = op.read(key, offset)
      offset += op.size(arg)
      args.push(arg)
    }

    return args
  }

  /**
   * Returns the minimum value for some key.
   * @param args Arguments to the key.
   * @returns the minimum value for that key.
   */
  public min(args: any[] = []): Buffer {
    for (let i = args.length; i < this.ops.length; i++) {
      const op = this.ops[i]
      args.push(op.min)
    }
    return this.encode(args)
  }

  /**
   * Returns the maximum value for some key.
   * @param args Arguments to the key.
   * @returns the maximum value for that key.
   */
  public max(args: any[] = []): Buffer {
    for (let i = args.length; i < this.ops.length; i++) {
      const op = this.ops[i]
      args.push(op.max)
    }
    return this.encode(args)
  }
}
