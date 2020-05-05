# Opcode Transpilation Overview

This pages provides a quick reference which discusses how every EVM opcode is handled in the transpilation process. There are three classes of opcodes: 1. Pure opcodes which do not need to be modified. 2. Replaced opcodes which are substituted with other bytecode by the transpiler. 3. Banned opcodes which are not replaced and simply disallowed.

### Pure Opcodes

The following opcodes perform stack operations which are constant in terms of L1/L2 state, and do not require modification:

* Arithmetic/pure-math opcodes:
    * `ADD, MUL, SUB, DIV, SDIV, MOD, SMOD, ADDMOD, MULMOD, EXP, SIGNEXTEND, LT, GT, SLT, SGT, EQ, ISZERO, AND, OR, XOR, NOT, BYTE, SHL, SHR, SAAR, SHA3`
* "Pure" code execution operations:
    * `PUSH1....PUSH32, DUP1...DUP16, SWAP1...SWAP16, POP, LOG0...LOG4, STOP, REVERT, RETURN, PC, GAS, JUMPDEST*`
      \*NOTE: In practice, `JUMPDEST`s are modified, but not "transpiled away" like the impure opcodes.  See JUMP transpilation [section](protocol-specifications/ovm/jump-transpilation.md) for more details.
* "Pure" memory modifying operations:
    * `MLOAD, MSTORE, MSTORE8, MSIZE`
* Permitted execution-context-dependent operations:
    * `CALLVALUE\*, CALLDATALOAD, CALLDATASIZE, CALLDATACOPY, CODESIZE, RETURNDATASIZE, RETURNDATACOPY`
      \*Note: `CALLVALUE` will always be 0 because we enforce that all `CALL` s always pass 0 in our purity checking.

### Replaced Opcodes

The following opcodes need to be dealt with at transpilation time to work with the Execution Manager. Note: their transpiled versions may modify memory and/or the stack in ways their non-transpiled versions would not, but at the end of execution, the resulting memory and stack are guaranteed to be the same as they otherwise would be. \#\#\# Non-memory-utilizing Opcodes These opcodes do not modify memory over the course of execution but do need to be transpiled to be compatible with the Execution Manager. They all utilize a bytecode replacement function, named `callContractWithStackElementsAndReturnWordToStack(...)` in the transpiler, that calls the Execution Manager to fulfill their logic. This function passes any stack elements consumed by the opcode as calldata to the associated Execution Manager function and pushes the result to the stack.

| Opcode | Description | Num Stack Arguments to Pass | Num Stack Elements Returned |
| :--- | :--- | :--- | :--- |
| `ADDRESS` | Returns the address of the currently execution contract. Needs to be modified since on L1 this would be the code contract's address, but on L2 it will be an OVM address. | 0 | 1 |
| `CALLER`\* | This is `msg.sender` in solidity. Needs to be modified since on L1 this would be the Execution Manager's address, but on L2 it is meant to be an OVM address. | 0 | 1 |
| `EXTCODESIZE` | This gets the size of an external contract's code. Needs to be modified on L2 since it is meant to accept an OVM address which doesn't exist on L1. | 1 \(`addr`\) | 1 |
| `EXTCODEHASH` | This gets the size of an external contract's code. Needs to be modified on L2 since it is meant to accept an OVM address which doesn't exist on L1. | 1 \(`addr`\) | 1 |
| `TIMESTAMP`\*\* | This gets the timestamp of the current block \(in Solidity: `block.timestamp`\). Needs to be transpiled to the `ovmTIMESTAMP`. | 0 | 1 |
| `SLOAD` | This gets the value of a storage slot at the first stack input \(`key`\). Needs to be transpiled to `ovmSLOAD` instead. | 1 \(`key`\) | 1 |
| `SSTORE` | This gets the value of a storage slot at the first stack input \(`key`\). Needs to be transpiled to `ovmSSTORE` instead. | 2 \(`key, value`\) | 0 |

\* Note 1: we are currently using metatransactions, having no EOAs, and assuming all transactions are handled with account abstraction. Because of this, at the initial entry point of a rollup transaction, `CALLER` will revert the transaction--unlike the EVM's usual behavior.

\*\* Note 2: The timestamp will correspond to the timestamp of the ORU block, and not any L1 Ethereum block.

#### Memory-reading opcodes

These opcodes read from, but do not write to, the current execution's memory, and also need to be transpiled to work in the context of the OVM. For now, there are two such opcodes: `CREATE` and `CREATE2`, whose stack inputs specify the memory range of the initcode used to deploy the new contract. There are two functions which we use to generate replacement functions in Typescript: `getCREATEReplacement(...)` and `getCREATE2Replacement(...)`. They work by prepending `calldata` \(`methodId`, as well as `salt` if `CREATE2`\) to the existing memory of the `initcode`, calling the associated function in Execution Manager, and pushing the `returndata` onto the stack.

#### Memory-writing opcodes

