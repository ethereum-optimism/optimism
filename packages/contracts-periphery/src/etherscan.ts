import fetch from 'node-fetch'

interface NetworkData {
  chainId: number
  names: string[]
  etherscanApiUrl: string
}

const networks: {
  [id: number]: NetworkData
} = {
  1: {
    chainId: 1,
    names: ['mainnet', 'main', 'eth', 'ethereum'],
    etherscanApiUrl: 'https://api.etherscan.io',
  },
  3: {
    chainId: 3,
    names: ['ropsten'],
    etherscanApiUrl: 'https://api-ropsten.etherscan.io',
  },
  4: {
    chainId: 4,
    names: ['rinkeby'],
    etherscanApiUrl: 'https://api-rinkeby.etherscan.io',
  },
  5: {
    chainId: 5,
    names: ['goerli'],
    etherscanApiUrl: 'https://api-goerli.etherscan.io',
  },
  10: {
    chainId: 10,
    names: ['optimism'],
    etherscanApiUrl: 'https://api-optimistic.etherscan.io',
  },
  42: {
    chainId: 42,
    names: ['kovan'],
    etherscanApiUrl: 'https://api-kovan.etherscan.io',
  },
  69: {
    chainId: 69,
    names: ['opkovan', 'kovan-optimism', 'optimistic-kovan'],
    etherscanApiUrl: 'https://api-kovan-optimistic.etherscan.io',
  },
}

export class Etherscan {
  net: NetworkData

  constructor(
    private readonly apiKey: string,
    private readonly network: string | number
  ) {
    if (typeof network === 'string') {
      this.net = Object.values(networks).find((net) => {
        return net.names.includes(network)
      })
    } else {
      this.net = networks[this.network]
    }
  }

  public async getContractSource(address: string): Promise<any> {
    const url = new URL(`${this.net.etherscanApiUrl}/api`)
    url.searchParams.append('module', 'contract')
    url.searchParams.append('action', 'getsourcecode')
    url.searchParams.append('address', address)
    url.searchParams.append('apikey', this.apiKey)
    const response = await fetch(url)
    const result = await response.json()
    return result.result[0]
  }

  public async getContractABI(address: string): Promise<any> {
    const source = await this.getContractSource(address)
    if (source.Proxy === '1') {
      const impl = await this.getContractSource(source.Implementation)
      return impl.ABI
    } else {
      return source.ABI
    }
  }
}
