=============================
Smart Contract and Exit Games
=============================
The proof for a chain of custody isn't useful unless it can also be passed to the main chain to keep funds secure.
The mechanism which accepts proofs on-chain is the core of plasma's security model.
We usually call this mechanism an "exit game".

When a user wants to move their money from plasma chain back to Ethereum, they start an "exit".
However, the user doesn't get their funds immediately.
Instead, each exit has to stand a "dispute period".

During the dispute period, users may submit "challenges" which claim the money being withdrawn isn't rightfully owned by the person who started the exit.
If the person who started the exit *does* in fact own the money, they are always able to calculate and present a "challenge response" which closes the challenge.
If there are no outstanding disputes at the end of the dispute period, the user can finalize their exit and receive the money.

The goal of the exit game is to keep user assets secure, even in the case of a malicious operator.
Particularly, there are three main attacks which we must mitigate:

- **Data withholding:** the operator may publish a root hash to the contract, but not tell anybody what the contents of the block are.
- **Invalid transactions:** the operator may include a transaction in a block whose ``sender`` was not the previous ``recipient`` in the chain of custody.
- **Censorship:** the operator may refuse to publish any transactions from a specific user.

In all of these cases, the challenge/response protocol of the exit game ensures that these behaviors do not allow theft.
Importantly, we also ensure that each challenge can be closed in at most one response.

Keeping Track of Deposits and Exits
===================================

Deposits Mapping
----------------
Each time a new set of coins is deposited, the contract updates a mapping that contains ``deposit`` structs.

From the contract:

.. code-block:: python

    struct deposit:
      untypedStart: uint256
      depositer: address
      precedingPlasmaBlockNumber: uint256

Note that this struct contains neither the ``untypedEnd`` or ``tokenType`` for the deposit.
That's because the contract uses those values as the keys in a mapping of mappings.
For example, to access the depositer of a given deposit, we can query ``deposits`` directly:

.. code-block:: python

    someDepositer: address = self.deposits[tokenType][untypedEnd].depositer

This choice was made to save gas and to simplify the smart contract.
Namely, it means that we don't need to store any sort of deposit ID in order to reference a deposit.

Exitable Ranges Mapping
-----------------------
The contract also needs to keep track of finalized exits in order prevent multiple exits on the same range.
This is a little trickier because exits don't occur in order like deposits do, and it'd be too expensive to search through a giant list of exits.

Our contract implements a constant-sized solution, which instead stores a list of "exitable ranges".
This list is updated as new exits occur.

From the smart contract:

.. code-block:: python

    struct exitableRange:
      untypedStart: uint256
      isSet: bool

Again, we use a double-nested mapping with keys ``tokenType`` and ``untypedEnd`` so that we may call ``self.exitable[tokenType][untpyedEnd].untypedStart`` to access the start of the range.
Note that Vyper returns ``0`` for all unset mapping keys, so we need an ``isSet`` bool so that users may not "trick" the contract by passing an unset ``exitableRange``.

The contract's ``self.exitable`` ranges are split and deleted based on successful calls to ``finalizeExit`` via a helper function called ``removeFromExitable``.
Note that exits on a previously exited range do not even need to be challenged; they'll never pass the ``checkRangeExitable`` function called in ``finalizeExit``.
You can find that code `here`_.

Similarities to Plasma Cash
===========================
At heart, the exit games in our spec are very similar to the original Plasma Cash design.
Exits are initiated with calls to:

.. code-block:: python

    beginExit(tokenType: uint256, blockNumber: uint256, untypedStart: uint256, untypedEnd: uint256) -> uint256

All exit challenges specify a particular coin ID, and a Plasma Cash-style challenge game is carried out on that particular coin.
Only a single coin needs to be proven invalid to cancel the entire exit.

Both exits and challenges are assigned a unique ``exitID`` and ``challengeID``.
These IDs are assigned in order based on an incrementing ``challengeNonce`` and ``exitNonce``.

Block-specific Transactions
===========================
In the original Plasma Cash spec, the exiter is required to specify both the exited transaction and its previous "parent" transaction to prevent the "in-flight" attack.
This attack occurs when the operator delays inclusion of a valid transaction and then inserts an invalid transaction before the valid one.

This poses a problem for our range-based schemes because a transaction may have multiple parents.
For example, if Alice sends ``(0, 50)`` to Carol, and Bob sends ``(50, 100)`` to Carol, Carol can now send ``(0, 100)`` to Dave.
If Dave wants to exit ``(0, 100)``, he would need to specify both ``(0, 50)`` and ``(50, 100)`` as parents.

If a range has dozens or even hundreds of parents, it becomes basically impossible to publish all of these parents on chain.
Instead, we opted for a simpler alternative in which each transaction specifies the block in which it should be included.
If the transaction is included in a different block, it's no longer valid.
This solves the in-flight attack because it becomes impossible for the operator to delay inclusion of the transaction.

