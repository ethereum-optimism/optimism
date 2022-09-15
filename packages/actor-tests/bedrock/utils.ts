import { providers, Wallet } from 'ethers'

const DEV_MNEMONIC =
  'test test test test test test test test test test test junk'

export const l1Provider = new providers.JsonRpcProvider(process.env.L1_RPC)
export const l2Provider = new providers.JsonRpcProvider(process.env.L2_RPC)

export const devWalletsL2 = () => {
  const wallets = []
  for (let i = 0; i < 20; i++) {
    wallets.push(
      Wallet.fromMnemonic(DEV_MNEMONIC, `m/44'/60'/0'/0/${i}`).connect(
        l2Provider
      )
    )
  }
  return wallets
}
