import { providers, Wallet } from 'ethers'

const DEV_MNEMONIC =
  'test test test test test test test test test test test junk'

export const devWalletsL2 = () => {
  const provider = new providers.JsonRpcProvider(process.env.L2_RPC)
  const wallets = []
  for (let i = 0; i < 20; i++) {
    wallets.push(
      Wallet.fromMnemonic(DEV_MNEMONIC, `m/44'/60'/0'/0/${i}`).connect(provider)
    )
  }
  return wallets
}
