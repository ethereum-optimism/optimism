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
}
