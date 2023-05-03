//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { AdminFAM, Faucet } from "../universal/faucet/Faucet.sol";
import { FaucetHelper } from "../testing/helpers/FaucetHelper.sol";

contract Faucet_Initializer is Test {
    address internal faucetContractAdmin;
    address internal faucetAuthAdmin;
    address internal nonAdmin;
    address internal fundsReceiver;
    uint256 internal faucetAuthAdminKey;
    uint256 internal nonAdminKey;

    Faucet faucet;
    AdminFAM adminFam;

    FaucetHelper faucetHelper;

    function setUp() public {
        faucetContractAdmin = makeAddr("faucetContractAdmin");
        fundsReceiver = makeAddr("fundsReceiver");

        faucetAuthAdminKey = 0xB0B0B0B0;
        faucetAuthAdmin = vm.addr(faucetAuthAdminKey);

        nonAdminKey = 0xC0C0C0C0;
        nonAdmin = vm.addr(nonAdminKey);

        _initializeContracts();
    }

   /**
     * @notice Instantiates a Faucet.
     */
    function _initializeContracts() internal {
        faucet = new Faucet(faucetContractAdmin);

        // Fill faucet with ether.
        vm.deal(address(faucet), 10 ether);

        adminFam = new AdminFAM(faucetAuthAdmin);
        adminFam.initialize("AdminFAM");

        faucetHelper =  new FaucetHelper();
    }


    function _enableFaucetAuthModule() internal {
        vm.prank(faucetContractAdmin);
        faucet.configure(adminFam, Faucet.ModuleConfig(true, 1 days, 1 ether));
    }

   /**
     * @notice Get signature as a bytes blob.
     *
     */
    function _getSignature(uint256 _signingPrivateKey, bytes32 _digest)
        internal
        pure
        returns (bytes memory)
    {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(_signingPrivateKey, _digest);

        bytes memory signature = abi.encodePacked(r, s, v);
        return signature;
    }

    /**
     * @notice Signs a proof with the given private key and returns the signature using
     *         the given EIP712 domain separator. This assumes that the issuer's address is the
     *         corresponding public key to _issuerPrivateKey.
     */
    function issueProofWithEIP712Domain(
        uint256 _issuerPrivateKey,
        bytes memory _eip712Name,
        bytes memory _contractVersion,
        uint256 _eip712Chainid,
        address _eip712VerifyingContract,
        address recipient,
        bytes memory id,
        uint256 nonce
    ) internal view returns (bytes memory) {
        AdminFAM.Proof memory proof = AdminFAM.Proof(recipient, bytes32(keccak256(abi.encode(nonce))), id);
        return
            _getSignature(
                _issuerPrivateKey,
                faucetHelper.getDigestWithEIP712Domain(
                    proof,
                    _eip712Name,
                    _contractVersion,
                    _eip712Chainid,
                    _eip712VerifyingContract
                )
            );
    }
}

contract FaucetTest is Faucet_Initializer {
    function test_initialize() external {
        assertEq(faucet.ADMIN(), faucetContractAdmin);
    }

    function test_AuthAdmin_drip_succeeds() external {
        _enableFaucetAuthModule();
        bytes memory signature
            = issueProofWithEIP712Domain(
                faucetAuthAdminKey,
                bytes("AdminFAM"),
                bytes(adminFam.version()),
                block.chainid,
                address(adminFam),
                fundsReceiver,
                abi.encodePacked(fundsReceiver),
                faucetHelper.currentNonce()
            );

        vm.prank(nonAdmin);
        faucet.drip(
            Faucet.DripParameters(payable(fundsReceiver), faucetHelper.consumeNonce()),
            Faucet.AuthParameters(adminFam, abi.encodePacked(fundsReceiver), signature));
    }

    function test_nonAdmin_drip_fails() external {
        _enableFaucetAuthModule();
        bytes memory signature
            = issueProofWithEIP712Domain(
                nonAdminKey,
                bytes("AdminFAM"),
                bytes(adminFam.version()),
                block.chainid,
                address(adminFam),
                fundsReceiver,
                abi.encodePacked(fundsReceiver),
                faucetHelper.currentNonce()
            );

        vm.prank(nonAdmin);
        vm.expectRevert("Faucet: drip parameters could not be verified by security module");
        faucet.drip(
            Faucet.DripParameters(payable(fundsReceiver), faucetHelper.consumeNonce()),
            Faucet.AuthParameters(adminFam, abi.encodePacked(fundsReceiver), signature));
    }
}
