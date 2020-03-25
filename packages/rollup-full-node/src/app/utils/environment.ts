import { DEFAULT_OPCODE_WHITELIST_MASK } from '@eth-optimism/ovm'

/**
 * Class to contain all environment variables referenced by the rollup full node
 * to consolidate access / updates and default values.
 */
export class Environment {
  // Local Node Config
  public static opcodeWhitelistMask(
    defaultValue: string = DEFAULT_OPCODE_WHITELIST_MASK
  ): string {
    return process.env.OPCODE_WHITELIST_MASK || defaultValue
  }
  public static persistedL2GanacheDbPath(defaultValue?: string) {
    return process.env.PERSISTED_L2_GANACHE_DB_FILE_PATH || defaultValue
  }
  public static l2ToL1MessageFinalityDelayInBlocks(
    defaultValue: number = 0
  ): number {
    return process.env.L2_TO_L1_MESSAGE_FINALITY_DELAY_IN_BLOCKS
      ? parseInt(process.env.L2_TO_L1_MESSAGE_FINALITY_DELAY_IN_BLOCKS, 10)
      : defaultValue
  }
  public static l1NodeLevelDBPath(defaultValue?: string): string {
    return process.env.L1_NODE_LEVELDB_PATH || defaultValue
  }

  // L2 Config
  public static l2RpcServerHost(defaultValue: string = '0.0.0.0'): string {
    return process.env.L2_RPC_SERVER_HOST || defaultValue
  }
  public static l2RpcServerPort(defaultValue: number = 8545): number {
    return process.env.L2_RPC_SERVER_PORT
      ? parseInt(process.env.L2_RPC_SERVER_PORT, 10)
      : defaultValue
  }
  public static l2NodeWeb3Url(defaultValue?: string): string {
    return process.env.L2_NODE_WEB3_URL || defaultValue
  }
  public static l2WalletMnemonic(defaultValue?: string): string {
    return process.env.L2_WALLET_MNEMONIC || defaultValue
  }

  // L1 Config
  public static l1NodeWeb3Url(defaultValue?: string): string {
    return process.env.L1_NODE_WEB3_URL || defaultValue
  }
  public static localL1NodePort(defaultValue: number = 7545): number {
    return process.env.LOCAL_L1_NODE_PORT
      ? parseInt(process.env.LOCAL_L1_NODE_PORT, 10)
      : defaultValue
  }
  public static sequencerMnemonic(defaultValue?: string): string {
    return process.env.L1_SEQUENCER_MNEMONIC || defaultValue
  }
  public static l2ToL1MessageReceiverAddress(defaultValue?: string): string {
    return process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS || defaultValue
  }
}
