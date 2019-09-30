import { SignatureProvider, SignatureVerifier } from '../../types/keystore'
import { ethers } from 'ethers'

export class DefaultSignatureVerifier implements SignatureVerifier {
  private static _instance: SignatureVerifier

  public static instance(): SignatureVerifier {
    if (!DefaultSignatureVerifier._instance) {
      DefaultSignatureVerifier._instance = new DefaultSignatureVerifier()
    }
    return DefaultSignatureVerifier._instance
  }

  public verifyMessage(message: string, signature: string): string {
    return ethers.utils.verifyMessage(message, signature)
  }
}

export class DefaultSignatureProvider implements SignatureProvider {
  public constructor(
    private readonly wallet: ethers.Wallet = ethers.Wallet.createRandom()
  ) {}

  public async sign(message: string): Promise<string> {
    return this.wallet.signMessage(message)
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
    return ethers.utils.getAddress(signature)
  }
}
