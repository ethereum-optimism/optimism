import { Address, TokenType, Transaction } from '.'
import { SignatureProvider, SignatureVerifier } from '@pigi/core'

/* Utilities */
export const generateTransferTx = (
  recipient: Address,
  tokenType: TokenType,
  amount: number
): Transaction => {
  return {
    tokenType,
    recipient,
    amount,
  }
}

export class IdentitySigner implements SignatureProvider {
  private static _instance: SignatureProvider
  public static instance(): SignatureProvider {
    if (!IdentitySigner._instance) {
      IdentitySigner._instance = new IdentitySigner()
    }
    return IdentitySigner._instance
  }

  public async sign(address: string, message: string): Promise<string> {
    return address
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
