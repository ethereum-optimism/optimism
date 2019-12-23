# Rollup Dev Tools
Package that contains developer tools for building on optimistic rollup.

# Dependencies
Please refer to the root README of this repo.

# Setup
Run `yarn install` to install necessary dependencies.

# Building
Run `yarn build` to build the code. Note: `yarn all` may be used to build and run tests.

# Testing
Run `yarn test` to run the unit tests.

# Tools
## Transpiler
Enables transpilation of L1 contracts to L2 bytecode.

### Configuration
The transpiler is configured via the `config/default.json` file, which should not be changed.
 
Sensitive config values and overrides can be configured in `config/.env` which will not be versioned.

See: `config/.env.example` for more info.

### Execution
The transpiler is executed by running

`yarn transpile <input path> <output path>`
or
`yarn transpile <bytecode hex string>`
