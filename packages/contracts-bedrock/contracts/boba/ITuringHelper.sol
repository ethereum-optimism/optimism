//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

interface ITuringHelper {

  /* Called from the external contract. It takes an api endponit URL
     and an abi-encoded request payload. The URL and the list of allowed
     methods are supplied when the contract is created. In the future
     some of this registration might be moved into l2geth, allowing for
     security measures such as TLS client certificates. A configurable timeout
     could also be added.

     Logs the offchain response so that a future verifier or fraud prover
     can replay the transaction and ensure that it results in the same state
     root as during the initial execution. Note - a future version might
     need to include a timestamp and/or more details about the
     offchain interaction.
  */
  function TuringTx(string memory _url, bytes memory _payload) external returns (bytes memory);
}
