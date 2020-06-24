# `gas-profiler`

`gas-profiler` is a `Node.js` utility for getting more insight into the gas usage of contract executions.
`gas-profiler` can be used as a command-line application or programmatically as an import to other `Node.js` applications.

## Installation

### Node.js

`gas-profiler` can be easily installed via `yarn` or `npm`:

```sh
yarn add @optimism-monorepo/gas-profiler --save
```

```sh
npm install @optimism-monorepo/gas-profiler --save
```

## Usage

### CLI

When installed globally, `gas-profiler` can be used as a command-line application:

```
> @optimism-monorepo/gas-profiler --help
usage: @optimism-monorepo/gas-profiler [-h] -c CONTRACTJSONPATH [-s CONTRACTSOURCEPATH] -m METHOD
                [-p PARAMS [PARAMS ...]] [-t]


Smart contract gas profiler

Optional arguments:
  -h, --help            Show this help message and exit.
  -c CONTRACTJSONPATH, --contract-json CONTRACTJSONPATH
                        Path to the contract's compiled JSON file.        
  -s CONTRACTSOURCEPATH, --contract-source CONTRACTSOURCEPATH
                        (Optional) Path to the contract's source file.    
  -m METHOD, --method METHOD
                        Contract method to call
  -p PARAMS [PARAMS ...], --params PARAMS [PARAMS ...]
                        (Optional) Contract method parameters
  -t, --trace           (Optional) Generates a full gas trace. Source file must
                        be specified.
```

`[-c, --contract-json]` and `[-m, --method]` are both required arguments. All other arguments are optional.

If the `[-t, --trace]` flag is enabled, `[-s, --contract-source]` **must** be provided.

### Node.js

`gas-profiler` can also be used as an import to `Node.js` applications:

```ts
import * as fs from 'fs';
import { GasProfiler } from '@optimism-monorepo/gas-profiler';

const myFunction = async () => {
  const profiler = new GasProfiler();

  // Must be called before running any profiles.
  await profiler.init();

  // Load your contract JSON file.
  const contractJson = JSON.parse(fs.readFileSync('./path/to/Contract.json', 'utf8'));

  const profile = await profiler.execute(contractJson, {
    method: 'myContractFunction',
    params: [1234, 'hello', 'wow!'],
  });

  console.log('Total gas used: ', profile].gasUsed);

  // Shut down the ganache instance.
  await profiler.kill();
}
```

## API

### GasProfiler

#### `init`
```ts
init(options?: GasProfilerOptions): Promise<void>
```

##### Description
Connects the profiler to an Ethereum node. Must be called at least once before any profiles can be executed. If no options are provided, `init` will also spin up a `ganache` instance on a random open port.

##### Parameters
* `options?: { provider: ethers.providers.Provider, wallet: ethers.Wallet }` - Custom `ethers` provider/wallet to use for profiling. If neither is provided, will also default to using `ganache`.

#### `kill`
```ts
kill(): Promise<void>
```

##### Description
Shuts down the profiler. Necessary only if no options were provided and a `ganache` instance was created.

#### `profile`
```ts
profile(target: ContractJson, sourcePath: string, parameters: ProfileParameters): Promise<ProfileResult>
```

##### Description
Generates a full gas profile for a given contract execution.

##### Parameters
* `target: ContractJson` - JSON `Solidity` compiler output for the target contract.
* `sourcePath: string` - Path to the `.sol` file for the target contract.
* `parameters: { method: string, params: any[] }` - Parameters for the contract execution.

##### Returns
* `ProfileResult` - Result of the profile execution, including line-by-line gas usage.

#### `execute`
```ts
execute(target: ContractJson, parameters: ProfileParameters): Promise<ProfileResult>
```

##### Description
Executes a contract method and returns total gas usage. Does **not** generate a full profile.

##### Parameters
* `target: ContractJson` - JSON `Solidity` compiler output for the target contract.
* `parameters: { method: string, params: any[] }` - Parameters for the contract execution.

##### Returns
* `ProfileResult` - Result of the profile execution, **not including** line-by-line gas usage.

#### `prettify`
```ts
prettify(trace: CodeTrace): string
```

##### Description
Prettifies a code trace into a nice line-by-line string.

##### Parameters
* `trace: CodeTrace` - Trace to prettify.

##### Returns
* `string` - Prettified trace, ready for `console.log`.