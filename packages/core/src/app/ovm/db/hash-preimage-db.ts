import { HashPreimageDBInterface } from '../../../types/ovm/db'
import { HashAlgorithm, Logger } from '../../../types/utils'
import { DB } from '../../../types/db'
import { getLogger, hashFunctionFor } from '../../utils'
import { Message } from '../../../types/serialization'
import { deserializeObject } from '../../serialization'

const log: Logger = getLogger('hash-preimage-db')

interface Record {
  preimage: string
  hashAlgorithm: HashAlgorithm
  hash: string
}

/*
 * DB to store and access hashes and their associated preimages.
 */
export class HashPreimageDB implements HashPreimageDBInterface {
  public constructor(private readonly db: DB) {}

  public async handleMessage(serializedMessage: string): Promise<void> {
    try {
      const message: Message = deserializeObject(serializedMessage) as Message
      if (message.data && 'preimage' in message.data) {
        await this.storePreimage(
          message.data['preimage'],
          HashAlgorithm.KECCAK256
        )
      }
    } catch (e) {
      log.debug(
        `Received a message that cannot be parsed. Ignoring. Message: ${serializedMessage}, error: ${e.message}, stack: ${e.stack}`
      )
    }
  }

  public async storePreimage(
    preimage: string,
    hashAlgorithm: HashAlgorithm
  ): Promise<void> {
    const hash: string = hashFunctionFor(hashAlgorithm)(preimage)

    const serialized: Buffer = HashPreimageDB.serializeRecord({
      preimage,
      hashAlgorithm,
      hash,
    })

    await this.db
      .bucket(Buffer.from(hashAlgorithm))
      .put(Buffer.from(hash), serialized)
  }

  public async getPreimage(
    hash: string,
    hashAlgorithm: HashAlgorithm
  ): Promise<string | undefined> {
    const recordBuffer: Buffer = await this.db
      .bucket(Buffer.from(hashAlgorithm))
      .get(Buffer.from(hash))

    if (!recordBuffer) {
      return undefined
    }

    return HashPreimageDB.deserializeRecord(recordBuffer).preimage
  }

  private static serializeRecord(record: Record): Buffer {
    return Buffer.from(
      JSON.stringify({
        preimage: record.preimage.toString(),
        hashAlgorithm: record.hashAlgorithm,
        hash: record.hash.toString(),
      })
    )
  }

  private static deserializeRecord(serialized: Buffer): Record {
    const obj: {} = JSON.parse(serialized.toString())
    return {
      preimage: obj['preimage'],
      hashAlgorithm: obj['hashAlgorithm'],
      hash: obj['hash'],
    }
  }
}
