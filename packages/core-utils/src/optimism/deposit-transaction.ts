import { getAddress } from '@ethersproject/address'
import { ContractReceipt, Event } from '@ethersproject/contracts'
import { BigNumber, BigNumberish } from '@ethersproject/bignumber'
import { keccak256 } from '@ethersproject/keccak256'
import { Zero } from '@ethersproject/constants'
import * as RLP from '@ethersproject/rlp'
import {
  arrayify,
  BytesLike,
  hexDataSlice,
  stripZeros,
  hexConcat,
  zeroPad,
} from '@ethersproject/bytes'

const formatBoolean = (value: boolean): Uint8Array => {
  return value ? new Uint8Array([1]) : new Uint8Array([])
}

const formatNumber = (value: BigNumberish, name: string): Uint8Array => {
  const result = stripZeros(BigNumber.from(value).toHexString())
  if (result.length > 32) {
    throw new Error(`invalid length for ${name}`)
  }
  return result
}

const handleBoolean = (value: string): boolean => {
  if (value === '0x') {
    return false
  }
  if (value === '0x01') {
    return true
  }
  throw new Error(`invalid boolean RLP hex value ${value}`)
}

const handleNumber = (value: string): BigNumber => {
  if (value === '0x') {
    return Zero
  }
  return BigNumber.from(value)
}

const handleAddress = (value: string): string => {
  if (value === '0x') {
    // @ts-ignore
    return null
  }
  return getAddress(value)
}

export enum SourceHashDomain {
  UserDeposit = 0,
  L1InfoDeposit = 1,
}

interface DepositTxOpts {
  sourceHash?: string
  from: string
  to: string | null
  mint: BigNumberish
  value: BigNumberish
  gas: BigNumberish
  isSystemTransaction: boolean
  data: string
  domain?: SourceHashDomain
  l1BlockHash?: string
  logIndex?: BigNumberish
  sequenceNumber?: BigNumberish
}

interface DepositTxExtraOpts {
  domain?: SourceHashDomain
  l1BlockHash?: string
  logIndex?: BigNumberish
  sequenceNumber?: BigNumberish
}

export class DepositTx {
  public type = 0x7e
  public version = 0x00
  private _sourceHash?: string
  public from: string
  public to: string | null
  public mint: BigNumberish
  public value: BigNumberish
  public gas: BigNumberish
  public isSystemTransaction: boolean
  public data: BigNumberish

  public domain?: SourceHashDomain
  public l1BlockHash?: string
  public logIndex?: BigNumberish
  public sequenceNumber?: BigNumberish

  constructor(opts: Partial<DepositTxOpts> = {}) {
    this._sourceHash = opts.sourceHash
    this.from = opts.from!
    this.to = opts.to!
    this.mint = opts.mint!
    this.value = opts.value!
    this.gas = opts.gas!
    this.isSystemTransaction = opts.isSystemTransaction || false
    this.data = opts.data!
    this.domain = opts.domain
    this.l1BlockHash = opts.l1BlockHash
    this.logIndex = opts.logIndex
    this.sequenceNumber = opts.sequenceNumber
  }

  hash() {
    const encoded = this.encode()
    return keccak256(encoded)
  }

  sourceHash() {
    if (!this._sourceHash) {
      let marker: string
      switch (this.domain) {
        case SourceHashDomain.UserDeposit:
          marker = BigNumber.from(this.logIndex).toHexString()
          break
        case SourceHashDomain.L1InfoDeposit:
          marker = BigNumber.from(this.sequenceNumber).toHexString()
          break
        default:
          throw new Error(`Unknown domain: ${this.domain}`)
      }

      if (!this.l1BlockHash) {
        throw new Error('Need l1BlockHash to compute sourceHash')
      }

      const l1BlockHash = this.l1BlockHash
      const input = hexConcat([l1BlockHash, zeroPad(marker, 32)])
      const depositIDHash = keccak256(input)
      const domain = BigNumber.from(this.domain).toHexString()
      const domainInput = hexConcat([zeroPad(domain, 32), depositIDHash])
      this._sourceHash = keccak256(domainInput)
    }
    return this._sourceHash
  }

