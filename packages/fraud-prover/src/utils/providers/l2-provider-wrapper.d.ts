import { JsonRpcProvider } from '@ethersproject/providers';
import { StateDiffProof } from '../../types';
export declare class L2ProviderWrapper {
    provider: JsonRpcProvider;
    constructor(provider: JsonRpcProvider);
    getStateRoot(index: number): Promise<string>;
    getTransaction(index: number): Promise<string>;
    getProof(index: number, address: string, slots?: string[]): Promise<any>;
    getStateDiffProof(index: number): Promise<StateDiffProof>;
    getRollupInfo(): Promise<any>;
    getAddressManagerAddress(): Promise<string>;
}
