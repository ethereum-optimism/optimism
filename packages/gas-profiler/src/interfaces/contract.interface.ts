export interface ContractJson {
  abi: any;
  evm: {
    bytecode: {
      linkReferences: any;
      object: string;
      opcodes: string;
      sourceMap: string;
    },
    deployedBytecode: {
      immutableReferences: any;
      linkReferences: any;
      object: string;
      opcodes: string;
      sourceMap: string;
    }
  };
  interface: any;
  bytecode: string;
}