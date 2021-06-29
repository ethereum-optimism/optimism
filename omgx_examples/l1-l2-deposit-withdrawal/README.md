# OMGX First Steps

## 1. To deploy on a LOCAL system

First, spin up a local system as usual (see the top-level Readme.md). Then, run:

```bash
$ yarn compile
$ yarn compile:ovm
```

```
$ hardhat compile --network optimism
Compiling 7 files with 0.7.6
contracts/L2DepositedERC20.sol:44:9: Warning: Unused function parameter. Remove or comment out the variable name to silence this warning.
        address _to,
        ^---------^

Compilation finished successfully
✨  Done in 6.26s.

```

Now, see if it works

```bash
$ yarn test:integration:ovm
```

```bash
yarn run v1.22.10
$ hardhat test --network optimism


  ERC20
    ✓ should have a name
    ✓ should have a total supply equal to the initial supply (42ms)
    ✓ should give the initial supply to the creator's address
    transfer(...)
      ✓ should revert when the sender does not have enough balance (96ms)
      ✓ should succeed when the sender has enough balance (235ms)
    transferFrom(...)
      ✓ should revert when the sender does not have enough of an allowance (38ms)
      ✓ should succeed when the owner has enough balance and the sender has a large enough allowance (181ms)


  7 passing (3s)

✨  Done in 5.48s.
```

## 2. To test on OMGX Rinkeby

First, run:

```bash
$ yarn compile:omgx
```

This will compile the contract, Now, see if it works

```bash
$ yarn test:integration:omgx
```

```bash
yarn run v1.22.10
$ hardhat test --network omgx_rinkeby


  L1 <> L2 Deposit and Withdrawal
    Initialization and initial balances
      ✓ should initialize L2 ERC20 (111ms)
      ✓ should have initial L1 balance of 1234 and initial L2 balance of 0 (44ms)
    L1 to L2 deposit
      ✓ should approve 1234 tokens for ERC20 gateway (132ms)
      ✓ should deposit 1234 tokens into L2 ERC20 (1318ms)
      ✓ should relay deposit message to L2 (4061ms)
      ✓ should have changed L1 balance to 0 and L2 balance to 1234 (39ms)
    L2 to L1 withdrawal
      ✓ should withdraw tokens back to L1 ERC20 and relay the message (228ms)
      ✓ should relay withdrawal message to L1 (4118ms)
      ✓ should have changed L1 balance back to 1234 and L2 balance back to 0


  9 passing (11s)

✨  Done in 14.05s.
```
