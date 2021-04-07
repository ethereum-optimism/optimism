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
  }

  interface HttpNetworkUserConfig {
    ovm?: boolean
  }

  interface HardhatNetworkConfig {
    ovm: boolean
  }

  interface HttpNetworkConfig {
    ovm: boolean
  }
}

declare module 'hardhat/types/runtime' {
  interface Network {
    ovm: boolean
  }
}
