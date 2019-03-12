declare module 'plasma-contracts' {
  interface CompiledContract {
    abi: any[]
    bytecode: string
  }

  const plasmaChainCompiled: CompiledContract
  const erc20Compiled: CompiledContract
  const plasmaRegistryCompiled: CompiledContract
}
