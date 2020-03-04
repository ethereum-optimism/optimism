===========
Limitations
===========

Some features of the Ethereum are not yet implemented, or just don't make sense to have, in the OVM.  This page documents some of those differences.

No Native ETH
-------------
On L1, sending/recieving ETH is a special opcode different from ERC20s.  This often leads to developers having to implement their contract functionality twice.  For the OVM, we decided to eliminate this complexity and enforce that ETH only exist as an ERC20 (WETH).  We think this will make developers' lives easier--but if you feel strongly otherwise, reach out and let us know!

Block Number
-------------
Many L2 constructions, including our rollup implementation, support instant, real-time transactions.  One way this manifests is that ``block.number`` is not easily interpretable or specified in the OVM.  Instead, contracts can use ``block.timestamp`` (though right now, this is not implemented so it stays at 0.)

Solidity libraries
-------------
Our transpiler does not currently support the usage of libraries in solidity -- this is one of the next things on our chopping block, so stay tuned!  In the meantime you can convert the libs into a regular smart contract.

Parent/Child chain communication
-------------
Communication between L1 and L2, also known as deposits and withdrawals, are not yet implemented in the OVM.  Stay tuned for more on this!