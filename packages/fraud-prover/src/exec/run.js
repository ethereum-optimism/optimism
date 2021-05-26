"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
exports.__esModule = true;
var ethers_1 = require("ethers");
var service_1 = require("../service");
var dotenv_1 = require("dotenv");
dotenv_1.config();
var env = process.env;
var L2_NODE_WEB3_URL = env.L2_NODE_WEB3_URL;
var L1_NODE_WEB3_URL = env.L1_NODE_WEB3_URL;
var ADDRESS_MANAGER_ADDRESS = env.ADDRESS_MANAGER_ADDRESS;
var L1_WALLET_KEY = env.L1_WALLET_KEY;
var MNEMONIC = env.MNEMONIC;
var HD_PATH = env.HD_PATH;
var RELAY_GAS_LIMIT = env.RELAY_GAS_LIMIT || '4000000';
var RUN_GAS_LIMIT = env.RUN_GAS_LIMIT || '95000000';
var POLLING_INTERVAL = env.POLLING_INTERVAL || '5000';
//const GET_LOGS_INTERVAL = env.GET_LOGS_INTERVAL || '2000'
var L2_BLOCK_OFFSET = env.L2_BLOCK_OFFSET || '1';
var L1_START_OFFSET = env.L1_BLOCK_OFFSET || '1';
var L1_BLOCK_FINALITY = env.L1_BLOCK_FINALITY || '0';
var FROM_L2_TRANSACTION_INDEX = env.FROM_L2_TRANSACTION_INDEX || '0';
var main = function () { return __awaiter(void 0, void 0, void 0, function () {
    var l2Provider, l1Provider, wallet, service;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0:
                if (!ADDRESS_MANAGER_ADDRESS) {
                    throw new Error('Must pass ADDRESS_MANAGER_ADDRESS');
                }
                if (!L1_NODE_WEB3_URL) {
                    throw new Error('Must pass L1_NODE_WEB3_URL');
                }
                if (!L2_NODE_WEB3_URL) {
                    throw new Error('Must pass L2_NODE_WEB3_URL');
                }
                l2Provider = new ethers_1.providers.JsonRpcProvider(L2_NODE_WEB3_URL);
                l1Provider = new ethers_1.providers.JsonRpcProvider(L1_NODE_WEB3_URL);
                if (L1_WALLET_KEY) {
                    wallet = new ethers_1.Wallet(L1_WALLET_KEY, l1Provider);
                }
                else if (MNEMONIC) {
                    wallet = ethers_1.Wallet.fromMnemonic(MNEMONIC, HD_PATH);
                    wallet = wallet.connect(l1Provider);
                }
                else {
                    throw new Error('Must pass one of L1_WALLET_KEY or MNEMONIC');
                }
                service = new service_1.FraudProverService({
                    l1RpcProvider: l1Provider,
                    l2RpcProvider: l2Provider,
                    addressManagerAddress: ADDRESS_MANAGER_ADDRESS,
                    l1Wallet: wallet,
                    deployGasLimit: parseInt(RELAY_GAS_LIMIT, 10),
                    runGasLimit: parseInt(RUN_GAS_LIMIT, 10),
                    fromL2TransactionIndex: parseInt(FROM_L2_TRANSACTION_INDEX, 10),
                    pollingInterval: parseInt(POLLING_INTERVAL, 10),
                    l2BlockOffset: parseInt(L2_BLOCK_OFFSET, 10),
                    l1StartOffset: parseInt(L1_START_OFFSET, 10),
                    l1BlockFinality: parseInt(L1_BLOCK_FINALITY, 10)
                });
                return [4 /*yield*/, service.start()];
            case 1:
                _a.sent();
                return [2 /*return*/];
        }
    });
}); };
exports["default"] = main;
