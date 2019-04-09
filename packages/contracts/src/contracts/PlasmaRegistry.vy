# External Contracts
contract PlasmaChain:
    def setup(operator: address, ethDecimalOffset: uint256): modifying

# Events
NewPlasmaChain: event({
    _name: indexed(bytes32),
    _address: address,
    _operator: address,
    _metadata: bytes32
})

# Variables
plasmaChainTemplate: public(address)
plasmaChains: public(map(bytes32, address))

@public
def __init__(_template: address):
    """Creates the registry.

    Args:
        _template (address): Address of the contract to use as
            the template for chains created through this registry.

    """
    self.plasmaChainTemplate = _template

@public
def createPlasmaChain(_name: bytes32, _operator: address, _metadata: bytes32) -> address:
    """Creates a new plasma chain and registers it.

    Plasma chains in the registry must be created through the registry
    to ensure that they all share the same code. We then store
    the address of the new chain in the registry and emit an event.

    Args:
        _name (bytes32): Name of the new plasma chain. Must be unique.
        _operator (address): Address of the plasma chain operator.
        _metadata (bytes32): Arbitrary additional metadata.
    
    Returns:
        address: Address of the new plasma chain.

    """
    assert self.plasmaChains[_name] == ZERO_ADDRESS

    plasmaChain: address = create_with_code_of(self.plasmaChainTemplate)
    PlasmaChain(plasmaChain).setup(_operator, 0)
    self.plasmaChains[_name] = plasmaChain

    log.NewPlasmaChain(_name, plasmaChain, _operator, _metadata)
    return plasmaChain
