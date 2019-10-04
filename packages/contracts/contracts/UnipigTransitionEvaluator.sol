pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {TransitionEvaluator} from "./TransitionEvaluator.sol";

contract UnipigTransitionEvaluator is TransitionEvaluator {
    bytes32 ZERO_BYTES32 = 0x0000000000000000000000000000000000000000000000000000000000000000;
    address UNISWAP_ADDRESS = 0x0000000000000000000000000000000000000000;
    uint UNISWAP_SLOT_INDEX = 0;
    uint UNISWAP_FEE_IN_BIPS = 30;
    // Transition Types
    uint CREATE_AND_TRANSFER_TYPE = 0;
    uint TRANSFER_TYPE = 1;
    uint SWAP_TYPE = 2;

    function evaluateTransition(
        bytes calldata _transition,
        dt.StorageSlot[2] calldata _storageSlots
    ) external view returns(bytes32[2] memory) {
        // Convert our inputs to memory
        bytes memory transition = _transition;
        dt.StorageSlot[2] memory storageSlots = _storageSlots;
        // Determine the transition type
        uint transitionType = inferTransitionType(transition);
        // And initalize updatedStorage which will contain the new storage values
        dt.Storage[2] memory updatedStorage;
        // Apply the transition and record the resulting storage slots
        if (transitionType == CREATE_AND_TRANSFER_TYPE) {
            dt.CreateAndTransferTransition memory createAndTransfer = decodeCreateAndTransferTransition(transition);
            updatedStorage = applyCreateAndTransferTransition(createAndTransfer, storageSlots);
        } else if (transitionType == TRANSFER_TYPE) {
            dt.TransferTransition memory transfer = decodeTransferTransition(transition);
            updatedStorage = applyTransferTransition(transfer, storageSlots);
        } else if (transitionType == SWAP_TYPE) {
            dt.SwapTransition memory swap = decodeSwapTransition(transition);
            updatedStorage = applySwapTransition(swap, storageSlots);
        } else {
            revert("Transition type not recognized!");
        }
        // Return the hash of both storage (leaf nodes to insert into the tree)
        bytes32[2] memory outputs;
        outputs[0] = getStorageHash(updatedStorage[0]);
        outputs[1] = getStorageHash(updatedStorage[1]);
        return outputs;
    }

    function verifyEcdsaSignatureOnHash(bytes memory _signature, bytes32 _hash, address _pubkey) private pure returns(bool) {
        bytes memory prefixedMessage = abi.encodePacked("\x19Ethereum Signed Message:\n32", _hash);
        bytes32 digest = keccak256(prefixedMessage);
        (uint8 v, bytes32 r, bytes32 s) = splitSignature(_signature);
        return ecrecover(digest, v, r, s) == _pubkey;
    }

    /**
     * Return the tx type inferred by the length of bytes
     */
    function inferTransitionType(
        bytes memory _transition
    ) public view returns(uint) {
        if (_transition.length == 352) {
            // Create account and Transfer
            return CREATE_AND_TRANSFER_TYPE;
        }
        if (_transition.length == 320) {
            // Transfer
            return TRANSFER_TYPE;
        }
        if (_transition.length == 384) {
            // Swap
            return SWAP_TYPE;
        }
        revert("Transition type not recognized!");
    }


    /**
     * Return the access list for this transition.
     * In unipig's case this is a uint32[2] for the two storage slots touched.
     */
    function getTransitionStateRootAndAccessList(
        bytes calldata _rawTransition
    ) external view returns(bytes32, uint32[2] memory) {
        // Initialize memory rawTransition
        bytes memory rawTransition = _rawTransition;
        // Initialize stateRoot and storageSlots
        bytes32 stateRoot;
        uint32[2] memory storageSlots;
        uint transitionType = inferTransitionType(rawTransition);
        if (transitionType == CREATE_AND_TRANSFER_TYPE) {
            dt.CreateAndTransferTransition memory transition = decodeCreateAndTransferTransition(rawTransition);
            stateRoot = transition.stateRoot;
            storageSlots[0] = transition.senderSlotIndex;
            storageSlots[1] = transition.recipientSlotIndex;
        }
        if (transitionType == TRANSFER_TYPE) {
            dt.TransferTransition memory transition = decodeTransferTransition(rawTransition);
            stateRoot = transition.stateRoot;
            storageSlots[0] = transition.senderSlotIndex;
            storageSlots[1] = transition.recipientSlotIndex;
        }
        if (transitionType == SWAP_TYPE) {
            dt.SwapTransition memory transition = decodeSwapTransition(rawTransition);
            stateRoot = transition.stateRoot;
            storageSlots[0] = transition.senderSlotIndex;
            storageSlots[1] = transition.uniswapSlotIndex;
        }
        return (stateRoot, storageSlots);
    }

    function getTransferTxHash(dt.TransferTx memory _transferTx) internal pure returns(bytes32) {
        return keccak256(abi.encode(_transferTx.sender, _transferTx.recipient, _transferTx.tokenType, _transferTx.amount));
    }

    function verifyEmptyStorage(dt.Storage memory _storage) internal pure {
        require(_storage.pubkey == 0x0000000000000000000000000000000000000000, "Pubkey of storage slot must be zero");
        require(_storage.balances[0] == 0, "Uni balance must be zero");
        require(_storage.balances[1] == 0, "Pigi balance must be zero");
    }

    function getSwapTxHash(dt.SwapTx memory _swapTx) internal pure returns(bytes32) {
        return keccak256(abi.encode(_swapTx.sender, _swapTx.tokenType, _swapTx.inputAmount, _swapTx.minOutputAmount, _swapTx.timeout));
    }

    /**
     * Apply a create storage slot and transfer account transition
     */
    function applyCreateAndTransferTransition(
        dt.CreateAndTransferTransition memory _transition,
        dt.StorageSlot[2] memory _storageSlots
    ) public view returns(dt.Storage[2] memory) {
        // Verify that the recipient storage is empty
        verifyEmptyStorage(_storageSlots[1].value);
        // Now set storage slot to have the pubkey of the recipient
        _storageSlots[1].value.pubkey = _transition.createdAccountPubkey;
        // Next create a transferTransition based on this createAndTransferTransition
        dt.TransferTransition memory transferTransition = dt.TransferTransition(
            _transition.stateRoot,
            _transition.senderSlotIndex,
            _transition.recipientSlotIndex,
            _transition.tokenType,
            _transition.amount,
            _transition.signature
        );
        // Now simply apply the transfer transition as usual
        return applyTransferTransition(transferTransition, _storageSlots);
    }

    /**
     * Apply a transfer stored account transition
     */
    function applyTransferTransition(
        dt.TransferTransition memory _transition,
        dt.StorageSlot[2] memory _storageSlots
    ) public view returns(dt.Storage[2] memory) {
        // First construct the transaction from the storage slots
        address sender = _storageSlots[0].value.pubkey;
        address recipient = _storageSlots[1].value.pubkey;
        dt.TransferTx memory transferTx = dt.TransferTx(sender, recipient, _transition.tokenType, _transition.amount);

        // Next check to see if the signature is valid
        require(verifyEcdsaSignatureOnHash(_transition.signature, getTransferTxHash(transferTx), sender), "Transfer signature is invalid!");
        // Also make sure we're not sending to Unipig
        require(_storageSlots[1].slotIndex != UNISWAP_SLOT_INDEX, "Transfer cannot be made to Unipig!");

        // Create an array to store our output storage slots
        dt.Storage[2] memory outputStorage;
        // Now we know the pubkeys are correct, let's compute the output of the transaction
        uint senderBalance = _storageSlots[0].value.balances[transferTx.tokenType];

        // First let's make sure the sender has enough money
        require(senderBalance > transferTx.amount, "Sender does not have enough money!");

        // Update the storage slots with the new balances
        _storageSlots[0].value.balances[transferTx.tokenType] -= transferTx.amount;
        _storageSlots[1].value.balances[transferTx.tokenType] += transferTx.amount;
        // Set the outputs
        outputStorage[0] = _storageSlots[0].value;
        outputStorage[1] = _storageSlots[1].value;
        // Return the outputs!
        return outputStorage;
    }

    /**
     * Apply a swap transition
     */
    function applySwapTransition(
        dt.SwapTransition memory _transition,
        dt.StorageSlot[2] memory _storageSlots
    ) public view returns(dt.Storage[2] memory) {
        address sender = _storageSlots[0].value.pubkey;
        address recipient = _storageSlots[1].value.pubkey;
        // Create our swapTx
        dt.SwapTx memory swapTx = dt.SwapTx(
            sender,
            _transition.tokenType,
            _transition.inputAmount,
            _transition.minOutputAmount,
            _transition.timeout
        );

        // Make sure that the provided storage slots are corrent
        require(verifyEcdsaSignatureOnHash(_transition.signature, getSwapTxHash(swapTx), sender), "Swap signature is invalid!");
        require(_storageSlots[1].slotIndex == UNISWAP_SLOT_INDEX && recipient == UNISWAP_ADDRESS, "Swap tx must be swapping with Unipig!");

        // Create an array to store our output storage slots
        dt.Storage[2] memory outputStorage;

        // Now we know the storage slots are correct, let's first make sure the sender has enough money to initiate the swap
        uint senderBalance = _storageSlots[0].value.balances[swapTx.tokenType];
        // Make sure the sender has enough money
        require(senderBalance > swapTx.inputAmount, "Sender of the swap tx does not have enough money!");

        // Store variables used for calculating the SWAP
        uint inputTokenType = swapTx.tokenType;
        uint outputTokenType = 1 - swapTx.tokenType;
        dt.Storage memory senderStorage = _storageSlots[0].value;
        dt.Storage memory uniswapStorage = _storageSlots[1].value;

        // Compute the SWAP
        uint invariant = uniswapStorage.balances[0] * uniswapStorage.balances[1];
        uint inputWithFee = swapTx.inputAmount * (10000 - UNISWAP_FEE_IN_BIPS) / 10000;
        uint totalInput = inputWithFee + uniswapStorage.balances[inputTokenType];
        uint newOutputBalance = invariant / totalInput;
        uint32 outputAmount = uniswapStorage.balances[outputTokenType] - uint32(newOutputBalance);
        // Make sure the output amount is above or equal to the minimum
        require(outputAmount >= swapTx.minOutputAmount, "Swap output amount is not above or equal to min output!");

        // Update the sender storage slots with the new balances
        senderStorage.balances[inputTokenType] -= swapTx.inputAmount;
        senderStorage.balances[outputTokenType] += outputAmount;
        // Update uniswap storage slots with the new balances
        uniswapStorage.balances[inputTokenType] += swapTx.inputAmount;
        uniswapStorage.balances[outputTokenType] -= outputAmount;

        // Set our output storage
        outputStorage[0] = senderStorage;
        outputStorage[1] = uniswapStorage;
        // Return the outputs!
        return outputStorage;
    }

    /**
     * Get the hash of the storage value.
     */
    function getStorageHash(dt.Storage memory _storage) public pure returns(bytes32) {
        // Here we don't use `abi.encode([struct])` because it's not clear
        // how to generate that encoding client-side.
        return keccak256(abi.encode(_storage.pubkey, _storage.balances[0], _storage.balances[1]));
    }

    /************
     * Decoding *
     ***********/

    /**
     * Decode a createAndTransferTransition
     * TODO: Decode directly into a struct.
     */
     function decodeCreateAndTransferTransition(bytes memory _rawBytes) internal pure returns(dt.CreateAndTransferTransition memory) {
         (
             bytes32 stateRoot,
             uint32 senderSlotIndex,
             uint32 recipientSlotIndex,
             address createdAccountPubkey,
             uint tokenType,
             uint32 amount,
             bytes memory signature
         ) = abi.decode((_rawBytes), (bytes32, uint32, uint32, address, uint, uint32, bytes));
         dt.CreateAndTransferTransition memory transition = dt.CreateAndTransferTransition(
             stateRoot,
             senderSlotIndex,
             recipientSlotIndex,
             createdAccountPubkey,
             tokenType,
             amount,
             signature
         );
         return transition;
     }

    /**
     * Decode a TransferTransition
     * TODO: Decode directly into a struct.
     */
     function decodeTransferTransition(bytes memory _rawBytes) internal pure returns(dt.TransferTransition memory) {
         (
             bytes32 stateRoot,
             uint32 senderSlotIndex,
             uint32 recipientSlotIndex,
             uint tokenType,
             uint32 amount,
             bytes memory signature
         ) = abi.decode((_rawBytes), (bytes32, uint32, uint32, uint, uint32, bytes));
         dt.TransferTransition memory transition = dt.TransferTransition(
             stateRoot,
             senderSlotIndex,
             recipientSlotIndex,
             tokenType,
             amount,
             signature
         );
         return transition;
     }

    /**
     * Decode a SwapTransition
     * TODO: Decode directly into a struct.
     */
     function decodeSwapTransition(bytes memory _rawBytes) internal pure returns(dt.SwapTransition memory) {
         (
             bytes32 stateRoot,
             uint32 senderSlotIndex,
             uint32 uniswapSlotIndex,
             uint tokenType,
             uint32 inputAmount,
             uint32 minOutputAmount,
             uint timeout,
             bytes memory signature
         ) = abi.decode((_rawBytes), (bytes32, uint32, uint32, uint, uint32, uint32, uint, bytes));
         dt.SwapTransition memory transition = dt.SwapTransition(
             stateRoot,
             senderSlotIndex,
             uniswapSlotIndex,
             tokenType,
             inputAmount,
             minOutputAmount,
             timeout,
             signature
         );
         return transition;
     }

    // splits a signature string into v, r, s
    function splitSignature(bytes memory sig) internal pure returns (uint8 v, bytes32 r, bytes32 s)
    {
        require(sig.length == 65, 'invalid signature length.');

        assembly {
            // first 32 bytes, after the length prefix.
            r := mload(add(sig, 32))
            // second 32 bytes.
            s := mload(add(sig, 64))
            // final byte (first byte of the next 32 bytes).
            v := byte(0, mload(add(sig, 96)))
        }

        return (v, r, s);
    }
    // recovers a signer for a message (technically, an ethereum-compliant signature on the HASH of the message, which our DefaultSignatureProvider performs.)
    function recoverSigner(bytes memory message, bytes memory sig) public pure returns (address)
    {
        bytes32 messageHash = keccak256(message);
        bytes memory prefixedMessage = abi.encodePacked("\x19Ethereum Signed Message:\n32", messageHash);
        bytes32 digest = keccak256(prefixedMessage);
        (uint8 v, bytes32 r, bytes32 s) = splitSignature(sig);

        return ecrecover(digest, v, r, s);
    }
}
