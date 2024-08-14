// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

/// @notice Deploys the Superchain contracts that can be shared by many chains.
/// We intentionally use the terms "Input" and "Output" to clearly distinguish this script from the
/// existing ones that use terms of "Config" and "Artifacts".
contract DeploySuperchain is Script {
    struct Roles {
        address proxyAdminOwner;
        address protocolVersionsOwner;
        address guardian;
    }

    struct Input {
        Roles roles;
        bool paused;
        ProtocolVersion requiredProtocolVersion;
        ProtocolVersion recommendedProtocolVersion;
    }

    struct Output {
        ProxyAdmin superchainProxyAdmin;
        SuperchainConfig superchainConfigImpl;
        SuperchainConfig superchainConfigProxy;
        ProtocolVersions protocolVersionsImpl;
        ProtocolVersions protocolVersionsProxy;
    }

    /// @notice This entrypoint is for end-users to deploy from an input file and write to an output file.
    function run(string memory _infile) public returns (Output memory output_, string memory outfile_) {
        Input memory _input = parseInputFile(_infile);
        output_ = runWithoutIO(_input);
        outfile_ = writeOutputFile(_infile, _input, output_);
        require(false, "DeploySuperchain: run is not implemented");
    }

    /// @notice This entrypoint is useful for e2e testing purposes, and doesn't use any file I/O.
    function runWithoutIO(Input memory _input) public returns (Output memory output_) {
        // Validate inputs.
        require(_input.roles.proxyAdminOwner != address(0), "zero address: proxyAdminOwner");
        require(_input.roles.protocolVersionsOwner != address(0), "zero address: protocolVersionsOwner");
        require(_input.roles.guardian != address(0), "zero address: guardian");

        // Deploy the proxy admin, with the owner set to the deployer.
        // We explicitly specify the deployer as `msg.sender` because for testing we deploy this script from a test
        // contract. If we provide no argument, the foundry default sender is be the broadcaster during test, but the
        // broadcaster needs to be the deployer since they are set to the initial proxy admin owner.
        vm.startBroadcast(msg.sender);

        output_.superchainProxyAdmin = new ProxyAdmin(msg.sender);
        vm.label(address(output_.superchainProxyAdmin), "SuperchainProxyAdmin");

        // Deploy implementation contracts.
        output_.superchainConfigImpl = new SuperchainConfig();
        output_.protocolVersionsImpl = new ProtocolVersions();

        // Deploy and initialize the proxies.
        output_.superchainConfigProxy = SuperchainConfig(address(new Proxy(address(output_.superchainProxyAdmin))));
        vm.label(address(output_.superchainConfigProxy), "SuperchainConfigProxy");
        output_.superchainProxyAdmin.upgradeAndCall(
            payable(address(output_.superchainConfigProxy)),
            address(output_.superchainConfigImpl),
            abi.encodeCall(SuperchainConfig.initialize, (_input.roles.guardian, _input.paused))
        );

        output_.protocolVersionsProxy = ProtocolVersions(address(new Proxy(address(output_.superchainProxyAdmin))));
        vm.label(address(output_.protocolVersionsProxy), "ProtocolVersionsProxy");
        output_.superchainProxyAdmin.upgradeAndCall(
            payable(address(output_.protocolVersionsProxy)),
            address(output_.protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (_input.roles.protocolVersionsOwner, _input.requiredProtocolVersion, _input.recommendedProtocolVersion)
            )
        );

        // Transfer ownership of the ProxyAdmin from the deployer to the specified owner.
        output_.superchainProxyAdmin.transferOwnership(_input.roles.proxyAdminOwner);

        vm.stopBroadcast();

        // Output assertions, to make sure outputs were assigned correctly.
        address[] memory addresses = new address[](5);
        addresses[0] = address(output_.superchainProxyAdmin);
        addresses[1] = address(output_.superchainConfigImpl);
        addresses[2] = address(output_.superchainConfigProxy);
        addresses[3] = address(output_.protocolVersionsImpl);
        addresses[4] = address(output_.protocolVersionsProxy);

        for (uint256 i = 0; i < addresses.length; i++) {
            require(addresses[i] != address(0), string.concat("zero address at index ", vm.toString(i)));
            require(addresses[i].code.length > 0, string.concat("no code at index ", vm.toString(i)));
        }

        // All addresses should be unique.
        for (uint256 i = 0; i < addresses.length; i++) {
            for (uint256 j = i + 1; j < addresses.length; j++) {
                string memory err = string.concat("duplicates at: ", vm.toString(i), ",", vm.toString(j));
                require(addresses[i] != addresses[j], err);
            }
        }
    }

    /// @notice This method is public for testing purposes, but there isn't a need for users to call this method
    /// directly.
    function parseInputFile(string memory _infile) public pure returns (Input memory input_) {
        _infile;
        input_;
        require(false, "DeploySuperchain: parseInputFile is not implemented");
    }

    /// @notice Writes an output file, where the filename is based on the input filename, e.g. an input file of
    /// `DeploySuperchain.in.toml` results in an output file of `DeploySuperchain.out.toml`.
    function writeOutputFile(
        string memory _infile,
        Input memory _input,
        Output memory _output
    )
        internal
        pure
        returns (string memory outfile_)
    {
        _infile;
        _input;
        _output;
        outfile_;
        require(false, "DeploySuperchain: writeOutputFile not implemented");
    }
}
