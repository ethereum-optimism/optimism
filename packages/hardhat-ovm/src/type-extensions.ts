import 'hardhat/types/config'

declare module 'hardhat/types/config' {
  interface HardhatUserConfig {
    ovm?: {
      solcVersion?: string
    }
  }

  interface HardhatConfig {
    ovm?: {
      solcVersion?: string
    }
  }

  interface HardhatNetworkUserConfig {
    ovm?: boolean
    ignoreRxList?: string[]
  }

  interface HttpNetworkUserConfig {
    ovm?: boolean
    ignoreRxList?: string[]
  }

  interface HardhatNetworkConfig {
    ovm: boolean
    ignoreRxList: string[]
    interval?: number
  }

  interface HttpNetworkConfig {
    ovm: boolean
    ignoreRxList: string[]
    interval?: number
  }
}

declare module 'hardhat/types/runtime' {
  interface Network {
    ovm: boolean
    ignoreRxList: string[]
  }
}
