import type { Account, Address, PublicClient, WalletClient } from 'viem'

/**
 * Manages per-chain public and wallet clients, with utilities for:
 * - Per-chain client lookup and retrieval
 * - Account resolution
 */
export class ClientManager {
  constructor(
    private readonly clients: Record<number, PublicClient>,
    private readonly walletClients: Record<number, WalletClient[]>,
  ) {}

  getPublicClient(chainId: number): PublicClient | undefined {
    return this.clients[chainId]
  }

  getWalletClient(chainId: number): WalletClient | undefined {
    return this.walletClients[chainId]?.[0]
  }

  getWalletClients(chainId: number): WalletClient[] {
    return this.walletClients[chainId] ?? []
  }

  /**
   * Lists every (signing EOA, chainId) pair the relayer holds a private
   * key for. Empty in sponsored mode (wallet clients carry no `account`).
   * Used by the admin /relayer-balance route to enumerate cells.
   */
  listSigningEoas(): Array<{ address: Address; chainId: number }> {
    const out: Array<{ address: Address; chainId: number }> = []
    for (const [chainIdStr, clients] of Object.entries(this.walletClients)) {
      const chainId = Number(chainIdStr)
      for (const c of clients) {
        const addr = c.account?.address
        if (addr) out.push({ address: addr, chainId })
      }
    }
    return out
  }

  /**
   * Resolves accounts for all wallet clients on a chain.
   * Returns pairs of { account, walletClient } for each wallet that can provide an account.
   */
  async resolveWallets(
    chainId: number,
  ): Promise<Array<{ account: Account; walletClient: WalletClient }>> {
    const walletClients = this.getWalletClients(chainId)
    const wallets: Array<{ account: Account; walletClient: WalletClient }> = []

    for (const walletClient of walletClients) {
      const account = await this.resolveAccount(walletClient)
      if (account) wallets.push({ account, walletClient })
    }

    return wallets
  }

  /**
   * Resolves the account to use for a transaction. If the wallet client
   * has an account configured (private key mode), returns it directly.
   * If not (sponsored mode), fetches addresses and picks a random one.
   * Returns undefined if no accounts are available.
   */
  async resolveAccount(
    walletClient: WalletClient,
  ): Promise<Account | undefined> {
    if (walletClient.account) return walletClient.account
    const accounts = await walletClient.getAddresses()
    if (accounts.length === 0) return undefined
    const randomIndex = Math.floor(Math.random() * accounts.length)
    return { address: accounts[randomIndex], type: 'json-rpc' }
  }

}
