# Forge Scripts in Go

This package provides Forge scripting functionality in Go. It allows you to write scripts in Solidity, but call them
from Go using an in-memory EVM. This is useful for building tooling, since it eliminates the need to shell out to Forge
or install the Solidity toolchain.

# Usage

To use the Forge scripts, you'll need to create a script host. A minimal example of how to do this below:

```go
artifactsFS := foundry.OpenArtifactsDir("path/to/forge-artifacts")
host := NewHost(logger, artifactsFS, nil, DefaultContext, nil)
```

You can pass in optional arguments to `NewHost` to customize its behavior:

- `WithBroadcastHook` will allow you to broadcast the transactions initiated using `vm.broadcast` on chain via an
  external broadcaster.
- `WithIsolatedBroadcast` will clear the warm state between calls, which allows broadcasted gas estimates to be more
  accurate.
- `WithCreate2Deployer` will configure the `CREATE2` opcode to broadcast transactions to the Arachnid `Create2Deployer`.

Typically, Forge scripts in Go are implemented using the following pattern:

1. Define a series of simple input and output contracts in Solidity.
2. Implement your script, and have it take the input/output contracts as arguments to the `run` method.
3. Define a struct that matches the input/output contracts in Go.
4. Etch the input and output structs as contracts in Go, then call `run` using the etched input/output contract
   addresses.
5. Your script will run, and the output contract will contain the results.

For an example, see [`opsm/implementations.go`](../deployer/opsm/implementations.go).

# Known Limitations

- Structs are not supported in contracts etched from Go code.
- Bytecode containing linked libraries are not currently supported.

# Best Practices

## Keep I/O Contracts Simple

I/O contracts are the bridge between the Go code and Solidity. You should keep them as simple as possible. Specifically:

1. Use only primitive types in the I/O contracts. This means no structs, arrays, or mappings. The Go script engine will
   automatically generate getters/setters for public struct fields.
2. Avoid defining logic in your I/O contracts. Any logic you add in Solidity will have to be reimplemented in Go. If you
   do add logic to your I/O contract (e.g., for validation) keep it simple.
3. Avoid mutating your I/O contracts once they have been created.

### Examples

This contract:

```solidity
contract Input {
    // Primitive types below
    uint256 internal _a;
    uint256 internal _b;
    bytes internal _c;

    // The below function will have to be reimplemented in Go, but it's really simple
    function valuesSet() public view returns (bool) {
        return _a != 0 && _b != 0 && _c.length != 0;
    }
}
```

Maps to this Go code:

```go
type Input struct {
A uint64
B uint64
C []byte

D uint64 `evm:"-"` // You can avoid creating setters/getters for this field using this tag
}

func (i *Input) ValuesSet() bool {
return true
}
```

The following are examples of antipatterns for I/O contracts:

```solidity
// Contracts with complex types
contract Input {
    MegaStruct internal _a; // structs are not supported
}

// Contracts containing logic that gets called from the script
contract Input {
    uint256 internal _a;

    function setA(uint256 a) public {
        require(a > 0, "a must be greater than 0");
        _a = a;
    }
}

contract FooScript is Script {
    function run(Input input, Output output) public {
        // This method will have to be reimplemented in Go for it to be callable. 
        // However, it does not need to be reimplemented if it's not called in the script.
        input.setA(5);
    }
}
```

## Prefer Parameters to Conventions

Scripts will be called from a variety of different environments. Therefore, the user should be able to provide
parameters to the scripts themselves to customize their behavior.

### Examples

Don't do this:

```solidity
// Contracts that assume a file will be read from a specific location
contract FooScript is Script {
    function fun(Input input, Output output) public {
        string memory data = vm.readFile("../../foo.toml");
    }
}
```

Do this instead:

```solidity
contract FooScript is Script {
    function fun(Input input, Output output) public {
        string memory data = vm.readFile(input.filePath);
    }
}
```