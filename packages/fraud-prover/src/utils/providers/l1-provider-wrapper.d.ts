import { JsonRpcProvider } from '@ethersproject/providers';
import { ethers, Contract } from 'ethers';
import { StateRootBatchHeader, StateRootBatchProof, TransactionBatchHeader, TransactionBatchProof, TransactionChainElement, OvmTransaction } from '../../types/ovm.types';
export declare class L1ProviderWrapper {
    provider: JsonRpcProvider;
    OVM_StateCommitmentChain: Contract;
    OVM_CanonicalTransactionChain: Contract;
    OVM_ExecutionManager: Contract;
    l1StartOffset: number;
    l1BlockFinality: number;
    private eventCache;
    constructor(provider: JsonRpcProvider, OVM_StateCommitmentChain: Contract, OVM_CanonicalTransactionChain: Contract, OVM_ExecutionManager: Contract, l1StartOffset: number, l1BlockFinality: number);
    findAllEvents(contract: Contract, filter: ethers.EventFilter, fromBlock?: number): Promise<ethers.Event[]>;
    getStateRootBatchHeader(index: number): Promise<StateRootBatchHeader>;
    getStateRoot(index: number): Promise<string>;
    getBatchStateRoots(index: number): Promise<string[]>;
    getStateRootBatchProof(index: number): Promise<StateRootBatchProof>;
    getTransactionBatchHeader(index: number): Promise<TransactionBatchHeader>;
    getBatchTransactions(index: number): Promise<{
        transaction: OvmTransaction;
        transactionChainElement: TransactionChainElement;
    }[]>;
    getTransactionBatchProof(index: number): Promise<TransactionBatchProof>;
    private _getStateRootBatchEvent;
    private _getTransactionBatchEvent;
}
