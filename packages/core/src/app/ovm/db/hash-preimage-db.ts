import { HashPreimageDbInterface } from '../../../types/ovm/db'
import { HashAlgorithm } from '../../../types/utils'
import { DB } from '../../../types/db'
import { hashFunctionFor } from '../../utils'
import { Message } from '../../../types/serialization'

interface Record {
  preimage: Buffer
  hashAlgorithm: HashAlgorithm
  hash: Buffer
}

/**
 * DB to store and access hashes and their associated preimages.
 */
export class HashPreimageDb implements HashPreimageDbInterface {
  public constructor(private readonly db: DB) {}

  public async handleMessage(message: Message): Promise<void> {
    // TODO: handle each specific type of message when we formally define different messages.
    if (message.data && 'preimage' in message.data) {
      await this.storePreimage(
        Buffer.from(message.data['preimage']),
        HashAlgorithm.KECCAK256
      )
    }
  }

  public async storePreimage(
    preimage: Buffer,
    hashAlgorithm: HashAlgorithm
  ): Promise<void> {
    const hash: Buffer = hashFunctionFor(hashAlgorithm)(preimage)

    const serialized: Buffer = HashPreimageDb.serializeRecord({
      preimage,
      hashAlgorithm,
      hash,
    })

    await this.db.bucket(Buffer.from(hashAlgorithm)).put(hash, serialized)
  }

  public async getPreimage(
    hash: Buffer,
    hashAlgorithm: HashAlgorithm
  ): Promise<Buffer | undefined> {
    const recordBuffer: Buffer = await this.db
      .bucket(Buffer.from(hashAlgorithm))
      .get(hash)

    if (!recordBuffer) {
      return undefined
    }

    return HashPreimageDb.deserializeRecord(recordBuffer).preimage
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
      preimage: Buffer.from(obj['preimage']),
      hashAlgorithm: obj['hashAlgorithm'],
      hash: Buffer.from(obj['hash']),
    }
  }
}
