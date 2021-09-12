// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/**
 * @title Lib_DefaultValues
 */
library Lib_DefaultValues {

    address internal constant DEFAULT_ADDRESS = 0xdEfa017defA017DeFA017DEfa017DeFA017DeFa0;

    uint256 internal constant DEFAULT_UINT256 =
    0xdefa017defa017defa017defa017defa017defa017defa017defa017defa017d;

    // The default x-domain message sender being set to a non-zero value makes
    // deployment a bit more expensive, but in exchange the refund on every call to
    // `relayMessage` by the L1 and L2 messengers will be higher.
    address internal constant DEFAULT_XDOMAIN_SENDER = 0x000000000000000000000000000000000000dEaD;

}
