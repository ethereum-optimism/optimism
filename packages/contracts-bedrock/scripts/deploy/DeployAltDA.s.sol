// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IDataAvailabilityChallenge } from "interfaces/L1/IDataAvailabilityChallenge.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { Script } from "forge-std/Script.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

contract DeployAltDA is Script {
    struct Input {
        bytes32 salt;
        IProxyAdmin proxyAdmin;
        address challengeContractOwner;
        uint256 challengeWindow;
        uint256 resolveWindow;
        uint256 bondSize;
        uint256 resolverRefundPercentage;
    }

    struct Output {
        IDataAvailabilityChallenge dataAvailabilityChallengeProxy;
        IDataAvailabilityChallenge dataAvailabilityChallengeImpl;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployDataAvailabilityChallengeProxy(_input, output_);
        deployDataAvailabilityChallengeImpl(_input, output_);
        initializeDataAvailabilityChallengeProxy(_input, output_);

        assertValidOutput(_input, output_);
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

    function deployDataAvailabilityChallengeProxy(Input memory _input, Output memory _output) internal virtual {
        bytes32 salt = _input.salt;
        vm.broadcast(msg.sender);
        IDataAvailabilityChallenge proxy = IDataAvailabilityChallenge(
            DeployUtils.create2({
                _name: "Proxy",
                _salt: salt,
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (msg.sender)))
            })
        );
        vm.label(address(proxy), "DataAvailabilityChallengeProxy");
        _output.dataAvailabilityChallengeProxy = proxy;
    }

    function deployDataAvailabilityChallengeImpl(Input memory _input, Output memory _output) internal virtual {
        bytes32 salt = _input.salt;
        vm.broadcast(msg.sender);
        IDataAvailabilityChallenge impl = IDataAvailabilityChallenge(
            DeployUtils.create2({
                _name: "DataAvailabilityChallenge",
                _salt: salt,
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IDataAvailabilityChallenge.__constructor__, ()))
            })
        );
        vm.label(address(impl), "DataAvailabilityChallengeImpl");
        _output.dataAvailabilityChallengeImpl = impl;
    }

    function initializeDataAvailabilityChallengeProxy(Input memory _input, Output memory _output) internal virtual {
        IProxy proxy = IProxy(payable(address(_output.dataAvailabilityChallengeProxy)));
        IDataAvailabilityChallenge impl = _output.dataAvailabilityChallengeImpl;
        IProxyAdmin proxyAdmin = IProxyAdmin(payable(address(_input.proxyAdmin)));

        address contractOwner = _input.challengeContractOwner;
        uint256 challengeWindow = _input.challengeWindow;
        uint256 resolveWindow = _input.resolveWindow;
        uint256 bondSize = _input.bondSize;
        uint256 resolverRefundPercentage = _input.resolverRefundPercentage;

        vm.startBroadcast(msg.sender);
        proxy.upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                IDataAvailabilityChallenge.initialize,
                (contractOwner, challengeWindow, resolveWindow, bondSize, resolverRefundPercentage)
            )
        );
        proxy.changeAdmin(address(proxyAdmin));
        vm.stopBroadcast();
    }

    function assertValidInput(Input memory _input) internal virtual {
        require(_input.salt != bytes32(0), "DeployAltDA: salt not set");
        require(address(_input.proxyAdmin) != address(0), "DeployAltDA: proxyAdmin not set");
        require(_input.challengeContractOwner != address(0), "DeployAltDA: challengeContractOwner not set");
        require(_input.challengeWindow != 0, "DeployAltDA: challengeWindow not set");
        require(_input.resolveWindow != 0, "DeployAltDA: resolveWindow not set");
        require(_input.bondSize != 0, "DeployAltDA: bondSize not set");
        require(_input.resolverRefundPercentage <= 100, "DeployAltDA: resolverRefundPercentage too large");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal virtual {
        address[] memory addresses = Solarray.addresses(
            address(_output.dataAvailabilityChallengeProxy), address(_output.dataAvailabilityChallengeImpl)
        );
        DeployUtils.assertValidContractAddresses(addresses);

        assertValidDataAvailabilityChallengeProxy(_input, _output);
        assertValidDataAvailabilityChallengeImpl(_output);
    }

    function assertValidDataAvailabilityChallengeProxy(Input memory _input, Output memory _output) internal virtual {
        DeployUtils.assertERC1967ImplementationSet(address(_output.dataAvailabilityChallengeProxy));

        IProxy proxy = IProxy(payable(address(_output.dataAvailabilityChallengeProxy)));
        vm.prank(address(0));
        address admin = proxy.admin();
        require(admin == address(_input.proxyAdmin), "DACP-10");

        DeployUtils.assertInitialized({ _contractAddress: address(proxy), _isProxy: true, _slot: 0, _offset: 0 });

        vm.prank(address(0));
        address impl = proxy.implementation();
        require(impl == address(_output.dataAvailabilityChallengeImpl), "DACP-20");

        IDataAvailabilityChallenge dac = _output.dataAvailabilityChallengeProxy;
        require(dac.owner() == _input.challengeContractOwner, "DACP-30");
        require(dac.challengeWindow() == _input.challengeWindow, "DACP-40");
        require(dac.resolveWindow() == _input.resolveWindow, "DACP-50");
        require(dac.bondSize() == _input.bondSize, "DACP-60");
        require(dac.resolverRefundPercentage() == _input.resolverRefundPercentage, "DACP-70");
    }

    function assertValidDataAvailabilityChallengeImpl(Output memory _output) internal view virtual {
        IDataAvailabilityChallenge dac = _output.dataAvailabilityChallengeImpl;
        DeployUtils.assertInitialized({ _contractAddress: address(dac), _isProxy: false, _slot: 0, _offset: 0 });
    }
}
