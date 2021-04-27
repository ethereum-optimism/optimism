export const SELLER_API_URL = "https://s8ry0qo8ql.execute-api.us-west-1.amazonaws.com/prod/";
export const BUYER_API_URL = "https://i1hy90c1db.execute-api.us-west-1.amazonaws.com/prod/";
export const HASHCAST_API_URL = "https://hsj4abmlzi.execute-api.us-west-1.amazonaws.com/prod/";
export const WEBSOCKET_API_URL = "wss://hxpkicwmek.execute-api.us-west-1.amazonaws.com/prod";

export const SELLER_OPTIMISM_API_URL = "https://pm7f0dp9ud.execute-api.us-west-1.amazonaws.com/prod/";
export const BUYER_OPTIMISM_API_URL = "https://n245h0ka3i.execute-api.us-west-1.amazonaws.com/prod/";
export const SERVICE_OPTIMISM_API_URL = "https://zlba6djrv6.execute-api.us-west-1.amazonaws.com/prod/";

export const VERSION = "1.0.26";

// global network
export const SELECT_NETWORK = 'rinkeby'; // kovan || local || rinkeby

export const INFURA_ID = "460f40a260564ac4a4f4b3fffb032dad";

export const NETWORK = (chainId) => {
  for (let n in NETWORKS) {
    if (NETWORKS[n].chainId === chainId) {
      return NETWORKS[n];
    }
  }
};

export const NETWORKS = {
  kovan: {
    name: "Kovan Network",
    color: "#7003DD",
    chainId: 42,
    rpcUrl: `https://kovan.infura.io/v3/${INFURA_ID}`,
    blockExplorer: "https://kovan.etherscan.io/",
    faucet: "https://gitter.im/kovan-testnet/faucet", //https://faucet.kovan.network/
    l1ETHGatewayAddress: '0x6647D5BD9EB9425838Bb89f76a166228b95671a3',
    l1MessengerAddress: '0xb89065D5eB05Cac554FDB11fC764C679b4202322'
  },
  kovanL2: {
    name: "Kovan Optimism Network",
    color: "#e0bfc3",
    chainId: 69,
    blockExplorer: "",
    rpcUrl: "https://kovan.optimism.io",
    l2ETHGatewayAddress: '0x4200000000000000000000000000000000000006',
    l2MessengerAddress: "0x4200000000000000000000000000000000000007"
  },
  localL1: {
    name: "Local L1 Network",
    color: "#666666",
    chainId: 31337,
    blockExplorer: "",
    rpcUrl: "http://" + window.location.hostname + ":9545",
    l1ETHGatewayAddress: '0x4F53A01556Dc6f120db9a1b4caE786D5f0b5792C',
    l1MessengerAddress: '0xA6404B184Ad3f6F41b6472f02ba45f25C6A66d49'
  },
  localL2: {
    name: "Local L2 Network",
    color: "#e0bfc3",
    chainId: 420,
    blockExplorer: "",
    rpcUrl: "http://" + window.location.hostname + ":8545",
    l2ETHGatewayAddress: '0x4200000000000000000000000000000000000006',
    l2MessengerAddress: "0x4200000000000000000000000000000000000007"
  },
  rinkeby: {
    name: "Rinkeby Network",
    color: "#7003DD",
    chainId: 4,
    rpcUrl: `https://rinkeby.infura.io/v3/a1372be75cef440da1a8295acbd977b4`,
    blockExplorer: "https://rinkeby.etherscan.io/",
    l1ETHGatewayAddress: '0xBa67f68C956178CB7fd1c882f9B882487Fa28898',
    l1MessengerAddress: '0x07A5992d8bE8c271B3baa5320975b6E8d8816e34'
  },
  rinkebyL2: {
    name: "Rinkeby L2 Network",
    color: "#e0bfc3",
    chainId: 420,
    blockExplorer: "",
    rpcUrl: "http://54.161.5.63:8545",
    l2ETHGatewayAddress: '0x4200000000000000000000000000000000000006',
    l2MessengerAddress: "0x4200000000000000000000000000000000000007"
  },
};

export const L1ETHGATEWAY = [
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "_from",
        type: "address",
      },
      {
        indexed: false,
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "DepositInitiated",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "WithdrawalFinalized",
    type: "event",
  },
  {
    inputs: [],
    name: "deposit",
    outputs: [],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "_to",
        type: "address",
      },
    ],
    name: "depositTo",
    outputs: [],
    stateMutability: "payable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "finalizeWithdrawal",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
];

export const L2DEPOSITEDERC20 = [
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "owner",
        type: "address",
      },
      {
        indexed: true,
        internalType: "address",
        name: "spender",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
    ],
    name: "Approval",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "DepositFinalized",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "from",
        type: "address",
      },
      {
        indexed: true,
        internalType: "address",
        name: "to",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
    ],
    name: "Transfer",
    type: "event",
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: "address",
        name: "_from",
        type: "address",
      },
      {
        indexed: false,
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        indexed: false,
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "WithdrawalInitiated",
    type: "event",
  },
  {
    inputs: [],
    name: "DOMAIN_SEPARATOR",
    outputs: [
      {
        internalType: "bytes32",
        name: "",
        type: "bytes32",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "PERMIT_TYPEHASH",
    outputs: [
      {
        internalType: "bytes32",
        name: "",
        type: "bytes32",
      },
    ],
    stateMutability: "pure",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "owner",
        type: "address",
      },
      {
        internalType: "address",
        name: "spender",
        type: "address",
      },
    ],
    name: "allowance",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "spender",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
    ],
    name: "approve",
    outputs: [
      {
        internalType: "bool",
        name: "",
        type: "bool",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "owner",
        type: "address",
      },
    ],
    name: "balanceOf",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "decimals",
    outputs: [
      {
        internalType: "uint8",
        name: "",
        type: "uint8",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "finalizeDeposit",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [],
    name: "name",
    outputs: [
      {
        internalType: "string",
        name: "",
        type: "string",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "owner",
        type: "address",
      },
    ],
    name: "nonces",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "owner",
        type: "address",
      },
      {
        internalType: "address",
        name: "spender",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
      {
        internalType: "uint256",
        name: "deadline",
        type: "uint256",
      },
      {
        internalType: "uint8",
        name: "v",
        type: "uint8",
      },
      {
        internalType: "bytes32",
        name: "r",
        type: "bytes32",
      },
      {
        internalType: "bytes32",
        name: "s",
        type: "bytes32",
      },
    ],
    name: "permit",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [],
    name: "symbol",
    outputs: [
      {
        internalType: "string",
        name: "",
        type: "string",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [],
    name: "totalSupply",
    outputs: [
      {
        internalType: "uint256",
        name: "",
        type: "uint256",
      },
    ],
    stateMutability: "view",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "to",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
    ],
    name: "transfer",
    outputs: [
      {
        internalType: "bool",
        name: "",
        type: "bool",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "from",
        type: "address",
      },
      {
        internalType: "address",
        name: "to",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "value",
        type: "uint256",
      },
    ],
    name: "transferFrom",
    outputs: [
      {
        internalType: "bool",
        name: "",
        type: "bool",
      },
    ],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "withdraw",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
  {
    inputs: [
      {
        internalType: "address",
        name: "_to",
        type: "address",
      },
      {
        internalType: "uint256",
        name: "_amount",
        type: "uint256",
      },
    ],
    name: "withdrawTo",
    outputs: [],
    stateMutability: "nonpayable",
    type: "function",
  },
];