contract PlasmaChain():
    def setup(operator: address, ethDecimalOffset: uint256): modifying

NewPlasmaChain: event({PlasmaChainAddress: indexed(address), OperatorAddress: indexed(address), ChainMetadata: indexed(bytes32)})

plasmaChainTemplate: public(address)

@public
def initializeRegistry(template: address):
    assert self.plasmaChainTemplate == ZERO_ADDRESS
    assert template != ZERO_ADDRESS
    self.plasmaChainTemplate = template

@public
def createPlasmaChain(operator: address, ChainMetadata: bytes32) -> address:
    assert self.plasmaChainTemplate != ZERO_ADDRESS
    plasmaChain: address = create_with_code_of(self.plasmaChainTemplate)
    PlasmaChain(plasmaChain).setup(operator, 0)
    log.NewPlasmaChain(plasmaChain, operator, ChainMetadata)
    return plasmaChain
