package resources

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface{}

// SupernodeAuthority is the supernode's implementation of SuperAuthority.
// It provides coordination and authority functions to op-node instances
// running within the supernode.
type SupernodeAuthority struct{}

// NewSupernodeAuthority creates a new SupernodeAuthority instance.
func NewSupernodeAuthority() *SupernodeAuthority {
	return &SupernodeAuthority{}
}

// Interface conformance assertion
var _ SuperAuthority = (*SupernodeAuthority)(nil)