This does, unfortunately, introduce one downside -- if a transaction isn't included in the specified block (for whatever reason), it needs to be re-signed and re-submitted.
Hopefully this won't happen too often in practice, but it's something to think about.

For those interested in a formal writeup and safety proof for this scheme, it's worth giving `this great post`_ a look.

Per-coin Transaction Validity
=============================
An unintuitive property of our exit games worth that's noting up front is that a certain transaction might be "valid" for some of the coins in its range, but not for others.

For example, imagine that Alice sends ``(0, 100)`` to Bob, who in turn sends ``(50, 100)`` to Carol.
Carol doesn't need to verify that Alice was the rightful owner of the full ``(0, 100)``.
Carol only needs an assurance that Alice owned ``(50, 100)`` -- the part of the custody chain which applies to Carol's range.

Though the transaction to Dave might in a sense be "invalid" if Alice didn't own ``(0, 50)``, the smart contract doesn't care for the purposes of disputes on exits for the coins ``(50, 100)``.
As long as the received coins are valid, invalid transactions on any other coins don't matter.

This is a **very important requirement** to preserve the size of light client proofs.
If Carol had to check the full ``(0, 100)``, she might also have to check an overlapping parent of ``(0, 10000)``, and then all of its parents, and so on.
This "cascading" effect could massively increase the size of proofs if transactions were very interdependent.

Note that this property also applies to atomic multisends, in which multiple ranges are *swapped*.
If Alice trades 1 ETH for Bob's 1 DAI, it is Alice's responsibility to check that Bob owns the 1 DAI before signing.
However, after, if Bob then sends the 1 ETH to Carol, Carol need not verify that Bob owned the 1 DAI, only that Alice owned the 1 ETH she sent to Bob.
Alice incurred the risk, so Carol doesn't have to.

From the standpoint of the smart contract, this property is a direct consequence of challenges always being submitted for a particular ``coinID`` within the exit.

Transaction Verification
========================
Only funds that came from valid transactions can be withdrawn.
We can check the validity of a transaction at the contract level via:

.. code-block:: python

    def checkTransactionProofAndGetTypedTransfer(
      transactionEncoding: bytes[277],
      transactionProofEncoding: bytes[1749],
      transferIndex: int128
    ) -> (
      address, # transfer.to
      address, # transfer.from
      uint256, # transfer.start (typed)
      uint256, # transfer.end (typed)
      uint256  # transaction.blockNumber
    )
 
An important feature here is the ``transferIndex`` argument.
Remember that a transaction may contain multiple transfers and that the transaction must be included in the tree once for each transfer.
However, since challenges refer to a specific ``coinID``, only a single transfer will be relevant.
As a result, challengers and responders have to give a ``transferIndex`` -- a reference to the index of the relevant transfer.

Once we decode the ``TransactionProof``, we can check the relevant ``TransferProof``:
 
.. code-block:: python

    def checkTransferProofAndGetTypedBounds(
      leafHash: bytes32,
      blockNum: uint256,
      transferProof: bytes[1749]
    ) -> (uint256, uint256)

Challenges That Immediately Block Exits
=======================================
Two kinds of challenges immediately cancel exits: those that show a specific coin is already spent, and those that show an exit comes before the deposit.

Spent-Coin Challenge
--------------------
This challenge is used to demonstrate that coins being withdrawn have already been spent.

.. code-block:: python

    @public
    def challengeSpentCoin(
      exitID: uint256,
      coinID: uint256,
      transferIndex: int128,
      transactionEncoding: bytes[277],
      transactionProofEncoding: bytes[1749],
    )

It uses ``checkTransactionProofAndGetTypedTransfer`` and then checks the following:

1. The challenged coinID lies within the specified exit.
2. The challenged coinID lies within the ``typedStart`` and ``typedEnd`` of the ``transferIndex``th element of ``transaction.transfers``.
3. The ``plasmaBlockNumber`` of the challenge is greater than that of the exit.
4. The ``transfer.sender`` is the exiter.

The introduction of atomic swaps does mean one thing: the spent coin challenge period must be strictly less than others.
There's an edge case in which the operator withholds an atomic swap between two or more parties.
Those parties must exit their coins from *before* the swap because they don't know if the swap was included.
If the swap was not included, then these exits will finalize successfully.
However, if the swap *was* included, then operator can submit a Spent-Coin Challenge and block these exits.

If we allowed the operator to submit this challenge at the last minute, we'd be creating a race condition in which the parties have no time to use the newly revealed information to cancel other exits.
Thus, the timeout is made shorter (1/2) than the regular challenge window, eliminating "last-minute response" attacks.

Before-Deposit Challenge
------------------------
This challenge is used to demonstrate that an exit comes from a ``plasmaBlockNumber`` earlier than the coin's deposit.

