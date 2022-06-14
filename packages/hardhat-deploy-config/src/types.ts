export type DeployConfigSpec<
  TDeployConfig extends {
    [key: string]: any
  }
> = {
  [K in keyof TDeployConfig]: {
    type: 'address' | 'number' | 'string' | 'boolean'
    default?: any
  }
}
