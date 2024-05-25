export * from './generated'

type PaginationOptions = {
  limit?: number
  cursor?: string
}

type Options = {
  baseUrl?: string
  address: `0x${string}`
} & PaginationOptions

const createQueryString = ({ cursor, limit }: PaginationOptions): string => {
  if (cursor === undefined && limit === undefined) {
    return ''
  }
  const queries: string[] = []
  if (cursor) {
    queries.push(`cursor=${cursor}`)
  }
  if (limit) {
    queries.push(`limit=${limit}`)
  }
  return `?${queries.join('&')}`
}

export const depositEndpoint = ({ baseUrl = '', address, cursor, limit }: Options): string => {
  return [baseUrl, 'deposits', `${address}${createQueryString({ cursor, limit })}`].join('/')
}

export const withdrawalEndoint = ({ baseUrl = '', address, cursor, limit }: Options): string => {
  return [baseUrl, 'withdrawals', `${address}${createQueryString({ cursor, limit })}`].join('/')
}