These opcodes write to, but do not require reading from, the current execution's memory, and also need to be transpiled to work in the context of the OVM. For now, there is only one such opcode: `EXTCODECOPY`. Its replacement function in typescript is called `getEXTCODECOPYReplacement(...)`. The transpiled bytecode passes the `methodId` and necessary stack inputs \(namely, the `addr` whose code is desired to copy\) as `calldata` to the associated Execution Manager function, and uses the memory modification inputs in the original stack as the `retOffset` and `retLength` for the transpiled `CALL` to the Execution Manager. Thus the Execution Manager's returned copy of the code modifies the memory as expected.

#### `CALL`-type Opcodes

To replace Call-type opcodes, we have to pass an existing slice of `calldata` at `argOffset, argLength`, along with the `methodId` and `target` address. The typescript function `getCallTypeReplacement(...)` handles these replacements, dynamically prepending the `methodId` and `addr` stack input to the existing `calldata` memory, updating the CALL's `argOffset` and `argLength` as necessary and routing the CALL to the appropriate Execution Manager function. Because `STATICCALL` and `DELEGATECALL` do not have a `value` stack input, the function accepts as an argument `stackPositionOfCallArgsMemOffset: number` to locate the memory parameters.

| Opcode | `stackPositionOfCallArgsMemOffset` |
| :--- | :--- |
| `CALL` | 3 |
| `STATICCALL` | 2 |
| `DELEGATECALL` | 2 |

#### Special cases: `CODECOPY` and `JUMP` s

There are two functions which are "Pure code execution operations" just like `CODESIZE`, `REVERT`, etc., however, they are used by the Solidity compiler in ways which the transpilation process affects, and need to be dealt with in the transpiler.

* Because we are inserting bytecode, we are changing the index of  every `JUMPDEST` proceeding each insertion operation. This means  our `JUMP` and `JUMPI` values need to be transpiled or they will  fail/go to the wrong place. We handle this by making all `JUMP` s  go to new bytecode that we append at the end that simply contains  a mapping from untranspiled `JUMPDEST` bytecode location to  transpiled `JUMPDEST` bytecode location. The logic finds the new  location and `JUMP` s to it. See the \["JUMP Modification"  page\]\([https://github.com/op-optimism/optimistic-rollup/wiki/JUMP-Transpilation](https://github.com/op-optimism/optimistic-rollup/wiki/JUMP-Transpilation)\)  for more details.

* The opcode `CODECOPY` works fine, in principle, in our code   contracts, as its effect on execution is independent of L1 state. However, because that code itself is modified by transpilation, we need to deal with it in the transpiler. See our `CODECOPY` [section](./codecopy.html) for how we handle these modifications.

### Banned Opcodes

The remaining opcodes are explicitly banned, either because we don't yet support them, or do not plan to/it's impossible.

#### Opcodes which could later be implemented

These opcodes are banned simply because we don't want to support them currently.

#### ETH-native Value

We have made the decision for now not to use native ETH, and instead do everything with wrapped ETH \(WETH\). Note: `CALLVALUE` is actually able to be whitelisted, because our Purity Checker enforces that all Calls are made with a value of 0. Contracts are welcome to use msg.value, it will just always return 0. This means that the following opcodes are banned, not just transpiled: - `BALANCE` -- gets `address(this).balance` While not a ban, another note here is that all `value`-related inputs to other opcodes like `CREATE` or `CALL` are overridden to `0` by their transpiled counterparts. We do have good inline documentation for how a native `value` could be added if needed. Another option is we could even transpile the native ETH opcodes to use `WETH` instead. TBD.

#### Others

* `NUMBER` -- the relationship between `block.number` in L1 and L2 is unclear so we've banned. In the future, we could even transpile toreturn `timestamp/avg. blocktime` but unclear if this is a good idea.
* `GASPRICE` -- Before we implement proper gas metering ramifications, we shouldn't transpile anything here. Down the line, we may need to and can potentially add it depending on how we handle.
* `GASLIMIT` -- see `GASPRICE`, same arguments apply.
* `BLOCKHASH` -- in theory the previous state roots can be accessible to the OVM, but because it is EXTREMELY manipulable by the single-party sequencer, and usually used as a bad source of randomness, we'll ban for now. Down the line, we can expose historic L1 blockhashes for this purpose, but that's a lot of work \(and still a bad idea for randomness even on L1!\).
* `ORIGIN` -- see note on `CALLER` and metatransactions. In the future, could transpile to the metatransaction library's standard, once we're more confident in that approach/choice.
* `CALLCODE` -- This opcode was a failed implementation of `DELEGATECALL`. Deprecated, extremely low priority to support.
* `SELFDESTRUCT` -- This opcode is currently unsupported, and we also will not be able to handle it's default functionality to send all ETH of self destructed contract to a designated address

#### "Impossible" opcodes

* `COINBASE` -- since we don't have inflation in L2
* `DIFFICULTY` -- since there is no sense of difficulty in L2. An analogous value in L2 is actually the MEVA price, but it's not so analogous that transpiling would make any sense.

