![cannon](https://upload.wikimedia.org/wikipedia/commons/8/80/Cannon%2C_Château_du_Haut-Koenigsbourg%2C_France.jpg)

The cannon (cannon cannon cannon) is an on chain interactive fraud prover

It's half geth, half of what I think truebit was supposed to be. When it's done, we'll be able to prove L1 blocks aren't fraud

1. Get minigeth to verify a block locally paying attention to oracle
2. Compile embedded minigeth 
3. Get embedded minigeth to verify a block using the oracle

...and then there's more stuff, but just that for now

## Usage
```
# verify the transition from 13284469 -> 13284470
./run.sh
```

## State Oracle API

Preimage(hash) -> value    # hash(value) == hash

PrefetchAccount, PrefetchStorage, PrefetchCode can be NOP in the VM

## License

All my code is MIT license, minigeth is LGPL3. Being developed under contract for @optimismPBC

# Very important TODO

TODO: update to picture of increasingly futuristic cannon as it starts to work
