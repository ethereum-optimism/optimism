# vyper-js
`vyper-js` is a simple JavaScript binding to the [Vyper](https://github.com/ethereum/vyper) python compiler.
`vyper-js` does **not** install Vyper for you (yet).
As a result, you currently need to [install Vyper](https://vyper.readthedocs.io/en/latest/installing-vyper.html) before you can use this library.

## Usage
### Node.js
It's pretty easy to use `vyper-js` in a `Node.js` application.
Simply install the package via `npm`:

```
npm install --save @pigi/vyper-js
```

Now just import it into your project:

```js
const vyperjs = require('@pigi/vyper-js')
...
const contract = await vyperjs.compile('./path/to/your/Contract.vy')
console.log(contract) // { bytecode: '0x.....', abi: [....], ... }
```

Check our our more detailed documentation below!

## Contributing
Check out our detailed [Contributing Guide](https://github.com/plasma-group/pigi/blob/master/README.md#contributing) if you'd like to contribute to this project!

## Documentation
The `vyper-js` API is pretty simple - there's currently only a single function!

### vyperjs.compile
```js
vyperjs.compile(path)
```

Compiles the vyper contract at the given path and outputs the compilation result

#### Params
1. `path` - `string`: Path to the Vyper file to compile.

#### Returns
1. `Object`: The compilation result.
    * `bytecode` - `string`: EVM bytecode of the compiled contract.
    * `bytecodeRuntime` - `string`: [Runtime bytecode](https://ethereum.stackexchange.com/questions/32234/difference-between-bytecode-and-runtime-bytecode) for the contract.
    * `abi` - `VyperAbiItem | VyperAbiItem[]`: Ethereum [contract ABI](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI).
    * `sourceMap` - `Object`: Source mapping object.
        * `breakpoints` - `number[]`: List of lines that have breakpoints.
        * `pcPosMap` - `{ [key: string]: [number, number] }`: Mapping of opcode positions to `[line_number, column_offset]` in the original file.
    * `methodIdentifiers` - `{ [key: string]: string }`: Mapping of method signatures to their unique hashes.
    * `version` - `string`: Vyper compiler version used to compile the file.
