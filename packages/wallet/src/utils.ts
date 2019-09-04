import { Address, TokenType, Transaction } from '.'

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
