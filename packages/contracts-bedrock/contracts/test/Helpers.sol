//SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";
import { ERC721 } from "@rari-capital/solmate/src/tokens/ERC721.sol";
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { OptimistInviter } from "../periphery/op-nft/OptimistInviter.sol";
import { IERC1271 } from "@openzeppelin/contracts/interfaces/IERC1271.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import {
    ECDSAUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/ECDSAUpgradeable.sol";
import {
    AdminFaucetAuthModule
} from "../periphery/faucet/authmodules/AdminFaucetAuthModule.sol";


contract TestERC20 is ERC20 {
    constructor() ERC20("TEST", "TST", 18) {}

    function mint(address to, uint256 value) public {
        _mint(to, value);
    }
}

contract TestERC721 is ERC721 {
    constructor() ERC721("TEST", "TST") {}

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    function tokenURI(uint256) public pure virtual override returns (string memory) {}
}

contract CallRecorder {
    struct CallInfo {
        address sender;
        bytes data;
        uint256 gas;
        uint256 value;
    }

    CallInfo public lastCall;

    function record() public payable {
        lastCall.sender = msg.sender;
        lastCall.data = msg.data;
        lastCall.gas = gasleft();
        lastCall.value = msg.value;
    }
}

contract Reverter {
    function doRevert() public pure {
        revert("Reverter reverted");
    }
}

contract SimpleStorage {
    mapping(bytes32 => bytes32) public db;

    function set(bytes32 _key, bytes32 _value) public payable {
        db[_key] = _value;
    }

    function get(bytes32 _key) public view returns (bytes32) {
        return db[_key];
    }
}

/**
 * Simple helper contract that helps with testing flow and signature for OptimistInviter contract.
 * Made this a separate contract instead of including in OptimistInviter.t.sol for reusability.
 */
contract OptimistInviterHelper {
    /**
     * @notice EIP712 typehash for the ClaimableInvite type.
     */
    bytes32 public constant CLAIMABLE_INVITE_TYPEHASH =
        keccak256("ClaimableInvite(address issuer,bytes32 nonce)");

    /**
     * @notice EIP712 typehash for the EIP712Domain type that is included as part of the signature.
     */
    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256(
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
        );

    /**
     * @notice Address of OptimistInviter contract we are testing.
     */
    OptimistInviter public optimistInviter;

    /**
     * @notice OptimistInviter contract name. Used to construct the EIP-712 domain.
     */
    string public name;

    /**
     * @notice Keeps track of current nonce to generate new nonces for each invite.
     */
    uint256 public currentNonce;

    constructor(OptimistInviter _optimistInviter, string memory _name) {
        optimistInviter = _optimistInviter;
        name = _name;
    }

    /**
     * @notice Returns the hash of the struct ClaimableInvite.
     *
     * @param _claimableInvite ClaimableInvite struct to hash.
     *
     * @return EIP-712 typed struct hash.
     */
    function getClaimableInviteStructHash(OptimistInviter.ClaimableInvite memory _claimableInvite)
        public
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encode(
                    CLAIMABLE_INVITE_TYPEHASH,
                    _claimableInvite.issuer,
                    _claimableInvite.nonce
                )
            );
    }

    /**
     * @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
     *         pseudorandom nonces.
     *
     * @return Nonce that should be used as part of ClaimableInvite.
     */
    function consumeNonce() public returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /**
     * @notice Returns a ClaimableInvite with the issuer and current nonce.
     *
     * @param _issuer Issuer to include in the ClaimableInvite.
     *
     * @return ClaimableInvite that can be hashed & signed.
     */
    function getClaimableInviteWithNewNonce(address _issuer)
        public
        returns (OptimistInviter.ClaimableInvite memory)
    {
        return OptimistInviter.ClaimableInvite(_issuer, consumeNonce());
    }

    /**
     * @notice Computes the EIP712 digest with default correct parameters.
     *
     * @param _claimableInvite ClaimableInvite struct to hash.
     *
     * @return EIP-712 compatible digest.
     */
    function getDigest(OptimistInviter.ClaimableInvite calldata _claimableInvite)
        public
        view
        returns (bytes32)
    {
        return
            getDigestWithEIP712Domain(
                _claimableInvite,
                bytes(name),
                bytes(optimistInviter.EIP712_VERSION()),
                block.chainid,
                address(optimistInviter)
            );
    }

    /**
     * @notice Computes the EIP712 digest with the given domain parameters.
     *         Used for testing that different domain parameters fail.
     *
     * @param _claimableInvite   ClaimableInvite struct to hash.
     * @param _name              Contract name to use in the EIP712 domain.
     * @param _version           Contract version to use in the EIP712 domain.
     * @param _chainid           Chain ID to use in the EIP712 domain.
     * @param _verifyingContract Address to use in the EIP712 domain.
     *
     * @return EIP-712 compatible digest.
     */
    function getDigestWithEIP712Domain(
        OptimistInviter.ClaimableInvite calldata _claimableInvite,
        bytes memory _name,
        bytes memory _version,
        uint256 _chainid,
        address _verifyingContract
    ) public pure returns (bytes32) {
        bytes32 domainSeparator = keccak256(
            abi.encode(
                EIP712_DOMAIN_TYPEHASH,
                keccak256(_name),
                keccak256(_version),
                _chainid,
                _verifyingContract
            )
        );
        return
            ECDSA.toTypedDataHash(domainSeparator, getClaimableInviteStructHash(_claimableInvite));
    }
}

