# @eth-optimism/hardhat-ovm

A plugin that brings OVM compiler support to Hardhat projects.

## Installation

```
yarn add --dev @eth-optimism/hardhat-ovm
```

Next, import the plugin inside your `hardhat.config.js`:

```js
// hardhat.config.js

require("@eth-optimism/hardhat-ovm")
```

Or if using TypeScript:

```ts
// hardhat.config.ts

import "@eth-optimism/hardhat-ovm"
```

## Configuration

**By default, this plugin will use OVM compiler version 0.7.6**.
Configure this plugin by adding an `ovm` field to your Hardhat config:

```js
// hardhat.config.js

require("@eth-optimism/hardhat-ovm")

module.exports = {
    ovm: {
        solcVersion: 'X.Y.Z' // Your version goes here.
    }
}

```

This package also has typings so it won't break your Hardhat config if you're using TypeScript.
