// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/**
 * @title Lib_PredeployAddresses
 */
library Lib_PredeployAddresses {
    address public constant ovmL2ToL1MessagePasser = 0x4200000000000000000000000000000000000000;
    address public constant ovmL1MessageSender = 0x4200000000000000000000000000000000000001;
    address public constant ovmDeployerWhitelist = 0x4200000000000000000000000000000000000002;
    address public constant ovmECDSAContractAccount = 0x4200000000000000000000000000000000000003;
    address public constant ovmSequencerEntrypoint = 0x4200000000000000000000000000000000000005;
    address public constant ovmETH = 0x4200000000000000000000000000000000000006;
    address public constant ovmL2CrossDomainMessenger = 0x4200000000000000000000000000000000000007;
    address public constant libAddressManager = 0x4200000000000000000000000000000000000008;
    address public constant ovmProxyEOA = 0x4200000000000000000000000000000000000009;
    address public constant ERC1820Registry = 0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24;
}
