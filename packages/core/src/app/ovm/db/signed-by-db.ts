import { SignedByDBInterface } from '../../../types/ovm/db/signed-by-db.interface'
import { Message, SignedMessage } from '../../../types/serialization'
import { DB } from '../../../types/db'
import { decryptWithPublicKey, Md5Hash } from '../../utils'
import { SignatureVerifier } from '../../../types/keystore'
import { DefaultSignatureVerifier } from '../../keystore'

interface Record {
  signerPublicKey: Buffer
  signature: Buffer
  message: Buffer
}

/**
 * DB to store and access message signatures.
 */
export class SignedByDB implements SignedByDBInterface {
  public constructor(
    private readonly db: DB,
    private readonly singatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance()
  ) {}

  public async handleMessage(
    message: Message,
    signedMessage?: SignedMessage
  ): Promise<void> {
    if (!!signedMessage) {
      await this.storeSignedMessage(
        signedMessage.signedMessage,
        signedMessage.sender
      )
    }
  }

  public async storeSignedMessage(
    signature: Buffer,
    signerPublicKey: Buffer
  ): Promise<void> {
    // TODO: USE SIGNATURE VERIFIER HERE
    const message: Buffer = decryptWithPublicKey(
      signerPublicKey,
      signature
    ) as Buffer
    const serialized: Buffer = SignedByDB.serializeRecord({
      signerPublicKey,
      signature,
      message,
    })

    await this.db
      .bucket(signerPublicKey)
      .put(SignedByDB.getKey(message), serialized)
  }

  public async getMessageSignature(
    message: Buffer,
    signerPublicKey
  ): Promise<Buffer | undefined> {
    const recordBuffer: Buffer = await this.db
      .bucket(signerPublicKey)
      .get(SignedByDB.getKey(message))

    if (!recordBuffer) {
      return undefined
    }

    return SignedByDB.deserializeRecord(recordBuffer).signature
  }

  public async getAllSignedBy(signerPublicKey: Buffer): Promise<Buffer[]> {
    const signed: Buffer[] = await this.db
      .bucket(signerPublicKey)
      .iterator()
      .values()

    return signed.map((m) => SignedByDB.deserializeRecord(m).message)
  }

  private static getKey(message: Buffer): Buffer {
    return Md5Hash(message)
  }

  private static serializeRecord(record: Record): Buffer {
    return Buffer.from(
      JSON.stringify({
        signerPublicKey: record.signerPublicKey.toString(),
        signature: record.signature.toString(),
        message: record.message.toString(),
      })
    )
  }

  private static deserializeRecord(serialized: Buffer): Record {
    const obj: {} = JSON.parse(serialized.toString())
    return {
      signerPublicKey: Buffer.from(obj['signerPublicKey']),
      signature: Buffer.from(obj['signature']),
      message: Buffer.from(obj['message']),
    }
  }
}
