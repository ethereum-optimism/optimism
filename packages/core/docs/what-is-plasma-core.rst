====================
What is plasma-core?
====================
``plasma-core`` makes up the core of the Plasma Group node ecosystem.
It contains almost all of the functionality that a full plasma node needs.
``plasma-core`` handles things like watching Ethereum, keeping the local state up to date, and talking to the operator.
A full list of services that ``plasma-core`` provides is documented in our architecture_ page.

``plasma-core`` is **not** a full plasma node!
This means that you'll need to `extend plasma-core`_ to expose the full set of necessary functionality.
However, we've already done this for you!
We've built out two different full plasma nodes using ``plasma-core`` as a backend.
Our most user-friendly node is ``plasma-extension``, a full plasma node inside a Chrome extension!

**Note**: ``plasma-extension`` is still under construction.
