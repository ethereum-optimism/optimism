declare module 'bufio' {
  class BufferWriter {
    public offset: number

    constructor()
    render(): Buffer
    getSize(): number
    seek(offset: number): this
    destroy(): this
    writeU8(n: number): this
    writeU16(n: number): this
    writeU16BE(n: number): this
    writeU24(n: number): this
    writeU24BE(n: number): this
    writeU32(n: number): this
    writeU32BE(n: number): this
    writeU40(n: number): this
    writeU40BE(n: number): this
    writeU48(n: number): this
    writeU48BE(n: number): this
    writeU56(n: number): this
    writeU56BE(n: number): this
    writeU64(n: number): this
    writeU64BE(n: number): this
    writeBytes(b: Buffer): this
    copy(value: number, start: number, end: number): this
  }

  class BufferReader {
    constructor(data: Buffer, copy?: boolean)
    getSize(): number
    check(n: number): void
    left(): number
    seek(offset: number): this
    start(): number
    end(): number
    destroy(): this
    readU8(): number
    readU16(): number
    readU16BE(): number
    readU24(): number
    readU24BE(): number
    readU32(): number
    readU32BE(): number
    readU40(): number
    readU40BE(): number
    readU48(): number
    readU48BE(): number
    readU56(): number
    readU56BE(): number
    readU64(): number
    readU64BE(): number
    readBytes(size: number, copy?: boolean): Buffer
  }

  class Struct {
    constructor()
    encode(extra?: object): Buffer
    decode<T extends Struct>(data: Buffer, extra?: object): T
    getSize(extra?: object): number
    fromHex(s: string, extra?: object): this
    toHex(): string
    write(bw: BufferWriter, extra?: object): BufferWriter
    read(br: BufferReader, extra?: object): this
    static read<T extends Struct>(br: BufferReader, extra?: object): T
    static decode<T extends Struct>(data: Buffer, extra?: object): T
    static fromHex<T extends Struct>(s: string, extra?: object): T
  }
}
