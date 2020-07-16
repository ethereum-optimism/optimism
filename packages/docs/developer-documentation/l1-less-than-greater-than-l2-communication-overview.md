# L1&lt;&gt;L2 Communication Overview

Our implementation will offer the following abstraction for implementing L1&lt;&gt;L2 communication.

The most common L1&lt;&gt;L2 Messaging use case is deposits and withdrawals. A message is sent from L1 to L2 to deposit, and sent back from L2 to L1 to withdraw.

### API

Here are roughly the functions you get access to \(ignoring any of the "fluff" arguments like nonces, etc. that are there in practice\):

#### L1 contracts can:

* `sendL1ToL2Message(calldata: bytes, targetL2Contract: address)` - sends a message the specified some contract on L2.  This will trigger a transaction with that `calldata` on L2.
* `verifyL2ToL1Message(data: bytes, L2MessageSender: address) returns(bool)` - Verifies that the specified message was sent by the specified contract on L2, and the dispute period has elapsed.

#### L2 contracts can:

* `getL1TxOrigin() returns(address)` - gets the address of the `msg.sender` \(L1 address\) who called `sendL1ToL2Message(...)`, triggering the L2 transaction.
* `sendL2ToL1Message(data: bytes)` - sends a message from L2 to L1, will be verifiable on L1 once the dispute period has elapsed.

### Pseudocode For Deposits and Withdrawals

Here is how you use the above messaging functions to implement deposit/withdrawals. You have two contracts, one on L1, and the other on L2. I will write out just the parts of each contract associated with deposits, and then do the same for withdrawals.

#### Deposits

**L1 contract**

```text
contract L1DepositWithdrawalContract {
    // the rollup contract to handle deposits/withdraws with
    address const ROLLUP_CONTRACT: 0x...;

    // L1 address of the ERC20 
    address const L1_ERC20_TOKEN: 0x...;

    // L2/ovm address of the same ERC20 for use in L2
    address const L2_ERC20_TOKEN_EQUIVALENT: 0x...;

    function depositIntoL2(uint value) {
        // store the deposited funds in this contract
        L1_ERC20_TOKEN_ADDRESS.transferFrom(
            msg.sender,
            address(this)
            value,
        )
        // now we tell the L2 contract about the deposit so it can mint equal funds there 
        // Generate the L1->L2 msg calldata
        const calldata = concat(
            getMethodId('processDeposit(uint,address)'),
            abi.encode(value, msg.sender)
        )
        // Send the L1->L2 msg
        ROLLUP_CONTRACT.sendL1ToL2Message(
            calldata,
            L2_ERC20_TOKEN_EQUIVALENT
        )
    }

    ...[withdrawal logic below]
}
```

**L2 Contract**

This OVM contract would be deployed at `L2_ERC20_TOKEN_EQUIVALENT` going by the above code. Assume it extends a base ERC20 class which allows `mint()`ing and `burn()`ing coins to particular addresses.

```text
import {OvmUtils} from 'somewhere';

contract L2DepositWithdrawalContract is ERC20WithMintAndBurn {
    // This would be the delpoyed L1 contract above
    address const L1_DEPOSIT_CONTRACT = 0x...;

    function processDeposit(
        uint amount,
        address depositer
    ) public {
        // authenticate this is really a deposit
        address l1TxOrigin = OvmUtils.getL1TxOrigin();
        require(l1TxOrigin == L1_DEPOSIT_CONTRACT, 'Only the deposit contract may mint this bridged L2 token.');
        // mint the L2 version of the token to depositer
        mint(amount, depositer);
    }

    ...[withdrawal logic below]
}
```

**Notes/explanation**

As you can see, only when some L2 token is locked in the L1 contract, does the L2 contract mint some new coins. This would enforce that the supply of the L2 token is exactly equal to the supply of locked L1 tokens.

#### Withdrawals

Withdrawals are basically the same thing in reverse! Starting with the L2 contract this time:

**L2 Contract**

```text
import {OvmUtils} from 'somewhere';

contract L2DepositWithdrawalContract is ERC20WithMintAndBurn {
    ...[deposit logic above]

    function withdrawToL1(amount) public {
        address withdrawer = msg.sender;
        // we need to tell the L1 contract who is withdrawing, and how much.
        bytes memory data = concat(withdrawer, amount);
        // send off the message to L1
        OvmUtils.sendL2ToL1Message(data);
        // Now that it will be unlockable on L1, it must be burned here
        burn(withdrawer, amount);
        // that's it, just wait the dispute period to reedeem!
    }

}
```

**L1 Contract**

```text
contract L1DepositWithdrawalContract {
    ...[deposit logic above]

    function redeemWithdrawal(
        uint amount        
    ) public {
        address withdrawer = msg.sender;
        // if the withrawal is permitted, this is what should've been sent.
        bytes memory expectedData = concat(withdrawer, amount);
        // verify that the correct L2 contract did indeed authenticate this withdrawal
        ROLLUP_CONTRACT.verifyL2ToL1Message(
            expectedData,
            L2_ERC20_TOKEN_EQUIVALENT
        )
        // send to the withdrawer
        L1_ERC20_TOKEN.transfer(amount, withdrawer)
    }
}
```

