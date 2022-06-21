import { BigNumber, BigNumberish } from 'ethers'
import { Level } from 'level'

import { ErrEntryInconsistency } from '../errors'

export type IndexLike = BigNumberish | 'latest'

export const makeKey = (
  key: number,
  index: BigNumberish | 'latest'
): string => {
  if (key < 0 || key >= 2 ** 8) {
    throw new Error(`key must be uint8: ${key}`)
  }

  if (index === 'latest') {
    return `${key.toString(16).padStart(2, '0')}:0:latest`
  } else {
    index = BigNumber.from(index).toNumber()
    if (index < 0 || index >= 2 ** 128) {
      throw new Error(`index must be uint128: ${index}`)
    }
    return `${key.toString(16).padStart(2, '0')}:1:${index
      .toString(16)
      .padStart(32, '0')}`
  }
}

export class SimpleDB {
  constructor(private db: Level) {}

  public async get(key: number, index: IndexLike): Promise<any> {
    try {
      return JSON.parse(await this.db.get(makeKey(key, index)))
    } catch (err) {
      return null
    }
  }

  public async put(key: number, index: IndexLike, value: any): Promise<void> {
    const batch = []
    if (index === 'latest') {
      batch.push({
        type: 'put',
        key: makeKey(key, 'latest'),
        value: JSON.stringify(value),
      })
    } else {
      index = BigNumber.from(index).toNumber()
      const latest = await this.get(key, 'latest')

      if (index > 0 && (latest === null || latest.index < index - 1)) {
        throw ErrEntryInconsistency
      }

      batch.push({
        type: 'put',
        key: makeKey(key, index),
        value: JSON.stringify(value),
      })

      if (latest === null || index > latest.index) {
        batch.push({
          type: 'put',
          key: makeKey(key, 'latest'),
          value: JSON.stringify(value),
        })
      }
    }

    await this.db.batch(batch)
  }
}
