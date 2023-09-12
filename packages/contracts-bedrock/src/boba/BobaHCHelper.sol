//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import { Burn } from "../libraries/Burn.sol";

import "./BobaTuringCredit.sol";

contract BobaHCHelper /*is Ownable*/ {
  //using SafeMath for uint256;
  using SafeERC20 for IERC20;

  address immutable HelperAddr     = 0x42000000000000000000000000000000000000Fd;   // Address of this predeploy
  address immutable OffchainCaller = 0xdEAddEadDeaDDEaDDeadDeAddeadDEaddeaD9901;   // special "From" address
  address immutable hcToken = 0x4200000000000000000000000000000000000023;	   // Boba L2 token
  address immutable legacyCreditAddr = 0x4200000000000000000000000000000000000020; // BobaTuringCredit address

  // Cost (in hcToken) for various operations
  uint256 constant RegCost = 100;  // Register an endpoint
  uint256 constant CallCost = 50;  // Call off-chain
  uint256 constant RNGCost = 30;   // Use a random-number function

  // Collected balance
  uint256 public ownerRevenue;
  address owner;

  mapping(address => uint256) public prepaidCredit;
  mapping(address => uint256) public pendingCharge; // Placeholder
  mapping(address => bool) public legacyWhitelist;

  struct EndpointEntry {
    address Owner;
    mapping (address => bool) PermittedCallers;
  }
  mapping (bytes32 => EndpointEntry) public Endpoints;

  mapping(bytes32 => bytes) OffchainResponses;

  // events
  event EndpointRegistered(string URL, address owner);
  event EndpointUnregistered(string URL, address owner);

  constructor () {
    owner = msg.sender;
  }

  // -------------------------------------------------------------------------
  // User interface (current API)

  function CallOffchain(string calldata _url, bytes calldata _payload) public returns (bool, bytes memory) {
   uint32 method = 0xc1fd7e46; // "CallOffchain(string,bytes)" signature

    bytes32 EK = keccak256(abi.encodePacked(_url));

    if (msg.sender != address(this)) {
      require (Endpoints[EK].Owner != address(0), "Endpoint not registered");
      require (Endpoints[EK].PermittedCallers[msg.sender], "Caller is not permitted");
    }

    bytes32 key = keccak256(abi.encodePacked(method, msg.sender, abi.encodePacked(_url), _payload));
    return getCachedResponse(key, msg.sender, CallCost);
  }

  // For simple random number generation in a single Tx, use the legacy GetRandom()
  function SequentialRandom(bytes32 session, bytes32 nextHash, uint256 myNum)
    public returns (bool, bytes32, uint256) {
    uint32  method = 0x32be428f; // "SequentialRandom(bytes32,bytes32,uint256)" signature

    bool success;
    bytes memory responsedata;
    bytes32 serverHash;
    uint256 responseNum;

    bytes32 key = keccak256(abi.encodePacked(method, msg.sender, session, nextHash, myNum));
    (success, responsedata) = getCachedResponse(key, msg.sender, RNGCost);

    if (success) {
      (serverHash, responseNum) = abi.decode(responsedata,(bytes32,uint256));
    }
    require(success && (responsedata.length == 64), "Random number generation failed");

    return (success, serverHash, responseNum);
  }

  // -------------------------------------------------------------------------
  // Management functions

  function RegisterEndpoint(string calldata _url, bytes32 _auth)
    public returns (bool) {

    BobaHCHelper Self = BobaHCHelper(HelperAddr);
    bytes32 EK = keccak256(abi.encodePacked(_url));

    IERC20(hcToken).safeTransferFrom(msg.sender, address(this), RegCost);
    ownerRevenue += RegCost;

    bytes memory request = abi.encodeWithSignature("RegisterV1()");
    bytes memory response;
    bool success;

    (success, response) = Self.CallOffchain(_url, request);

    if (success && response.length == 32 && keccak256(response) == _auth) {
      Endpoints[EK].Owner = msg.sender;
      emit EndpointRegistered(_url, msg.sender);
    } else {
      success = false;
    }

    return success;
  }

  function UnregisterEndpoint(string calldata _url)
    public {

    bytes32 EK = keccak256(abi.encodePacked(_url));
    require(Endpoints[EK].Owner != address(0), "Endpoint is not registered");
    require(Endpoints[EK].Owner == msg.sender, "Not the Endpoint owner");

    delete(Endpoints[EK]);
    emit EndpointUnregistered(_url, msg.sender);
  }

  function AddPermittedCaller(string calldata _url, address _callerAddress) public {
    bytes32 EK = keccak256(abi.encodePacked(_url));
    require(Endpoints[EK].Owner != address(0), "Endpoint is not registered");
    require(Endpoints[EK].Owner == msg.sender, "Not the Endpoint owner");

    Endpoints[EK].PermittedCallers[_callerAddress] = true;
  }

  function RemovePermittedCaller(string calldata _url, address _callerAddress) public {
    bytes32 EK = keccak256(abi.encodePacked(_url));
    require(Endpoints[EK].Owner != address(0), "Endpoint is not registered");
    require(Endpoints[EK].Owner == msg.sender, "Not the Endpoint owner");

    Endpoints[EK].PermittedCallers[_callerAddress] = false;
  }

  function CheckPermittedCaller(string calldata _url, address _callerAddress) public view returns (bool) {
    bytes32 EK = keccak256(abi.encodePacked(_url));
    require(Endpoints[EK].Owner != address(0), "Endpoint is not registered");
    return(Endpoints[EK].PermittedCallers[_callerAddress]);
  }

  function AddCredit(address _caller, uint256 _amount) public {
    require(_amount > 0, "Invalid amount");

    IERC20(hcToken).safeTransferFrom(msg.sender, address(this), _amount);
    prepaidCredit[_caller] += _amount;
  }

    /**
     * @dev Owner withdraws revenue
     *
     * @param _withdrawAmount the revenue amount that the owner wants to withdraw
     */
  function withdrawRevenue(uint256 _withdrawAmount) public /* onlyOwner onlyInitialized */ {
        require(_withdrawAmount <= ownerRevenue, "Invalid Amount");
        ownerRevenue -= _withdrawAmount;
        // emit WithdrawRevenue(msg.sender, _withdrawAmount);
        IERC20(hcToken).safeTransfer(owner, _withdrawAmount);
  }

    function transferOwnership(address _newOwner) public /* onlyOwner */ {
        require(_newOwner != address(0));
        owner = _newOwner;
        //emit TransferOwnership(msg.sender, _newOwner);
    }

  // -------------------------------------------------------------------------
  // Offchain interface point. The Sequencer watches for this function to revert
  // and then inserts a special transaction to call PutResponse before re-running
  // the user Tx which will now find a populated map entry.

  function getCachedResponse(bytes32 _cacheKey, address _caller, uint256 _cost)
    internal returns (bool, bytes memory) {
    if (_caller != address(this)) {
      uint256 creditCheck = _cost + pendingCharge[_caller];
      require (prepaidCredit[_caller] >= creditCheck, "Insufficient credit");

      prepaidCredit[_caller] -= _cost;
      ownerRevenue += _cost;
      // TODO - could extend this to also charge credits which would be paid to the endpoint owner.
    }

    bytes memory cachedResponse = OffchainResponses[_cacheKey];
    require(cachedResponse.length > 0, "HC: Missing cache entry"); // Trigger string; don't edit.

    delete(OffchainResponses[_cacheKey]);
    return abi.decode(cachedResponse,(bool, bytes));
  }

  function PutResponse(bytes32 _cacheKey, bool _success, bytes calldata _returndata) public {
    require(msg.sender == OffchainCaller, "Invalid PutResponse() caller");

    // Eventually this should just overwrite
    require(OffchainResponses[_cacheKey].length == 0, "DEBUG - Already exists");
    OffchainResponses[_cacheKey] = abi.encode(_success, _returndata);
  }

  // -------------------------------------------------------------------------
  // Legacy interface

  function GetLegacyResponse(uint32 rType, string memory _url, bytes memory _payload)
    public returns (bytes memory) {
    require (rType == 1 || rType == 0x02000001, "TURING: Geth intercept failure");
    uint32 method = 0xd40c48b0; // GetLegacyResponse(uint32,string,bytes)

    // TBD whether this is needed
    require(legacyWhitelist[msg.sender] || true, "Legacy caller is not in whitelist");

    bytes32 key = keccak256(abi.encodePacked(method, msg.sender, abi.encodePacked(_url), _payload));
    bool success;
    bytes memory responsedata;

    // Legacy calls are billed by the old TuringCredit contract. Unlike the simple
    // GetRandom() this does not fall through to the new billing mechanism.
    if (legacyCreditAddr != address(0)) {
      BobaTuringCredit legacyCredit = BobaTuringCredit(legacyCreditAddr);
      uint256 legacyPrice = legacyCredit.turingPrice();
      bool legacyPaid = legacyCredit.spendCredit(msg.sender, legacyPrice);
      require(legacyPaid, "Insufficient credit in legacy BobaTuringCredit contract");
    }

    (success,responsedata) = getCachedResponse(key, msg.sender, 0);
    require(success);

    return responsedata;
  }

  // This function was part of the legacy API but is also supported for new
  // applications for which this simple algorithm has acceptable security.
  function GetRandom(uint32 rType, uint256 _random)
    public returns (uint256) {
    require (rType == 1, "TURING: Geth intercept failure");

    uint32 method = 0x493d57d6; // GetRandom(uint32,uint256)
    bool success;
    bytes memory responsedata;
    bytes32 key = keccak256(abi.encodePacked(method, msg.sender));

    uint256 cost = RNGCost;

    if (legacyCreditAddr != address(0)) {
      BobaTuringCredit legacyCredit = BobaTuringCredit(legacyCreditAddr);
      uint256 legacyPrice = legacyCredit.turingPrice();
      bool legacyPaid = legacyCredit.spendCredit(msg.sender, legacyPrice);
      if (legacyPaid) {
        cost = 0;
      }
    }
    (success, responsedata) = getCachedResponse(key, msg.sender, cost);
    require(success && (responsedata.length == 32), "HC: GetRandom failure");

    return abi.decode(responsedata,(uint256));
  }
}
