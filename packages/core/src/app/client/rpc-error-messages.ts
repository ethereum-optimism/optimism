export interface RpcErrorMessage {
  code: number
  message: string
}

export const RPC_ERROR_MESSAGES = {
  /* Wallet */
  ACCOUNT_NOT_FOUND: {
    code: -20001,
    message: 'Account Not Found',
  },
  INVALID_PASSWORD: {
    code: -20002,
    message: 'Invalid Password',
  },
  ACCOUNT_LOCKED: {
    code: -20003,
    message: 'Account Locked',
  },

  /* Transactions */
  INVALID_TRANSACTION_ENCODING: {
    code: -20004,
    message: 'Invalid Transaction Encoding',
  },
  INVALID_TRANSACTION: {
    code: -20005,
    message: 'Invalid Transaction',
  },

  /* State Queries */
  INVALID_STATE_QUERY: {
    code: -20006,
    message: 'Invalid State Query',
  },
}
