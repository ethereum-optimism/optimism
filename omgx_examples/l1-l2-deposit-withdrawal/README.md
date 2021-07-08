# OMGX L1-L2 Deposit/Withdrawal

## 1. Compiling the contracts

First, spin up a local system as usual (see the top-level `Readme.md`). Then, run:

```bash
$ yarn
$ yarn compile       #for your local L1
$ yarn compile:ovm   #for your local L2
$ yarn compile:omgx  #for OMGX Rinkeby
```

## 2. Testing

```bash
$ yarn test:integration:ovm  #for your local L1/L2
$ yarn test:integration:omgx #for OMGX Rinkeby
```

```bash
% yarn test:integration:omgx

L1 <> L2 Deposit and Withdrawal
    Initialization and initial balances
      ✓ should have initial L1 balance of 1234 and initial L2 balance of 0 (40ms)
    L1 to L2 deposit
      ✓ should approve 1234 tokens for ERC20 bridge (107ms)
      ✓ should deposit 1234 tokens into L2 ERC20 (4270ms)
      ✓ should relay deposit message to L2 (4085ms)
      ✓ should have changed L1 balance to 0 and L2 balance to 1234
    L2 to L1 withdrawal
      ✓ should approve 1234 tokens for ERC20 bridge (98ms)
      ✓ should withdraw tokens back to L1 ERC20 and relay the message (179ms)
      ✓ should relay withdrawal message to L1 (4094ms)
      ✓ should have changed L1 balance back to 1234 and L2 balance back to 0


  9 passing (13s)

✨  Done in 14.33s.
```