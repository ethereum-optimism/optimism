# watch-eth
`watch-eth` is a robust library for watching Ethereum events.
Applications sometimes make critical decisions based on Ethereum events.
It's therefore important that these applications have a way to watch for events that won't fail.

## Installation
### npm
You can install `watch-eth` via `npm`:

```
npm install --save watch-eth
```

## Usage
Start by creating an `EventWatcher` instance for your contract:

```js
const { EventWatcher } = require('watch-eth')

const watcher = new EventWatcher({
  address: '0xc8a5ba5868a5e9849962167b2f99b2040cee2031',
  abi: [{"anonymous":false,"inputs":[{"indexed":false,"name":"_value","type":"uint256"}],"name":"TestEvent","type":"event"}],
  finalityDepth: 12   // optional
  pollInterval: 10000 // optional
})
```

Now you can subscribe or unsubscribe to an event:

```js
const listener = () => {
  console.log('Detected TestEvent!')
}

watcher.subscribe('TestEvent', listener)
// Stuff
watcher.unsubscribe('TestEvent', listener)
```

## Design
### Modularity
`watch-eth` is designed to be modular, meaning most of the components can be replaced with custom logic.
For example, if you want to use `ethers` for connecting to Ethereum instead of `web3`, you can simply replace `DefaultEthProvider` with a custom class.

### Event Validity Conditions
`watch-eth` will only relay an Ethereum event if it satisfies the following conditions:

1. Event has not been seen before.
2. Event is under a certain number of blocks.
