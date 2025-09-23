package gnosis

var safeABIString = `[
    	{
        	"inputs": [],
        	"name": "getOwners",
        	"outputs": [{"internalType": "address[]", "name": "", "type": "address[]"}],
        	"stateMutability": "view",
        	"type": "function"
    	},
    	{
        	"inputs": [{"type": "address", "name": "owner"}],
        	"name": "isOwner",
        	"outputs": [{"type": "bool", "name": ""}],
        	"stateMutability": "view",
        	"type": "function"
    	},
        {
            "inputs": [
                {"type": "address", "name": "to"},
                {"type": "uint256", "name": "value"},
                {"type": "bytes", "name": "data"},
                {"type": "uint8", "name": "operation"},
                {"type": "uint256", "name": "safeTxGas"},
                {"type": "uint256", "name": "baseGas"},
                {"type": "uint256", "name": "gasPrice"},
                {"type": "address", "name": "gasToken"},
                {"type": "address", "name": "refundReceiver"},
                {"type": "bytes", "name": "signatures"}
            ],
            "name": "execTransaction",
            "outputs": [{"type": "bool", "name": "success"}],
            "type": "function"
        },
        {
            "inputs": [],
            "name": "nonce",
            "outputs": [{"type": "uint256", "name": ""}],
            "type": "function"
        },
        {
            "inputs": [],
            "name": "getThreshold",
            "outputs": [{"type": "uint256", "name": ""}],
            "type": "function"
        }
    ]`
