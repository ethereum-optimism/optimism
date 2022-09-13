import { fetchJson } from 'ethers/lib/utils'

import { OPTIMISM_TOKENLIST_URI } from './misc-constants'

interface Options {
  uri?: string
  chainId?: number
  filterTerm?: string
}

export interface TokenList {
  name: string
  logoURI: string
  tokens: TokenListItem[]
}

export interface TokenListItem {
  chainId: number
  address: string
  name: string
  symbol: string
  decimals: number
  logoURI?: string
  extensions?: {
    optimismBridgeAddress?: string
    ens?: string
  }
}

export const fetchTokenList = async ({
  uri = OPTIMISM_TOKENLIST_URI,
  chainId,
  filterTerm,
}: Options = {}): Promise<TokenList> => {
  const list: TokenList = await fetchJson(uri)
  return {
    ...list,
    tokens: list.tokens
      // filter by chainId
      .filter((token) => !chainId || token.chainId === chainId)
      // filter by filterTerm
      .filter(
        (token) =>
          !filterTerm ||
          token.address.toLowerCase().includes(filterTerm.toLowerCase()) ||
          token.symbol.toLowerCase().includes(filterTerm.toLowerCase()) ||
          token.name.toLowerCase().includes(filterTerm.toLowerCase())
      ),
  }
}
