import 'hardhat/types/runtime'
import 'hardhat/types/config'

interface DeployConfigSpec {
  [key: string]: {
    type: 'address' | 'number' | 'string' | 'boolean'
    default?: any
  }
}

declare module 'hardhat/types/config' {
  interface HardhatUserConfig {
    deployConfigSpec?: DeployConfigSpec
  }

  interface HardhatConfig {
    deployConfigSpec?: DeployConfigSpec
  }

  interface ProjectPathsUserConfig {
    deployConfig?: string
  }

  interface ProjectPathsConfig {
    deployConfig?: string
  }
}

declare module 'hardhat/types/runtime' {
  interface HardhatRuntimeEnvironment {
    deployConfig: {
      // TODO: Is there any good way to type this?
      [key: string]: any
    }
  }
}
