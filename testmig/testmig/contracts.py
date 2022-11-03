ADDRESSES = {
    'Lib_AddressManager': '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
    'Proxy__OVM_L1StandardBridge': '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
    'Proxy__OVM_L1CrossDomainMessenger': '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
}

DEPLOYER_ADDR = '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266'
DEPLOYER_ADDR_PADDED = '0x000000000000000000000000f39Fd6e51aad88F6F4ce6aB8827279cffFb92266'

SLOTS_TO_MODIFY = (
    # _owner slot
    (ADDRESSES['Lib_AddressManager'], '0x0', DEPLOYER_ADDR_PADDED),
    # proxy owner slot
    (ADDRESSES['Proxy__OVM_L1StandardBridge'], '0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103',
     DEPLOYER_ADDR_PADDED),
    # owner slot on the underlying implementation
    (ADDRESSES['Proxy__OVM_L1CrossDomainMessenger'], '0x33', DEPLOYER_ADDR_PADDED)
)
