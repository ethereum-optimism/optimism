import { KeyType } from '../../../../interfaces'
import {
  assertType,
  assertLen,
  sizeBuffer,
  readBuffer,
  writeBuffer,
  sizeString,
  readString,
  writeString,
  sizeHex,
  writeHex,
  BUFFER_MIN,
  BUFFER_MAX,
} from './utils'

// TODO: Add bcoin acknowledgements for this code!

/* tslint:disable:no-bitwise */
export const types = {
  char: {
    min: '\x00',
    max: '\xff',
    dynamic: false,
    size() {
      return 1
    },
    read(k, o) {
      assertLen(o + 1 <= k.length)
      return String.fromCharCode(k[o])
    },
    write(k, v, o) {
      assertType(typeof v === 'string')
      assertType(v.length === 1)
      assertLen(o + 1 <= k.length)
      k[o] = v.charCodeAt(0)
      return 1
    },
  },
  uint8: {
    min: 0,
    max: 0xff,
    dynamic: false,
    size() {
      return 1
    },
    read(k, o) {
      assertLen(o + 1 <= k.length)
      return k[o]
    },
    write(k, v, o) {
      assertType((v & 0xff) === v)
      assertLen(o + 1 <= k.length)
      k[o] = v
      return 1
    },
  },
  uint16: {
    min: 0,
    max: 0xffff,
    dynamic: false,
    size() {
      return 2
    },
    read(k, o) {
      assertLen(o + 2 <= k.length)
      return k.readUInt16BE(o)
    },
    write(k, v, o) {
      assertType((v & 0xffff) === v)
      assertLen(o + 2 <= k.length)
      k.writeUInt16BE(v, o)
      return 2
    },
  },
  uint32: {
    min: 0,
    max: 0xffffffff,
    dynamic: false,
    size() {
      return 4
    },
    read(k, o) {
      assertLen(o + 4 <= k.length)
      return k.readUInt32BE(o)
    },
    write(k, v, o) {
      assertType(v >>> 0 === v)
      assertLen(o + 4 <= k.length)
      k.writeUInt32BE(v, o)
      return 4
    },
  },
  buffer: {
    min: BUFFER_MIN,
    max: BUFFER_MAX,
    dynamic: true,
    size(v) {
      return sizeBuffer(v)
    },
    read(k, o) {
      return readBuffer(k, o)
    },
    write(k, v, o) {
      return writeBuffer(k, v, o)
    },
  },
  hex: {
    min: BUFFER_MIN.toString('hex'),
    max: BUFFER_MAX.toString('hex'),
    dynamic: true,
    size(v) {
      return sizeString(v, 'hex')
    },
    read(k, o) {
      return readString(k, o, 'hex')
    },
    write(k, v, o) {
      return writeString(k, v, o, 'hex')
    },
  },
  ascii: {
    min: BUFFER_MIN.toString('binary'),
    max: BUFFER_MAX.toString('binary'),
    dynamic: true,
    size(v) {
      return sizeString(v, 'binary')
    },
    read(k, o) {
      return readString(k, o, 'binary')
    },
    write(k, v, o) {
      return writeString(k, v, o, 'binary')
    },
  },
  utf8: {
    min: BUFFER_MIN.toString('utf8'),
    max: BUFFER_MAX.toString('utf8'),
    dynamic: true,
    size(v) {
      return sizeString(v, 'utf8')
    },
    read(k, o) {
      return readString(k, o, 'utf8')
    },
    write(k, v, o) {
      return writeString(k, v, o, 'utf8')
    },
  },
  hash160: {
    min: Buffer.alloc(20, 0x00),
    max: Buffer.alloc(20, 0xff),
    dynamic: false,
    size() {
      return 20
    },
    read(k, o) {
      assertLen(o + 20 <= k.length)
      return k.slice(o, o + 20)
    },
    write(k, v, o) {
      assertType(Buffer.isBuffer(v))
      assertType(v.copy(k, o) === 20)
      return 20
    },
  },
  hash256: {
    min: Buffer.alloc(32, 0x00),
    max: Buffer.alloc(32, 0xff),
    dynamic: false,
    size() {
      return 32
    },
    read(k, o) {
      assertLen(o + 32 <= k.length)
      return k.slice(o, o + 32)
    },
    write(k, v, o) {
      assertType(Buffer.isBuffer(v))
      assertType(v.copy(k, o) === 32)
      return 32
    },
  },
  hash: {
    min: Buffer.alloc(1, 0x00),
    max: Buffer.alloc(64, 0xff),
    dynamic: true,
    size(v) {
      assertType(Buffer.isBuffer(v))
      return 1 + v.length
    },
    read(k, o) {
      assertLen(o + 1 <= k.length)
      assertLen(k[o] >= 1 && k[o] <= 64)
      assertLen(o + 1 + k[o] <= k.length)
      return k.slice(o + 1, o + 1 + k[o])
    },
    write(k, v, o) {
      assertType(Buffer.isBuffer(v))
      assertType(v.length >= 1 && v.length <= 64)
      assertLen(o + 1 <= k.length)

      k[o] = v.length

      assertType(v.copy(k, o + 1) === v.length)

      return 1 + v.length
    },
  },
  hhash160: {
    min: Buffer.alloc(20, 0x00),
    max: Buffer.alloc(20, 0xff),
    dynamic: false,
    size() {
      return 20
    },
    read(k, o) {
      assertLen(o + 20 <= k.length)
      return k.toString('hex', o, o + 20)
    },
    write(k, v, o) {
      assertType(writeHex(k, v, o) === 20)
      return 20
    },
  },
  hhash256: {
    min: Buffer.alloc(32, 0x00),
    max: Buffer.alloc(32, 0xff),
    dynamic: false,
    size() {
      return 32
    },
    read(k, o) {
      assertLen(o + 32 <= k.length)
      return k.toString('hex', o, o + 32)
    },
    write(k, v, o) {
      assertType(writeHex(k, v, o) === 32)
      return 32
    },
  },
  hhash: {
    min: Buffer.alloc(1, 0x00),
    max: Buffer.alloc(64, 0xff),
    dynamic: true,
    size(v) {
      return 1 + sizeHex(v)
    },
    read(k, o) {
      assertLen(o + 1 <= k.length)
      assertLen(k[o] >= 1 && k[o] <= 64)
      assertLen(o + 1 + k[o] <= k.length)
      return k.toString('hex', o + 1, o + 1 + k[o])
    },
    write(k, v, o) {
      const size = sizeHex(v)

      assertType(size >= 1 && size <= 64)
      assertLen(o + 1 <= k.length)

      k[o] = size

      assertType(writeHex(k, v, o + 1) === size)

      return 1 + size
    },
  },
}

/* tslint:enable:no-bitwise */