// solhint-disable max-line-length
/**
 * Simple ERC1271 wallet that can be used to test the ERC1271 signature checker.
 * https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/mocks/ERC1271WalletMock.sol
 */
contract TestERC1271Wallet is Ownable, IERC1271 {
    constructor(address originalOwner) {
        transferOwnership(originalOwner);
    }

    function isValidSignature(bytes32 hash, bytes memory signature)
        public
        view
        override
        returns (bytes4 magicValue)
    {
        return
            ECDSA.recover(hash, signature) == owner() ? this.isValidSignature.selector : bytes4(0);
    }
}

/**
 * Simple helper contract that helps with testing the Faucet contract.
 */
contract FaucetHelper {
    /**
     * @notice EIP712 typehash for the Proof type.
     */
    bytes32 public constant PROOF_TYPEHASH =
        keccak256("Proof(address recipient,bytes32 nonce,bytes32 id)");

    /**
     * @notice EIP712 typehash for the EIP712Domain type that is included as part of the signature.
     */
    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256(
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
        );

    /**
     * @notice Keeps track of current nonce to generate new nonces for each drip.
     */
    uint256 public currentNonce;

    /**
     * @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
     *         pseudorandom nonces.
     *
     * @return Nonce that should be used as part of drip parameters.
     */
    function consumeNonce() public returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /**
     * @notice Returns the hash of the struct Proof.
     *
     * @param _proof Proof struct to hash.
     *
     * @return EIP-712 typed struct hash.
     */
    function getProofStructHash(AdminFaucetAuthModule.Proof memory _proof)
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(PROOF_TYPEHASH, _proof.recipient, _proof.nonce, _proof.id));
    }

    /**
     * @notice Computes the EIP712 digest with the given domain parameters.
     *         Used for testing that different domain parameters fail.
     *
     * @param _proof             Proof struct to hash.
     * @param _name              Contract name to use in the EIP712 domain.
     * @param _version           Contract version to use in the EIP712 domain.
     * @param _chainid           Chain ID to use in the EIP712 domain.
     * @param _verifyingContract Address to use in the EIP712 domain.
     * @param _verifyingContract Address to use in the EIP712 domain.
     * @param _verifyingContract Address to use in the EIP712 domain.
     *
     * @return EIP-712 compatible digest.
     */
    function getDigestWithEIP712Domain(
        AdminFaucetAuthModule.Proof memory _proof,
        bytes memory _name,
        bytes memory _version,
        uint256 _chainid,
        address _verifyingContract
    ) public pure returns (bytes32) {
        bytes32 domainSeparator = keccak256(
            abi.encode(
                EIP712_DOMAIN_TYPEHASH,
                keccak256(_name),
                keccak256(_version),
                _chainid,
                _verifyingContract
            )
        );
        return ECDSAUpgradeable.toTypedDataHash(domainSeparator, getProofStructHash(_proof));
    }
}
