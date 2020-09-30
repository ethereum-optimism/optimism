/* Internal Imports */
import { ethers } from 'ethers-v4'
import { SignatureProvider, SignatureVerifier } from '../types'
import { hexStrToBuf } from './misc'

export class DefaultSignatureVerifier implements SignatureVerifier {
  private static _instance: SignatureVerifier

  public static instance(): SignatureVerifier {
    if (!DefaultSignatureVerifier._instance) {
      DefaultSignatureVerifier._instance = new DefaultSignatureVerifier()
    }
    return DefaultSignatureVerifier._instance
  }

  public verifyMessage(message: string, signature: string): string {
    // NOTE: we are hashing the message to sign to make the contracts easier (fixed prefix instead of legth prefix).   This should be changed once we support the alternative.
    const messageAsBuf: Buffer = hexStrToBuf(message)
    const messageHash: string = ethers.utils.keccak256(messageAsBuf)
    return ethers.utils.verifyMessage(hexStrToBuf(messageHash), signature)
  }
}

export class DefaultSignatureProvider implements SignatureProvider {
  public constructor(
    private readonly wallet: ethers.Wallet = ethers.Wallet.createRandom()
  ) {}

  public async sign(message: string): Promise<string> {
    // NOTE: we are hashing the message to sign to make the contracts easier (fixed prefix instead of legth prefix).   This should be changed once we support the alternative.
    const messageAsBuf: Buffer = hexStrToBuf(message)
    const messageHash: string = ethers.utils.keccak256(messageAsBuf)
    return this.wallet.signMessage(hexStrToBuf(messageHash))
  }

  public async getAddress(): Promise<string> {
    return this.wallet.getAddress()
  }
}

export class IdentitySigner implements SignatureProvider {
  public constructor(private address: string) {}

  public async getAddress(): Promise<string> {
    return this.address
  }

  public async sign(message: string): Promise<string> {
    return this.address
  }
}

export class IdentityVerifier implements SignatureVerifier {
  private static _instance: SignatureVerifier
  public static instance(): SignatureVerifier {
    if (!IdentityVerifier._instance) {
      IdentityVerifier._instance = new IdentityVerifier()
    }
    return IdentityVerifier._instance
  }
  public verifyMessage(message: string, signature: string): string {
    return signature
  }
}

export class ChecksumAgnosticIdentityVerifier implements SignatureVerifier {
  private static _instance: SignatureVerifier
  public static instance(): SignatureVerifier {
    if (!ChecksumAgnosticIdentityVerifier._instance) {
      ChecksumAgnosticIdentityVerifier._instance = new IdentityVerifier()
    }
    return ChecksumAgnosticIdentityVerifier._instance
  }
  public verifyMessage(message: string, signature: string): string {
    return ethers.utils.getAddress(signature)
  }
}
