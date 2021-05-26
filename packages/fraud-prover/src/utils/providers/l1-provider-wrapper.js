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
exports.L1ProviderWrapper = void 0;
var ethers_1 = require("ethers");
var merkletreejs_1 = require("merkletreejs");
var core_utils_1 = require("@eth-optimism/core-utils");
var L1ProviderWrapper = /** @class */ (function () {
    function L1ProviderWrapper(provider, OVM_StateCommitmentChain, OVM_CanonicalTransactionChain, OVM_ExecutionManager, l1StartOffset, l1BlockFinality) {
        this.provider = provider;
        this.OVM_StateCommitmentChain = OVM_StateCommitmentChain;
        this.OVM_CanonicalTransactionChain = OVM_CanonicalTransactionChain;
        this.OVM_ExecutionManager = OVM_ExecutionManager;
        this.l1StartOffset = l1StartOffset;
        this.l1BlockFinality = l1BlockFinality;
        this.eventCache = {};
    }
    L1ProviderWrapper.prototype.findAllEvents = function (contract, filter, fromBlock) {
        return __awaiter(this, void 0, void 0, function () {
            var cache, events, startingBlockNumber, latestL1BlockNumber, _a, _b;
            return __generator(this, function (_c) {
                switch (_c.label) {
                    case 0:
                        cache = this.eventCache[filter.topics[0]] || {
                            startingBlockNumber: fromBlock || this.l1StartOffset,
                            events: []
                        };
                        events = [];
                        startingBlockNumber = cache.startingBlockNumber;
                        return [4 /*yield*/, this.provider.getBlockNumber()];
                    case 1:
                        latestL1BlockNumber = _c.sent();
                        _c.label = 2;
                    case 2:
                        if (!(startingBlockNumber < latestL1BlockNumber)) return [3 /*break*/, 5];
                        _b = (_a = events).concat;
                        return [4 /*yield*/, contract.queryFilter(filter, startingBlockNumber, Math.min(startingBlockNumber + 2000, latestL1BlockNumber - this.l1BlockFinality))];
                    case 3:
                        events = _b.apply(_a, [_c.sent()]);
                        if (startingBlockNumber + 2000 > latestL1BlockNumber) {
                            cache.startingBlockNumber = latestL1BlockNumber;
                            cache.events = cache.events.concat(events);
                            return [3 /*break*/, 5];
                        }
                        startingBlockNumber += 2000;
                        return [4 /*yield*/, this.provider.getBlockNumber()];
                    case 4:
                        latestL1BlockNumber = _c.sent();
                        return [3 /*break*/, 2];
                    case 5:
                        this.eventCache[filter.topics[0]] = cache;
                        return [2 /*return*/, cache.events];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getStateRootBatchHeader = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var event;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this._getStateRootBatchEvent(index)];
                    case 1:
                        event = _a.sent();
                        if (!event) {
                            return [2 /*return*/];
                        }
                        return [2 /*return*/, {
                                batchIndex: event.args._batchIndex,
                                batchRoot: event.args._batchRoot,
                                batchSize: event.args._batchSize,
                                prevTotalElements: event.args._prevTotalElements,
                                extraData: event.args._extraData
                            }];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getStateRoot = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var stateRootBatchHeader, batchStateRoots;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getStateRootBatchHeader(index)];
                    case 1:
                        stateRootBatchHeader = _a.sent();
                        if (stateRootBatchHeader === undefined) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.getBatchStateRoots(index)];
                    case 2:
                        batchStateRoots = _a.sent();
                        return [2 /*return*/, batchStateRoots[index - stateRootBatchHeader.prevTotalElements.toNumber()]];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getBatchStateRoots = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var event, transaction, stateRoots;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this._getStateRootBatchEvent(index)];
                    case 1:
                        event = _a.sent();
                        if (!event) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.provider.getTransaction(event.transactionHash)];
                    case 2:
                        transaction = _a.sent();
                        stateRoots = this.OVM_StateCommitmentChain.interface.decodeFunctionData('appendStateBatch', transaction.data)[0];
                        return [2 /*return*/, stateRoots];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getStateRootBatchProof = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var batchHeader, stateRoots, elements, i, hash, leaves, tree, batchIndex, treeProof;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getStateRootBatchHeader(index)];
                    case 1:
                        batchHeader = _a.sent();
                        return [4 /*yield*/, this.getBatchStateRoots(index)];
                    case 2:
                        stateRoots = _a.sent();
                        elements = [];
                        for (i = 0; i < Math.pow(2, Math.ceil(Math.log2(stateRoots.length))); i++) {
                            if (i < stateRoots.length) {
                                elements.push(stateRoots[i]);
                            }
                            else {
                                elements.push(ethers_1.ethers.utils.keccak256('0x' + '00'.repeat(32)));
                            }
                        }
                        hash = function (el) {
                            return Buffer.from(ethers_1.ethers.utils.keccak256(el).slice(2), 'hex');
                        };
                        leaves = elements.map(function (element) {
                            return core_utils_1.fromHexString(element);
                        });
                        tree = new merkletreejs_1.MerkleTree(leaves, hash);
                        batchIndex = index - batchHeader.prevTotalElements.toNumber();
                        treeProof = tree
                            .getProof(leaves[batchIndex], batchIndex)
                            .map(function (element) {
                            return element.data;
                        });
                        return [2 /*return*/, {
                                stateRoot: stateRoots[batchIndex],
                                stateRootBatchHeader: batchHeader,
                                stateRootProof: {
                                    index: batchIndex,
                                    siblings: treeProof
                                }
                            }];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getTransactionBatchHeader = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var event;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this._getTransactionBatchEvent(index)];
                    case 1:
                        event = _a.sent();
                        if (!event) {
                            return [2 /*return*/];
                        }
                        return [2 /*return*/, {
                                batchIndex: event.args._batchIndex,
                                batchRoot: event.args._batchRoot,
                                batchSize: event.args._batchSize,
                                prevTotalElements: event.args._prevTotalElements,
                                extraData: event.args._extraData
                            }];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getBatchTransactions = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var event, emGasLimit, transaction, transactions, txdata, shouldStartAtBatch, totalElementsToAppend, numContexts, nextTxPointer, i, contextPointer, context_1, j, txDataLength, txData;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this._getTransactionBatchEvent(index)];
                    case 1:
                        event = _a.sent();
                        if (!event) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.OVM_ExecutionManager.getMaxTransactionGasLimit()];
                    case 2:
                        emGasLimit = _a.sent();
                        return [4 /*yield*/, this.provider.getTransaction(event.transactionHash)];
                    case 3:
                        transaction = _a.sent();
                        if (event.isSequencerBatch) {
                            transactions = [];
                            txdata = core_utils_1.fromHexString(transaction.data);
                            shouldStartAtBatch = ethers_1.BigNumber.from(txdata.slice(4, 9));
                            totalElementsToAppend = ethers_1.BigNumber.from(txdata.slice(9, 12));
                            numContexts = ethers_1.BigNumber.from(txdata.slice(12, 15));
                            nextTxPointer = 15 + 16 * numContexts.toNumber();
                            for (i = 0; i < numContexts.toNumber(); i++) {
                                contextPointer = 15 + 16 * i;
                                context_1 = {
                                    numSequencedTransactions: ethers_1.BigNumber.from(txdata.slice(contextPointer, contextPointer + 3)),
                                    numSubsequentQueueTransactions: ethers_1.BigNumber.from(txdata.slice(contextPointer + 3, contextPointer + 6)),
                                    ctxTimestamp: ethers_1.BigNumber.from(txdata.slice(contextPointer + 6, contextPointer + 11)),
                                    ctxBlockNumber: ethers_1.BigNumber.from(txdata.slice(contextPointer + 11, contextPointer + 16))
                                };
                                for (j = 0; j < context_1.numSequencedTransactions.toNumber(); j++) {
                                    txDataLength = ethers_1.BigNumber.from(txdata.slice(nextTxPointer, nextTxPointer + 3));
                                    txData = txdata.slice(nextTxPointer + 3, nextTxPointer + 3 + txDataLength.toNumber());
                                    transactions.push({
                                        transaction: {
                                            blockNumber: context_1.ctxBlockNumber.toNumber(),
                                            timestamp: context_1.ctxTimestamp.toNumber(),
                                            gasLimit: emGasLimit,
                                            entrypoint: '0x4200000000000000000000000000000000000005',
                                            l1TxOrigin: '0x' + '00'.repeat(20),
                                            l1QueueOrigin: 0,
                                            data: core_utils_1.toHexString(txData)
                                        },
                                        transactionChainElement: {
                                            isSequenced: true,
                                            queueIndex: 0,
                                            timestamp: context_1.ctxTimestamp.toNumber(),
                                            blockNumber: context_1.ctxBlockNumber.toNumber(),
                                            txData: core_utils_1.toHexString(txData)
                                        }
                                    });
                                    nextTxPointer += 3 + txDataLength.toNumber();
                                }
                            }
                            return [2 /*return*/, transactions];
                        }
                        else {
                            return [2 /*return*/, []];
                        }
                        return [2 /*return*/];
                }
            });
        });
    };
    L1ProviderWrapper.prototype.getTransactionBatchProof = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var batchHeader, transactions, elements, i, tx, hash, leaves, tree, batchIndex, treeProof;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.getTransactionBatchHeader(index)];
                    case 1:
                        batchHeader = _a.sent();
                        return [4 /*yield*/, this.getBatchTransactions(index)];
                    case 2:
                        transactions = _a.sent();
                        elements = [];
                        for (i = 0; i < Math.pow(2, Math.ceil(Math.log2(transactions.length))); i++) {
                            if (i < transactions.length) {
                                tx = transactions[i];
                                elements.push("0x01" + ethers_1.BigNumber.from(tx.transaction.timestamp)
                                    .toHexString()
                                    .slice(2)
                                    .padStart(64, '0') + ethers_1.BigNumber.from(tx.transaction.blockNumber)
                                    .toHexString()
                                    .slice(2)
                                    .padStart(64, '0') + tx.transaction.data.slice(2));
                            }
                            else {
                                elements.push('0x' + '00'.repeat(32));
                            }
                        }
                        hash = function (el) {
                            return Buffer.from(ethers_1.ethers.utils.keccak256(el).slice(2), 'hex');
                        };
                        leaves = elements.map(function (element) {
                            return hash(element);
                        });
                        tree = new merkletreejs_1.MerkleTree(leaves, hash);
                        batchIndex = index - batchHeader.prevTotalElements.toNumber();
                        treeProof = tree
                            .getProof(leaves[batchIndex], batchIndex)
                            .map(function (element) {
                            return element.data;
                        });
                        return [2 /*return*/, {
                                transaction: transactions[batchIndex].transaction,
                                transactionChainElement: transactions[batchIndex].transactionChainElement,
                                transactionBatchHeader: batchHeader,
                                transactionProof: {
                                    index: batchIndex,
                                    siblings: treeProof
                                }
                            }];
                }
            });
        });
    };
    L1ProviderWrapper.prototype._getStateRootBatchEvent = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var events, matching, deletions, results, _loop_1, _i, matching_1, event_1;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.findAllEvents(this.OVM_StateCommitmentChain, this.OVM_StateCommitmentChain.filters.StateBatchAppended())];
                    case 1:
                        events = _a.sent();
                        if (events.length === 0) {
                            return [2 /*return*/];
                        }
                        matching = events.filter(function (event) {
                            return (event.args._prevTotalElements.toNumber() <= index &&
                                event.args._prevTotalElements.toNumber() +
                                    event.args._batchSize.toNumber() >
                                    index);
                        });
                        return [4 /*yield*/, this.findAllEvents(this.OVM_StateCommitmentChain, this.OVM_StateCommitmentChain.filters.StateBatchDeleted())];
                    case 2:
                        deletions = _a.sent();
                        results = [];
                        _loop_1 = function (event_1) {
                            var wasDeleted = deletions.some(function (deletion) {
                                return (deletion.blockNumber > event_1.blockNumber &&
                                    deletion.args._batchIndex.toNumber() ===
                                        event_1.args._batchIndex.toNumber());
                            });
                            if (!wasDeleted) {
                                results.push(event_1);
                            }
                        };
                        for (_i = 0, matching_1 = matching; _i < matching_1.length; _i++) {
                            event_1 = matching_1[_i];
                            _loop_1(event_1);
                        }
                        if (results.length === 0) {
                            return [2 /*return*/];
                        }
                        if (results.length > 2) {
                            throw new Error("Found more than one batch header for the same state root, this shouldn't happen.");
                        }
                        return [2 /*return*/, results[results.length - 1]];
                }
            });
        });
    };
    L1ProviderWrapper.prototype._getTransactionBatchEvent = function (index) {
        return __awaiter(this, void 0, void 0, function () {
            var events, event, batchSubmissionEvents, batchSubmissionEvent;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.findAllEvents(this.OVM_CanonicalTransactionChain, this.OVM_CanonicalTransactionChain.filters.TransactionBatchAppended())];
                    case 1:
                        events = _a.sent();
                        if (events.length === 0) {
                            return [2 /*return*/];
                        }
                        event = events.find(function (event) {
                            return (event.args._prevTotalElements.toNumber() <= index &&
                                event.args._prevTotalElements.toNumber() +
                                    event.args._batchSize.toNumber() >
                                    index);
                        });
                        if (!event) {
                            return [2 /*return*/];
                        }
                        return [4 /*yield*/, this.findAllEvents(this.OVM_CanonicalTransactionChain, this.OVM_CanonicalTransactionChain.filters.SequencerBatchAppended())];
                    case 2:
                        batchSubmissionEvents = _a.sent();
                        if (batchSubmissionEvents.length === 0) {
                            ;
                            event.isSequencerBatch = false;
                        }
                        else {
                            batchSubmissionEvent = batchSubmissionEvents.find(function (event) {
                                return (event.args._startingQueueIndex.toNumber() <= index &&
                                    event.args._startingQueueIndex.toNumber() +
                                        event.args._totalElements.toNumber() >
                                        index);
                            });
                            if (batchSubmissionEvent) {
                                ;
                                event.isSequencerBatch = true;
                            }
                            else {
                                ;
                                event.isSequencerBatch = false;
                            }
                        }
                        return [2 /*return*/, event];
                }
            });
        });
    };
    return L1ProviderWrapper;
}());
exports.L1ProviderWrapper = L1ProviderWrapper;
