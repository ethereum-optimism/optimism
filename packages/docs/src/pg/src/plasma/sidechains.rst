===========================================
Understanding: Plasma Chains vs. Sidechains
===========================================
People often talk about plasma chains and sidechains like they're the same thing.
Sometimes people even refer to plasma chains as sidechains.
However, **plasma chains and sidechains are very different!**
It's really important to understand these differences because plasma chains and sidechains make different promises about the safety of your funds.

Sidechains
==========
The Basics
----------
The idea of the "sidechain" was first popularized by `this paper published in 2014`_.
First applied mainly to Bitcoin, the sidechain concept was basically to run another blockchain *alongside* some other "main" blockchain.
These two blockchains could then talk to each other in a special way that made is possible for assets to move between the two chains.

Let's take a look at how this might look in the world of Ethereum.
If we want to create an Ethereum sidechain, we first have to create another blockchain.
We're going to create an Ethereum clone for the sake of this thought experiment.

Setting up an Ethereum clone is really simple.
You'd just need to run any standard Ethereum client (like `Parity`_ or `Geth`_) and set it up to create a new blockchain instead of connecting to an existing one.
You'd also need a "consensus mechanism," which basically just means you need a way to create new blocks.
In theory you could use Proof-of-Work, the same system Ethereum uses, but for now let's just assign ourselves the sole power to create blocks (basically "Proof-of-Authority").

Now you'd just need some way for assets to move between the two blockchains.
Usually, this is done by creating a smart contract on Ethereum.
When users want to move assets from Ethereum onto your sidechain, they deposit those assets into a smart contract sitting on Ethereum.
You'd watch for these deposits on Ethereum and re-create those assets on your sidechain.
Similarily, when users want to move assets from your sidechain back onto Ethereum, you delete those assets from your sidechain and allow the user to unlock the asset again on Ethereum.
It's really as simple as that!

The Pros
--------
If you think about what we just described, there's really no reason why the person who originally deposited some asset also has to be the same person to withdraw the asset.
This is what makes sidechains so cool -- assets can be moved around a lot before they're withdrawn.
Even though we might've made dozens of transactions on the sidechain, only two transactions (the deposit and the withdrawal) ever occur on Ethereum.
Since transactions on the sidechain are almost always cheaper than transactions on Ethereum, we get scalability!

The Cons
--------
If you think about the thing we just described, you might see some flaws.
Remember that we gave you the sole power to create new blocks.
What happens if you stop producing blocks altogether?
Or even worse, what happens if you stop allowing anyone to withdraw funds from the sidechain?

It's completely possible for you to do both of these things.
Usually this is somewhat mitigated by creating a sidechain with a more robust consensus mechanism.
For example, you could copy Ethereum's Proof-of-Work.

Unfortunately this still doesn't fix all of the problems with sidechains.
There's a reason why transactions on the sidechain are cheaper than transactions on Ethereum.
When you're paying fees on a blockchain, you're paying the miners who keep the blockchain secure.
Generally speaking, the more you pay in fees, the more security you get.
If the sidechain had just as much hash power as Ethereum (so the same level of security), transactions on the sidechain would cost pretty much the same as transactions on Ethereum.

All of this means that, in general, if a sidechain is cheaper than Ethereum then it's going to be (proportionally) less secure than Ethereum.
**If the sidechain fails (meaning the consensus mechanism gets compromised), you could lose all of your funds.**
So it's all about the amount of risk you're willing to take.
You might feel comfortable putting 1 ETH on a sidechain but not 100 ETH.

Plasma Chains
=============
The Basics
----------
Plasma chains were popularized by the `plasma paper published in 2016`_.
In a nutshell, plasma chains are *sort of* like sidechains, except they trade off some utility for extra security.

Just like sidechains, plasma chains have a consensus mechanism that creates blocks.
However, unlike sidechains, the "root" of each plasma chain block is published to Ethereum.
Block "roots" are basically little pieces of information that users can use to prove things about the contents of those blocks.
For example, a user could use a block root to prove that they made a transaction in that specific block.

.. todo::
    
    Write an article about how users can use a block root to prove things about the contents of that block.

The Pros
--------
Plasma chain block roots act sort of like "save points" in the blockchain.
Remember that one of the major cons of sidechains is that sidechain consensus mechanisms can stop producing blocks and lock everyone's funds up forever.
Since it's possible for users to use block roots to show that they received funds on the plasma chain, plasma doesn't have this problem!
If the plasma chain consensus mechanism stops creating blocks, users can use the block roots to make *claims* to Ethereum ("I claim I had 10 ETH on the plasma chain and I want to withdraw it.").

Effectively, this means that plasma chains are safer than sidechains by design.
Your funds are only ever at risk if *Ethereum* fails, but you probably have bigger problems.
Simply stated, a plasma chain is as secure as the main chain consensus mechanism, whereas as sidechain is only as secure as its own consensus mechanism.
This convenient property also means that the plasma chain can use really simple consensus mechanisms (like just a single authority!) and still be safe.

.. todo::

    Link out to the operator explainer page.

The Cons
--------
So plasma chains give us cheap transactions that are as secure as the main blockchain.
But what's the catch?

Well, when we're using a *sidechain* we have to trust the sidechain consensus mechanism.
If that mechanism fails, we're out of luck anyway.
That trust makes it possible to do really complex things because we also implicitly trust that the sidechain will be around in the future.

On a *plasma chain*, we keep funds more secure by not making that assumption.
We always have to assume that the plasma chain consensus mechanism could fail at any moment and need to design around that.
This adds extra restrictions to the things that are possible on a plasma chain.

Take, for example, a very long (let's say 1 year) timelock contract.
You could definitely put that contract on a sidechain if you trust that the sidechain will be around in a year.
But since we *don't* trust that the plasma chain will be around in year (even if it's the exact same consensus mechanism!), we need to think a little bit outside of the box.
We basically need to make sure that if the consensus mechanism fails, we have a way to move the *entire timelock contract* back onto Ethereum.
Luckily that's not so difficult, but it's more complex than it would be on the sidechain.

Things get really complex when it's not so clear how the thing on the plasma chain gets to move back onto Ethereum.
A timelock contract that's just holding your money makes sense because it seems obvious that *you* should be able to move the contract.
But what if we're talking about a timelock contract that's holding money for 100 people at once?
Now it's not so clear anymore.

Put simply, the major con of plasma chains is that you can't really do the same complex operations that you could do on sidechains.
Importantly, though, the *reason* you can't do these complex things is because you're taking more precautions in order to ensure that your funds stay safe.

.. _`this paper published in 2014`: https://blockstream.com/sidechains.pdf
.. _`Parity`: https://www.parity.io/
.. _`Geth`: https://github.com/ethereum/go-ethereum/wiki/geth
.. _`plasma paper published in 2016`: http://plasma.io/
