// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

// import '@openzeppelin/contracts/math/SafeMath.sol';
// import '@openzeppelin/contracts/access/Ownable.sol';
// import './L2DepositedERC20.sol';

// contract TokenPool is Ownable {
//     using SafeMath for uint256;

//     mapping(address => uint256) lastRequest;
//     address tokenAddress;
    
//     event RequestToken (
//         address _requestAddress,
//         uint256 _timestamp,
//         uint256 _amount
//     );
    
//     function registerTokenAddress(
//         address _tokenAddress
//     )   
//         public
//         onlyOwner() 
//     {
//         tokenAddress = _tokenAddress;
//     }
    
//     function requestToken() 
//         public
//     {   
//         require(lastRequest[msg.sender].add(3600) <= block.timestamp, "Request limit");
//         L2DepositedERC20(tokenAddress).transfer(msg.sender, 10e18);
//         lastRequest[msg.sender] = block.timestamp;
        
//         emit RequestToken(msg.sender, block.timestamp, 10e18);
//     }
// }