/*
Package match provides matching presets and utils for selecting devstack components.

Matchers can be composed, e.g.:
- `And(OpGeth, WithLabel("name", "alice"))` to select the op-geth node named "alice".
- `Or(OpGeth, OpReth)` to select an op-geth or op-reth node.
- `Not(OpGeth)` to select anything but an op-geth node.

Custom matchers can also be implemented:
- MatchFn can filter a list of elements down to just the matched elements
- MatchElemFn can filter by checking each individual element

For convenience, aliases for common matchers are also provided,
e.g. for matching "chain A", or matching the first L2 EL node.
*/
package match
