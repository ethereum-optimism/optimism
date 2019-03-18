=====================
Plasma Chain Operator
=====================
When learning about plasma you’ll eventually run into the idea of the plasma chain ‘operator’.
For the most part, the operator is exactly what it sounds like — a single entity that’s responsible for aggregating transactions into blocks and then publishing those blocks to some other “main” blockchain (like Ethereum).
Blockchains are only usable if new blocks are being created, so, by producing these new blocks, the operator is quite literally responsible for keeping the whole plasma chain running. 

This might be a little confusing at first.
Initial reactions are usually something along the lines of, “What? Plasma chains have operators? Aren’t blockchains supposed to be decentralized?”
Well, it turns out that the answer is, as you might’ve guessed, “Kinda.”

Back to Basics
==============
Let’s go back to basics and talk a little bit about why most blockchains use decentralized block production mechanisms in the first place.
Blockchains are big logs of things that have happened, broken into concrete and ordered time-steps we call blocks.
Sometimes these “things that have happened” are simple - “A sent X amount of money to B” - sometimes they’re more complex - “Kitty A mated with Kitty B and created Kitty C”.
No matter what these events are, we usually want to make sure that we have a few key properties:

1. No one should be able to re-write history.
2. No one should be blocked from making transactions.

Let’s imagine we have a blockchain run by a single person.
For the sake of argument, assume that you *must* use the blockchain run by that person for some reason.
Well, unfortunately it’s quite easy for that person to break the first property.
If that person says “transaction X happened” and then later says “transaction X never happened”, there’s not much you can really do.
*You* know that the transaction happened, but the blockchain itself doesn’t. 

It’s also really easy for that person to break the second property.
If they don’t want you to send transactions, they can just refuse to add any transactions that come from you.
Had a bunch of money on that blockchain? Too bad, you’re not getting it back. 

Plasma Magic
============
The problems with blockchains run by a single person are why blockchains usually have fancy mechanisms that ensure that it’s extremely expensive to rewrite history.
It's also why blockchains usually have lots of different people who can create blocks -- no single person can stop someone from making transactions.
So why can we have a single person running plasma chain?
It’s because we cheat (sort of).

In plasma world, we get the first property by taking advantage of the “main blockchain” that we were talking about earlier.
Plasma chain operators need to publish a block “commitment” (sort of like a very compressed version of the block) to the main blockchain for every block they produce.
A smart contract on the main blockchain ensures that the operator can never publish the same block twice.
As long as the main blockchain has the first property, so does the plasma chain!
There’s no way for the operator to re-write history unless they can re-write history on the main chain.

The second property (blocking users from transacting) is where things get interesting.
Unfortunately, it’s still possible for the operator to censor transactions from anyone they want.
*However*, this is where the magic of plasma comes in.
Plasma chains are designed in a way that **no matter what, a user can always withdraw their money** from the plasma chain back to the main chain.
Even if the operator is actively trying to steal money from you, you’ll still be able to get it back.
Being censored obviously isn’t great, but it’s not as bad when you can always take your money somewhere else. 

Decentralizing the Operator
===========================
The one thing that wasn’t really mentioned here is the fact that the “operator” can actually consist of multiple people making decisions about what blocks to publish.
This could be as simple as a having a few designed people who take turns making blocks, or as complex as a Proof-of-Stake system that selects block producers randomly.
Either way, it's complicated than just having a single person run everything, but it’s probably the way to go for projects that want to get rid of censorship.
At the same time, it tends to be unimportant research-wise whether the operator is a single person or many people.
As a result, you'll often see people just assuming the operator is a single person for simplicity.
