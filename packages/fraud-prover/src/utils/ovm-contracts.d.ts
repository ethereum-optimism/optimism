import { Contract } from 'ethers';
import { JsonRpcProvider } from '@ethersproject/providers';
export declare const loadContract: (name: string, address: string, provider: JsonRpcProvider) => Contract;
export declare const loadContractFromManager: (name: string, Lib_AddressManager: Contract, provider: JsonRpcProvider) => Promise<Contract>;
export declare const loadProxyFromManager: (name: string, proxy: string, Lib_AddressManager: Contract, provider: JsonRpcProvider) => Promise<Contract>;
