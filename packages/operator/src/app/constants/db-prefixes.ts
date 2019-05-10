// State Manager
export const COIN_ID_PREFIX = Buffer.from([128])
export const ADDRESS_PREFIX = Buffer.from([127])
export const DEPOSIT_PREFIX = Buffer.from([126])

// Block Manager
export const BLOCK_TX_PREFIX = Buffer.from([255])
export const BLOCK_DEPOSIT_PREFIX = Buffer.from([254])
export const BLOCK_INDEX_PREFIX = Buffer.from([253])
export const BLOCK_ROOT_HASH_PREFIX = Buffer.from([252])
export const NUM_LEVELS_PREFIX = Buffer.from([251])
export const NODE_DB_PREFIX = Buffer.from([250])
export const BLOCK_NUM_TXS_PREFIX = Buffer.from([249])
export const BLOCK_TIMESTAMP_PREFIX = Buffer.from([248])
export const HASH_TO_TX_PREFIX = Buffer.from([247])