.. code-block:: python

    @public
    def challengeBeforeDeposit(
      exitID: uint256,
      coinID: uint256,
      depositUntypedEnd: uint256
    )

The contract looks up ``self.deposits[self.exits[exitID].tokenType][depositUntypedEnd].precedingPlasmaBlockNumber`` and checks that it's is later than the exit's block number.
If so, it cancels the exit immediately.

Optimistic Exits and Inclusion Challenges
=========================================
Our contract allows an exit to occur without actually checking that the transaction referenced in the exit was included in the plasma chain.
This is called an "optimistic exit," and allows us to reduce gas costs for users who are behaving honestly.
However, this means that it's possible for someone start an exit from a transaction that never happened.

As a result, we expose a way for someone to challenge this type of exit:

.. code-block:: python

    @public
    def challengeInclusion(exitID: uint256)

Then, the user who started the exit can respond by showing that the transaction or deposit from which they are exiting really did happen:

.. code-block:: python

    @public
    def respondTransactionInclusion(
      challengeID: uint256,
      transferIndex: int128,
      transactionEncoding: bytes[277],
      transactionProofEncoding: bytes[1749],
    )
    ...
    @public
    def respondDepositInclusion(
      challengeID: uint256,
      depositEnd: uint256
    )

We need this special second case so that users can withdraw money even if the operator is censoring all transactions after their deposit.

Both responses cancel the challenge if:
1. The deposit or transaction was indeed at the exit's plasma block number.
2. The depositer or recipient is indeed the exiter.
3. The start and end of the exit were within the deposit or transfer's start and end

Invalid-History Challenge
=========================
The Invalid-History Challenge is the most complex challenge-response game in both vanilla Plasma Cash and this spec.
This part of the protocol mitigates the attack in which the operator includes an forged "invalid" transaction whose sender is not the previous recipient.

Effectively, this challenge allows the rightful owner of a coin to request that the exiter provide a proof that the owner has spent their funds.
The idea here is that if the rightful owner really is the rightful owner, then the exiter will not be able to provide such a transaction.

Both invalid history challenges and responses can be either deposits or transactions.

Challenging
-----------
There are two ways to challenge, depending on the current rightful owner:

.. code-block:: python

    @public
    def challengeInvalidHistoryWithTransaction(
      exitID: uint256,
      coinID: uint256,
      transferIndex: int128,
      transactionEncoding: bytes[277],
      transactionProofEncoding: bytes[1749]
    )

and

.. code-block:: python

    @public
    def challengeInvalidHistoryWithDeposit(
      exitID: uint256,
      coinID: uint256,
      depositUntypedEnd: uint256
    )

Both of these methods call an additional method, ``challengeInvalidHistory``:

.. code-block:: python

    @private
    def challengeInvalidHistory(
      exitID: uint256,
      coinID: uint256,
      claimant: address,
      typedStart: uint256,
      typedEnd: uint256,
      blockNumber: uint256
    )

This method does the legwork of checking that the ``coinID`` is within the challenged exit, and that the ``blockNumber`` is earlier than the exit.

Responding
----------
Of course it's also possible for someone to submit a fraudulent Invalid-History Challenge.
Therefore we give exiters two ways to respond to this type of challenge.

The first is to respond with a transaction showing that the challenger did, in fact, spend their money:

.. code-block:: python

    @public
    def respondInvalidHistoryTransaction(
      challengeID: uint256,
      transferIndex: int128,
      transactionEncoding: bytes[277],
      transactionProofEncoding: bytes[1749],
    )

The smart contract then performs the following checks:
1. The ``transferIndex``th ``Transfer`` in the ``transactionEncoding`` covers the challenged ``coinID``.
2. The ``transferIndex``th ``transfer.sender`` was indeed the claimant for that invalid history challenge.
3. The transaction's plasma block number lies between the invalid history challenge and the exit.

The second response is to show the challenge came *before* the coins were actually deposited - making the challenge invalid.
This is similar to a ``challengeBeforeDeposit``, but for the exit itself.

.. code-block:: python

    @public
    def respondInvalidHistoryDeposit(
      challengeID: uint256,
      depositUntypedEnd: uint256
    )

In this case, there is no check on the sender being the challenge recipient, since the challenge was invalid.
So the contract just needs to check:
1. The deposit covers the challenged ``coinID``.
2. The deposit's plasma block number lies between the challenge and the exit.

If all of these conditions are true, the exit is cancelled.

.. _here: https://github.com/plasma-group/plasma-contracts/blob/068954a8584e4168daf38ebeaa3257ec08caa5aa/contracts/PlasmaChain.vy#L380
.. _this great post: https://ethresear.ch/t/plasma-cash-with-smaller-exit-procedure-and-a-general-approach-to-safety-proofs/1942
