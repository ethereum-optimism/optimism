# CODECOPY Transpilation

The opcode `CODECOPY` accepts `memOffset`, `codeOffset`, and `length` inputs from the stack, modifying the memory so that `memory[memOffset:memOffset + length] = code[codeOffset:codeOffset + length]`. Since we are by definition modifying the `code` of a contract by transpiling, there is no general way to handle pre-transpiled `CODECOPYs` since the impact on execution is dependent on how the `CODECOPY` was expected to be used. For Solidity, there are three ways in which `CODECOPY` is used:

> 1. Constants which exceed 32 bytes in length are stored in the
>
>    bytecode and `CODECOPY` ed for use in execution. \(if \&lt;=32 bytes
>
>    they are just `PUSHN` ed\)
>
> 2. All constructor logic for initcode is prefixed to the bytecode to
>
>    be deployed, and this prefix runs \`CODECOPY\(suffixToDeploy\), ...,
>
>    RETURN`during`CREATE\(2\)\` .
>
> 3. Constructor parameters are passed as bytecode and then are
>
>    `CODECOPY` ed to memory before being treated like calldata.

This document explains each of these cases and how they're handled transpilation-side.

### Constants

Constants larger than 32 bytes are stored in the bytecode and `CODECOPY` ed to access in execution. Initial investigation shows that the pattern which always occurs leading up to a `CODECOPY` for constants is:

```text
...
PUSH2 // offset
PUSH1 // length
SWAP2 // where to shove it into memory
CODECOPY
...
```

With some memory allocation operations preceding the `PUSH, PUSH, SWAP...` which may also be standard \(haven't tested\). Some sample code which this example was pulled from can be found [this gist](https://gist.github.com/ben-chain/677457843793d7c6c7feced4e3b9311a) .

To deal with constants, we still want to copy the correct constant--this will just be at a different index once we insert transpiled bytecode above it. So, we just increase the `codeOffset` input to `CODECOPY` in every case that a constant is being loaded into memory. Hopefully, all constants are appended to the end of a file so that we may simply add a fixed offset for every constant.

### Returning deployed bytecode in `CREATE(2)`

All constructor logic for initcode is prefixed to the bytecode to be deployed, and this prefix runs `CODECOPY(suffixToDeploy), ..., RETURN` during `CREATE(2)`. If constructor logic is empty \(i.e. no `constructor()` function specified in Solidity\) this prefix is quite simple but still exists. This `CODECOPY` simply puts the prefix into memory so that the deployed byetcode can be deployed. So, what we need to do is increase the `length` input both to `CODECOPY` and the `RETURN`. The `CODECOPY, RETURN` pattern seems to appear in the following format:

```text
PUSH2 // codecopy's and RETURN's length
DUP1 // DUPed to use twice, for RETURN and CODECOPY both
PUSH2 // codecopy's offset
PUSH1 codecopy's destOffset
CODECOPY // copy
PUSH1 0 // RETURN offset
RETURN // uses above RETURN offset and DUP'ed length above
```

So by adding to the consumed bytes of the first `PUSH2` above, in accordance to the extra bytes added by transpilation, we make sure the correct length is both `CODECOPY`ed and `RETURN` ed. Note that, if we have constructor logic which gets transpiled, this will require modifying the `// codecopy's offset` line above as well.

### Constructor Parameters

Constructor parameters are passed as bytecode and then are `CODECOPY` ed to memory before being treated like calldata. This is because the EVM execution which is initiated by `CREATE(2)` does not have a calldata parameter, so the inputs must be passed in a different way. For more discussion, check out [this discussion](https://ethereum.stackexchange.com/questions/58866/how-does-a-contracts-constructor-work-and-load-input-values) on stack exchange.

We handle this similarly to how we handle constants, by changing the `codeOffset` input appropriately. Both constants used in the constructor and constructor inputs are at the end of the file.

The pattern it uses is:

```text
...
[PC 0000000015] PUSH2: 0x01cf // should be initcode.length + deployedbytecode.length
[PC 0000000018] CODESIZE
[PC 0000000019] SUB // subtract however big the code is from the amount pushed above to get the length of constructor input
[PC 000000001a] DUP1
[PC 000000001b] PUSH2: 0x01cf // should also be initcode.length + deployedbytecode.length
[PC 000000001e] DUP4
[PC 000000001f] CODECOPY
```

