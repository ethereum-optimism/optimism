/**
 * Available Accounts
==================
(0) 0xde034D4336E819ffE0A6F1A041125E6bb6F51091 (100000 ETH)
(1) 0x8EF4BBf1670f256F7d72dd19c6359559792d8609 (100000 ETH)
(2) 0x3F8B49756D53c442870c3001F5608EE688F3f888 (100000 ETH)
(3) 0x8c7ef759BAf8335Bb46b4ee72927f50eA22B237d (100000 ETH)
(4) 0x198e5b8E94002A79cA3056fc97ECC437C42e68fc (100000 ETH)
(5) 0x83A55fFFbf9fE6B023C906Ee52B47c225ba323e9 (100000 ETH)
(6) 0x759a7e290dB389FdB58a417D853B616D54028f73 (100000 ETH)
(7) 0xB2cea1272c8772A537e55152bFB55A3197633145 (100000 ETH)
(8) 0x9bcBee4DD7C3bbeB1924ccF8fa7bd72b56a18d6c (100000 ETH)
(9) 0xFA299Db1777f4D1CA7Ee2fa42744a54f9331F117 (100000 ETH)

Private Keys
==================
(0) 0x521522d98061a4b90e903fd0ca01544190007ea92bbf6af0eb4c7ad3c9d25296
(1) 0xf905742ed1316f79b44cccd58f9482a303e7d51922ffaa3b047bea2257618352
(2) 0xa73fa64ced172642c614b52cb8b1b3d79d07fc8ba7b782d4306906b92b247b28
(3) 0x8b9a9a41c20da5e591e38ba78ca86f7f95d74f14ef36d69995fa4b8b63646938
(4) 0x7c1de38e6bd9afad7f4f7550f74d111887b13c1f297f5cb203b5d8ae158b0f62
(5) 0xa13215864c1067e5367988af437597f7149b99a3fc624668b1bab67aeb6136ba
(6) 0x2331fd2ee1fd1235049e7160a24bcdb41577e62f70fb3a8cfaabba507492c228
(7) 0xc178268dd4e7757f77bc0e96529ac14e2e7c63f04930dce612c7f67d7e76d95d
(8) 0xcc690c601e5530f61fce9a65843ab80dee282ee0fabb80dccaf15dcfcb16a334
(9) 0xdf9b8cb52897c9691def01826e6d6e3ec60c34efa49e4f9a16d16c610ca9957f
 */
let user0Address = '0xde034D4336E819ffE0A6F1A041125E6bb6F51091'


// var EIP20NonStandardInterface = artifacts.require("EIP20NonStandardInterface");
// var ComptrollerInterface = artifacts.require("ComptrollerInterface");

var ComptrollerG1 = artifacts.require("ComptrollerG1");
var ComptrollerG3 = artifacts.require("ComptrollerG3");
var Comptroller = artifacts.require("Comptroller");
var CErc20 = artifacts.require("CErc20");
var ComptrollerG2 = artifacts.require("ComptrollerG2");
var ComptrollerG6 = artifacts.require("ComptrollerG6");
var Exponential = artifacts.require("Exponential");

// var PriceOracle = artifacts.require("PriceOracle");
// var LegacyInterestRateModel = artifacts.require("LegacyInterestRateModel");
// var CToken = artifacts.require("CToken");

var ComptrollerG5 = artifacts.require("ComptrollerG5");
var LegacyJumpRateModelV2 = artifacts.require("LegacyJumpRateModelV2");
var ComptrollerG4 = artifacts.require("ComptrollerG4");
var CErc20Delegator = artifacts.require("CErc20Delegator");
var CErc20Delegate = artifacts.require("CErc20Delegate");
var CDaiDelegate = artifacts.require("CDaiDelegate");
var ExponentialNoError = artifacts.require("ExponentialNoError");
var SafeMath = artifacts.require("SafeMath");

/**
 * Error: Could not find artifacts for ErrorReporter from any sources
 */
// var ErrorReporter = artifacts.require("ErrorReporter");



var Unitroller = artifacts.require("Unitroller");
var Migrations = artifacts.require("Migrations");
var CarefulMath = artifacts.require("CarefulMath");
var JumpRateModel = artifacts.require("JumpRateModel");
var Timelock = artifacts.require("Timelock");
var EIP20Interface = artifacts.require("EIP20Interface");
var InterestRateModel = artifacts.require("InterestRateModel");
var JumpRateModelV2 = artifacts.require("JumpRateModelV2");

/**
 * Error: Could not find artifacts for ComptrollerStorage from any sources
 */
// var ComptrollerStorage = artifacts.require("ComptrollerStorage");

var WhitePaperInterestRateModel = artifacts.require("WhitePaperInterestRateModel");
var Reservoir = artifacts.require("Reservoir");
var CCompLikeDelegate = artifacts.require("CCompLikeDelegate");
var CErc20Immutable = artifacts.require("CErc20Immutable");
var CEther = artifacts.require("CEther");
var DAIInterestRateModelV3 = artifacts.require("DAIInterestRateModelV3");
var Maximillion = artifacts.require("Maximillion");
var SimplePriceOracle = artifacts.require("SimplePriceOracle");
var BaseJumpRateModelV2 = artifacts.require("BaseJumpRateModelV2");
var GovernorBravoDelegate = artifacts.require("GovernorBravoDelegate");
var GovernorBravoDelegator = artifacts.require("GovernorBravoDelegator");
var Comp = artifacts.require("Comp");


let adminAddress = '0x3a6C1f6C2de6c47e45d1Fd2d04C0F2601CF5979C';

/**
 * Error: Could not find artifacts for CTokenInterfaces from any sources
 */
// var CTokenInterfaces = artifacts.require("CTokenInterfaces");

module.exports = function(deployer, network, accounts) {
  deployer.then(async () => {
  await deployer.deploy(GovernorBravoDelegate);
  await deployer.deploy(SafeMath);
  await deployer.deploy(Comp, adminAddress);
  await deployer.deploy(Timelock, adminAddress, 172800);
  await deployer.deploy(GovernorBravoDelegator, Timelock.address, Comp.address, Timelock.address, GovernorBravoDelegate.address, 17280, 1, "100000000000000000000000"); 
  });
};



