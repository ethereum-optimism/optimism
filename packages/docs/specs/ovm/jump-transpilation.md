# JUMP Transpilation

`JUMP` and `JUMPI` instructions are only allowed to jump onto
destinations that are (1) occupied by a `JUMPDEST` opcode, and (2) are
not inside PUSH data. We make sure we update the destinations we are
`JUMP`ing to to account for the extra opcodes we are adding in by
appending an assembly switch/case to all transpiled bytecode.

## The Approach

Here's how we deal with it, at a high level: 1. Create a map of all
pre-transpilation `JUMPDEST` locations to post-transpilation `JUMPDEST`
locations 2. Add some footer bytecode that acts as a `switch` statement,
reading the pre-transpilation `JUMPDEST` location and `JUMP`ing to the
associated post-transpilation `JUMPDEST` 3. Replace all `JUMP`s and
`JUMPI`s to `JUMP` to the footer bytecode switch statement.

### JUMP Transpilation Detail: Replacements

Note: operations will be listed as `[operation]` -- `[resulting stack]`

`JUMP`: - Expected Stack: `<dest>` - Replacement: - `PUSH32 <JUMPDEST of
footer switch statement>` -- `<JUMPDEST of footer switch statement>
<dest>` - `JUMP` -- `<dest>` - Total Replacement: - `JUMP` =\> `PUSH32
<JUMPDEST of footer switch statement> JUMP`

`JUMPI`: - Expected Stack: `<dest> <condition>` - Replacement: - `SWAP1`
-- `<condition> <dest>` - `PUSH32 <JUMPDEST of footer switch statement>`
-- `<JUMPDEST of footer switch statement> <condition> <dest>` - `JUMPI`
-- `<dest>` - `POP` -- `empty` - Total Replacement: - `JUMPI` =\> `SWAP1
PUSH32 <JUMPDEST of footer switch statement> JUMPI POP`

`JUMPDEST`: - Expected Stack: `<prev jumpdest from footer switch>`
(footer switch statement results in 1 excess stack element) -
Replacement: - `JUMPDEST` -- `<prev jumpdest from footer switch>` -
`POP` -- `empty` - Total Replacement: - `JUMPDEST` =\> `JUMPDEST POP`

JUMP Transpilation Detail: Footer Switch Statement
---------------------------------------

  - Expected Stack: `<prev jumpdest>`
  - Single comparison:
      - `DUP1` -- `<prev jumpdest> <prev jumpdest>`
      - `PUSH32 <first compare jumpdest>` -- `<first compare jumpdest>
        <prev jumpdest> <prev jumpdest>`
      - `EQ` -- `<true/false> <prev jumpdest>`
      - `PUSH32 <post-transpile jumpdest>` -- `<post-transpile jumpdest>
        <true/false> <prev jumpdest>`
      - `JUMPI` -- `<prev jumpdest>`

Duplicate above code once for each (compare jumpdest, post-transpile
jumpdest) pair

**Note on bytecode interpretation**

Note that properly processing these conditions requires preprocessing
the code; a particularly pathological use case is `PUSH2 JUMPDEST PUSH1
PUSH2 JUMPDEST PUSH1 PUSH2 JUMPDEST PUSH1 ...`, as this code has all
`JUMPDEST`s invalid but an alternative piece of code equivalent to this
but only with the leading `PUSH2` replaced with another op (eg.
`BALANCE`) will have all `JUMPDESTS` valid. We appropriately deal with
this, both in our transpiler and purity checker.
