# Introduction

The devnet-sdk DSL is a high level test library, specifically designed for end to end / acceptance testing of the
OP Stack. It aims to make the development and maintenance of whole system tests faster and easier.

The high level API helps make the actual test read in a more declarative style and separate the technical details of how
an action is actually performed. The intended result is that tests express the requirements, while the DSL provides the
technical details of how those requirements are met. This ensures that as the technical details change, the DSL can
be updated rather than requiring that each test be updated individual - significantly reducing the maintenance cost for
a large test suite. Similarly, if there is flakiness in tests, it can often be solved by improving the DSL to
properly wait for pre or post conditions or automatically perform required setup steps and that fix is automatically
applied everywhere, including tests added in the future.

## Guiding Principles

These guiding principles allow the test suite to evolve and grow over time in a way that ensures the tests are
maintainable and continue to be easy to write. With multiple different teams contributing to tests, over a long time
period, shared principles are required to avoid many divergent approaches and frameworks emerging which increase the
cognitive load for developers writing tests and increase the maintenance costs for existing tests.

### Keep It Simple

Avoid attempting to make the DSL read like plain English. This is a domain-specific language and the domain experts are
actually the test developers, not non-technical users. Each statement should clearly describe what it is trying to do,
but does not need to read like an English sentence.

Bias very strongly towards making the tests simpler, even if the DSL implementation then needs to be more complex.
Complexity in tests will be duplicated for each test case whereas complexity in the DSL is more centralised and is
encapsulated so it is much less likely to be a distraction.

### Consistency

The "language" of the DSL emerges by being consistent in the structures and naming used for things. Take the time to
refactor things to ensure that the same name is used consistently for a concept right across the DSL.

Bias towards following established patterns rather than doing something new. While introducing a new pattern might make
things cleaner in a particular test, it introduces additional cognitive load for people to understand when working with
the tests. It is usually (but not always) better to preserve consistency than to have a marginally nicer solution for
one specific scenario.

The [style guide](./style_guide.md) defines a set of common patterns and guidelines that should be followed.
