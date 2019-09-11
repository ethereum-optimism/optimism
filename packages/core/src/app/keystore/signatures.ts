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
  public constructor(private readonly wallet: ethers.Wallet) {}

  public async sign(_address: string, message: string): Promise<string> {
    return this.wallet.signMessage(message)
  }
}
