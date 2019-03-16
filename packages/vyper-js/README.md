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
