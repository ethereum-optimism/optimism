import { ExecutionResult, GenesisData, TxExecutionOptions, VMOptions } from 'ethereumjs-vm';
export declare class VM {
    private vm;
    constructor(options?: VMOptions);
    generateGenesis(initState: GenesisData): Promise<any>;
    runTx(options: TxExecutionOptions): Promise<ExecutionResult>;
}
