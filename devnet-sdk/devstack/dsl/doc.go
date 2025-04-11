/*
Package dsl provides DSL (domain specific language) to interact with a devstack system.

Each component in the devstack has a DSL wrapper.
The wrapper itself does not have any state, and may be recreated or shallow-copied.

Each DSL wrapper provides an Escape method, in case the DSL is not sufficient for a given use-case.
The Escape method usage should be kept to a minimum, for readability and maintainability.
*/
package dsl
