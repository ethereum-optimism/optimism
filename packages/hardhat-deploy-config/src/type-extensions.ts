import 'hardhat/types/runtime'
import 'hardhat/types/config'

import { DeployConfigSpec } from './types'

declare module 'hardhat/types/config' {
  interface HardhatUserConfig {
    deployConfigSpec?: DeployConfigSpec<any>
  }

  interface HardhatConfig {
    deployConfigSpec?: DeployConfigSpec<any>
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
