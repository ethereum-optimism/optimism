/*
Package dsl provides DSL (domain specific language) to interact with a devstack system.

Each component in the devstack has a DSL wrapper.
The wrapper itself does not have any state, and may be recreated or shallow-copied.

Each DSL wrapper provides an Escape method, in case the DSL is not sufficient for a given use-case.
The Escape method is a temporary compromise to allow more incremental development of and
migration to the DSL. It should be avoided whenever possible and will be removed in the future.
*/
package dsl
