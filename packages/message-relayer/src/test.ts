import { ethers } from 'ethers'
import { relayAllMessagesInL2Transaction } from './relay-tx'

const main = async () => {
  const pk =
    '0xf946da3ac20284914ac590bcfd37e818e39d2bf0013f6f56212c79a5705b337b'
  const l1 = 'https://kovan.infura.io/v3/afd695a8cfbc4de1838c1b285307b80f'
  const l2 = 'https://kovan.optimism.io'
  const tx =
    '0xbb162fd5f14367f15ad2cba080b509a333a4f59aa8de5d4f6b28b1c2a23de81d'

  const l1p = new ethers.providers.JsonRpcProvider(l1)
  const wallet = new ethers.Wallet(pk, l1p)
  await relayAllMessagesInL2Transaction(wallet, l1, l2, tx)
}

main()