  encode() {
    const fields: any = [
      this.sourceHash() || '0x',
      getAddress(this.from) || '0x',
      this.to != null ? getAddress(this.to) : '0x',
      formatNumber(this.mint || 0, 'mint'),
      formatNumber(this.value || 0, 'value'),
      formatNumber(this.gas || 0, 'gas'),
      formatBoolean(this.isSystemTransaction),
      this.data || '0x',
    ]

    return hexConcat([
      BigNumber.from(this.type).toHexString(),
      RLP.encode(fields),
    ])
  }

  decode(raw: BytesLike, extra: DepositTxExtraOpts = {}) {
    const payload = arrayify(raw)
    if (payload[0] !== this.type) {
      throw new Error(`Invalid type ${payload[0]}`)
    }
    this.version = payload[1]
    const transaction = RLP.decode(payload.slice(1))
    this._sourceHash = transaction[0]
    this.from = handleAddress(transaction[1])
    this.to = handleAddress(transaction[2])
    this.mint = handleNumber(transaction[3])
    this.value = handleNumber(transaction[4])
    this.gas = handleNumber(transaction[5])
    this.isSystemTransaction = handleBoolean(transaction[6])
    this.data = transaction[7]

    if ('l1BlockHash' in extra) {
      this.l1BlockHash = extra.l1BlockHash
    }
    if ('domain' in extra) {
      this.domain = extra.domain
    }
    if ('logIndex' in extra) {
      this.logIndex = extra.logIndex
    }
    if ('sequenceNumber' in extra) {
      this.sequenceNumber = extra.sequenceNumber
    }
    return this
  }

  static decode(raw: BytesLike, extra?: DepositTxExtraOpts): DepositTx {
    return new this().decode(raw, extra)
  }

  fromL1Receipt(receipt: ContractReceipt, index: number): DepositTx {
    if (!receipt.events) {
      throw new Error('cannot parse receipt')
    }
    const event = receipt.events[index]
    if (!event) {
      throw new Error(`event index ${index} does not exist`)
    }
    return this.fromL1Event(event)
  }

  static fromL1Receipt(receipt: ContractReceipt, index: number): DepositTx {
    return new this({}).fromL1Receipt(receipt, index)
  }

  fromL1Event(event: Event): DepositTx {
    if (event.event !== 'TransactionDeposited') {
      throw new Error(`incorrect event type: ${event.event}`)
    }
    if (typeof event.args === 'undefined') {
      throw new Error('no event args')
    }
    if (typeof event.args.from === 'undefined') {
      throw new Error('"from" undefined')
    }
    this.from = event.args.from
    if (typeof event.args.to === 'undefined') {
      throw new Error('"to" undefined')
    }
    if (typeof event.args.version === 'undefined') {
      throw new Error(`"verison" undefined`)
    }
    if (!event.args.version.eq(0)) {
      throw new Error(`Unsupported version ${event.args.version.toString()}`)
    }
    if (typeof event.args.opaqueData === 'undefined') {
      throw new Error(`"opaqueData" undefined`)
    }

    const opaqueData = event.args.opaqueData
    if (opaqueData.length < 32 + 32 + 8 + 1) {
      throw new Error(`invalid opaqueData size: ${opaqueData.length}`)
    }

    let offset = 0
    this.mint = BigNumber.from(hexDataSlice(opaqueData, offset, offset + 32))
    offset += 32
    this.value = BigNumber.from(hexDataSlice(opaqueData, offset, offset + 32))
    offset += 32
    this.gas = BigNumber.from(hexDataSlice(opaqueData, offset, offset + 8))
    offset += 8
    const isCreation = BigNumber.from(opaqueData[offset]).eq(1)
    offset += 1
    this.to = isCreation === true ? null : event.args.to
    const length = opaqueData.length - offset
    this.isSystemTransaction = false
    this.data = hexDataSlice(opaqueData, offset, offset + length)
    this.domain = SourceHashDomain.UserDeposit
    this.l1BlockHash = event.blockHash
    this.logIndex = event.logIndex
    return this
  }

  static fromL1Event(event: Event): DepositTx {
    return new this({}).fromL1Event(event)
  }
}
