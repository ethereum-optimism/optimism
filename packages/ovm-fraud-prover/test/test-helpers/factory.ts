import { ethers } from '@nomiclabs/buidler';
import { ContractFactory, Signer } from "ethers";
import { getContractDefinition } from "@eth-optimism/rollup-contracts"

export const getContractFactory = (contract: string, signer: Signer): ContractFactory => {
  const definition = getContractDefinition(contract)
  return new ethers.ContractFactory(definition.abi, definition.evm.bytecode.object, signer)
}