import type { Address } from 'viem'
import { create } from 'zustand'

type AddressEntry = {
  address: Address
}

type AddressRecord = Record<Address, AddressEntry>

type RecentAddressStore = {
  addressEntryByAddress: AddressRecord
  addAddress: (address: Address) => void
}

export const useRecentAddressStore = create<RecentAddressStore>((set) => ({
  addressEntryByAddress: {
    '0xAaA2b0D6295b91505500B7630e9E36a461ceAd1b': {
      address: '0xAaA2b0D6295b91505500B7630e9E36a461ceAd1b',
    },
  },

  addAddress: (address: Address) => {
    set((state) => ({
      addressEntryByAddress: {
        [address]: { address },
        ...state.addressEntryByAddress,
      },
    }))
  },
}))
