import { ethers, Wallet, Contract } from 'ethers';
import { Provider, JsonRpcProvider } from 'ethers/providers';
import { keccak256, FunctionDescription } from 'ethers/utils';

import { CodeTrace } from './interfaces/trace.interface';
import { ContractJson } from './interfaces/contract.interface';
import { getTransactionTrace, prettifyTransactionTrace } from './helpers/tx-trace';
import { getToolbox, getInterface, Toolbox } from './helpers/utils';

interface ProfileResult {
  hash: string;
  gasUsed: number;
  trace?: CodeTrace;
}

interface ProfileParameters {
  method: string;
  params: any[];
}

interface GasProfilerOptions {
  provider: Provider;
  wallet: Wallet;
}

/**
 * Utility for generating gas profiles of contract executions.
 */
export class GasProfiler {
  private _ready: boolean;
  private _toolbox: Toolbox;
  private _cache: { [hash: string]: Contract } = {};

  /**
   * Initializes the profiler. Must be called at least once.
   * @param options Options to pass to the profiler.
   */
  public async init(options?: GasProfilerOptions): Promise<void> {
    if (this._ready) {
      return;
    }

    if (options) {
      this._toolbox = options;
    } else {
      this._toolbox = await getToolbox();
    }

    this._ready = true;
  }

  /**
   * Shuts down the profiler if using a temporary ganache instance.
   */
  public async kill(): Promise<void> {
    if (this._toolbox.ganache && this._toolbox.ganache.running) {
      await this._toolbox.ganache.stop();
    }
  }

  /**
   * Generates a profile for a given contract execution.
   * @param target compiled contract JSON object.
   * @param sourcePath path to the contract source file.
   * @param parameters parameter defining the execution.
   * @returns a profile for the execution.
   */
  public async profile(
    target: ContractJson,
    sourcePath: string,
    parameters: ProfileParameters,
  ): Promise<ProfileResult> {
    this._checkReady();
    
    const result = await this._runTransaction(target, parameters);
    const trace = await getTransactionTrace(this._toolbox.provider as JsonRpcProvider, sourcePath, target, result.hash);

    return {
      ...result,
      trace,
    };
  }

  /**
   * Executes a set of given method calls and returns gas used and the result.
   * Does not generate a full profile.
   * @param target compiled contract JSON object.
   * @param parameters parameters defining the execution.
   * @returns gas used and result of the execution.
   */
  public async execute(
    target: ContractJson,
    parameters: ProfileParameters,
  ): Promise<ProfileResult> {
    this._checkReady();

    return this._runTransaction(target, parameters)
  }

  /**
   * Prettifies a code trace.
   * @param trace trace to print.
   * @returns prettified trace
   */
  public prettify(
    trace: CodeTrace
  ): string {
    return prettifyTransactionTrace(trace);
  }

  /**
   * Checks that the profiler has been initialized.
   */
  private _checkReady(): void {
    if (!this._ready) {
      throw new Error("GasProfiler not initialized (call .init)");
    }
  }

  /**
   * Executes a particular transaction against some target contract.
   * @param target compiled contract JSON object.
   * @param parameters parameters for the transaction.
   * @returns the result of the transaction.
   */
  private async _runTransaction(
    target: ContractJson,
    parameters: ProfileParameters,
  ): Promise<ProfileResult> {
    const deployed = await this._deploy(target);
    const subject = getInterface(deployed).functions[parameters.method];

    const transaction = await this._makeSignedTransaction(deployed, subject, parameters.params);
    const response = await this._toolbox.provider.sendTransaction(transaction);
    const receipt = await this._toolbox.provider.getTransactionReceipt(response.hash);

    return {
      hash: response.hash,
      gasUsed: receipt.gasUsed.toNumber(),
    };
  }

  /**
   * Generates a signed transaction for a given contract execution.
   * @param target compiled contract JSON object.
   * @param method contract method to call.
   * @param params parameters to pass to the method.
   * @returns the signed transaction.
   */
  private async _makeSignedTransaction(
    target: Contract,
    method: FunctionDescription,
    params: any[],
  ): Promise<string> {
    const nonce = await this._toolbox.provider.getTransactionCount(this._toolbox.wallet.address);
    const calldata = method.encode(params);
    const transaction = {
      gasLimit: 8000000,
      data: calldata,
      to: target.address,
      nonce: nonce,
    };
    return this._toolbox.wallet.sign(transaction);
  } 

  /**
   * Deploys a contract.
   * @param target compiled contract JSON object.
   * @returns deployed `ethers` contract object.
   */
  private async _deploy(target: ContractJson): Promise<Contract> {
    const hash = keccak256('0x' + target.evm.bytecode.object);
    if (hash in this._cache) {
      return this._cache[hash];
    }

    const targetFactory = new ethers.ContractFactory(target.abi, target.evm.bytecode.object, this._toolbox.wallet);
    const deployed = await targetFactory.deploy();
    this._cache[hash] = deployed;
    return deployed;
  }
}
