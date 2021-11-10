/* Imports: External */
import { LevelUp } from 'levelup'
import { BigNumber } from 'ethers'

export class SimpleDB {
  constructor(public db: LevelUp) {}

  public async get<TEntry>(key: string, index: number): Promise<TEntry | null> {
    try {
      // TODO: Better checks here.
      return JSON.parse(await this.db.get(this._makeKey(key, index)))
    } catch (err) {
      return null
    }
  }

  public async range<TEntry>(
    key: string,
    startIndex: number,
    endIndex: number
  ): Promise<TEntry[] | []> {
    try {
      return new Promise<any[]>((resolve) => {
        const entries: any[] = []
        this.db
          .createValueStream({
            gte: this._makeKey(key, startIndex),
            lt: this._makeKey(key, endIndex),
          })
          .on('data', (transaction: string) => {
            entries.push(JSON.parse(transaction))
          })
          .on('error', () => {
            resolve(null)
          })
          .on('close', () => {
            // TODO: Close vs end? Need to double check later.
            resolve(entries)
          })
          .on('end', () => {
            resolve(entries)
          })
      })
    } catch (err) {
      return []
    }
  }

  public async put<TEntry>(
    entries: {
      key: string
      index: number
      value: TEntry
    }[]
  ): Promise<void> {
    return this.db.batch(
      entries.map((entry) => {
        return {
          type: 'put',
          key: this._makeKey(entry.key, entry.index),
          value: JSON.stringify(entry.value),
        }
      })
    )
  }

  private _makeKey(key: string, index: number): string {
    // prettier-ignore
    return `${key}:${BigNumber.from(index).toString().padStart(32, '0')}`
  }
}
