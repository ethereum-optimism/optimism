"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.L1ProviderWrapper = void 0;
const ethers_1 = require("ethers");
const merkletreejs_1 = require("merkletreejs");
const hex_utils_1 = require("../hex-utils");
class L1ProviderWrapper {
    constructor(provider, OVM_StateCommitmentChain, OVM_CanonicalTransactionChain, OVM_ExecutionManager, l1StartOffset, l1BlockFinality) {
        this.provider = provider;
        this.OVM_StateCommitmentChain = OVM_StateCommitmentChain;
        this.OVM_CanonicalTransactionChain = OVM_CanonicalTransactionChain;
        this.OVM_ExecutionManager = OVM_ExecutionManager;
        this.l1StartOffset = l1StartOffset;
        this.l1BlockFinality = l1BlockFinality;
        this.eventCache = {};
    }
    async findAllEvents(contract, filter, fromBlock) {
        const cache = this.eventCache[filter.topics[0]] || {
            startingBlockNumber: fromBlock || this.l1StartOffset,
            events: [],
        };
        let events = [];
        let startingBlockNumber = cache.startingBlockNumber;
        let latestL1BlockNumber = await this.provider.getBlockNumber();
        while (startingBlockNumber < latestL1BlockNumber) {
            events = events.concat(await contract.queryFilter(filter, startingBlockNumber, Math.min(startingBlockNumber + 2000, latestL1BlockNumber - this.l1BlockFinality)));
            if (startingBlockNumber + 2000 > latestL1BlockNumber) {
                cache.startingBlockNumber = latestL1BlockNumber;
                cache.events = cache.events.concat(events);
                break;
            }
            startingBlockNumber += 2000;
            latestL1BlockNumber = await this.provider.getBlockNumber();
        }
        this.eventCache[filter.topics[0]] = cache;
        return cache.events;
    }
    async getStateRootBatchHeader(index) {
        const event = await this._getStateRootBatchEvent(index);
        if (!event) {
            return;
        }
        return {
            batchIndex: event.args._batchIndex,
            batchRoot: event.args._batchRoot,
            batchSize: event.args._batchSize,
            prevTotalElements: event.args._prevTotalElements,
            extraData: event.args._extraData,
        };
    }
    async getStateRoot(index) {
        const stateRootBatchHeader = await this.getStateRootBatchHeader(index);
        if (stateRootBatchHeader === undefined) {
            return;
        }
        const batchStateRoots = await this.getBatchStateRoots(index);
        return batchStateRoots[index - stateRootBatchHeader.prevTotalElements.toNumber()];
    }
    async getBatchStateRoots(index) {
        const event = await this._getStateRootBatchEvent(index);
        if (!event) {
            return;
        }
        const transaction = await this.provider.getTransaction(event.transactionHash);
        const [stateRoots,] = this.OVM_StateCommitmentChain.interface.decodeFunctionData('appendStateBatch', transaction.data);
        return stateRoots;
    }
    async getStateRootBatchProof(index) {
        const batchHeader = await this.getStateRootBatchHeader(index);
        const stateRoots = await this.getBatchStateRoots(index);
        const elements = [];
        for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(stateRoots.length))); i++) {
            if (i < stateRoots.length) {
                elements.push(stateRoots[i]);
            }
            else {
                elements.push(ethers_1.ethers.utils.keccak256('0x' + '00'.repeat(32)));
            }
        }
        const hash = (el) => {
            return Buffer.from(ethers_1.ethers.utils.keccak256(el).slice(2), 'hex');
        };
        const leaves = elements.map((element) => {
            return hex_utils_1.fromHexString(element);
        });
        const tree = new merkletreejs_1.MerkleTree(leaves, hash);
        const batchIndex = index - batchHeader.prevTotalElements.toNumber();
        const treeProof = tree
            .getProof(leaves[batchIndex], batchIndex)
            .map((element) => {
            return element.data;
        });
        return {
            stateRoot: stateRoots[batchIndex],
            stateRootBatchHeader: batchHeader,
            stateRootProof: {
                index: batchIndex,
                siblings: treeProof,
            },
        };
    }
    async getTransactionBatchHeader(index) {
        const event = await this._getTransactionBatchEvent(index);
        if (!event) {
            return;
        }
        return {
            batchIndex: event.args._batchIndex,
            batchRoot: event.args._batchRoot,
            batchSize: event.args._batchSize,
            prevTotalElements: event.args._prevTotalElements,
            extraData: event.args._extraData,
        };
    }
    async getBatchTransactions(index) {
        const event = await this._getTransactionBatchEvent(index);
        if (!event) {
            return;
        }
        const emGasLimit = await this.OVM_ExecutionManager.getMaxTransactionGasLimit();
        const transaction = await this.provider.getTransaction(event.transactionHash);
        if (event.isSequencerBatch) {
            const transactions = [];
            const txdata = hex_utils_1.fromHexString(transaction.data);
            const shouldStartAtBatch = ethers_1.BigNumber.from(txdata.slice(4, 9));
            const totalElementsToAppend = ethers_1.BigNumber.from(txdata.slice(9, 12));
            const numContexts = ethers_1.BigNumber.from(txdata.slice(12, 15));
            let nextTxPointer = 15 + 16 * numContexts.toNumber();
            for (let i = 0; i < numContexts.toNumber(); i++) {
                const contextPointer = 15 + 16 * i;
                const context = {
                    numSequencedTransactions: ethers_1.BigNumber.from(txdata.slice(contextPointer, contextPointer + 3)),
                    numSubsequentQueueTransactions: ethers_1.BigNumber.from(txdata.slice(contextPointer + 3, contextPointer + 6)),
                    ctxTimestamp: ethers_1.BigNumber.from(txdata.slice(contextPointer + 6, contextPointer + 11)),
                    ctxBlockNumber: ethers_1.BigNumber.from(txdata.slice(contextPointer + 11, contextPointer + 16)),
                };
                for (let j = 0; j < context.numSequencedTransactions.toNumber(); j++) {
                    const txDataLength = ethers_1.BigNumber.from(txdata.slice(nextTxPointer, nextTxPointer + 3));
                    const txData = txdata.slice(nextTxPointer + 3, nextTxPointer + 3 + txDataLength.toNumber());
                    transactions.push({
                        transaction: {
                            blockNumber: context.ctxBlockNumber.toNumber(),
                            timestamp: context.ctxTimestamp.toNumber(),
                            gasLimit: emGasLimit,
                            entrypoint: '0x4200000000000000000000000000000000000005',
                            l1TxOrigin: '0x' + '00'.repeat(20),
                            l1QueueOrigin: 0,
                            data: hex_utils_1.toHexString(txData),
                        },
                        transactionChainElement: {
                            isSequenced: true,
                            queueIndex: 0,
                            timestamp: context.ctxTimestamp.toNumber(),
                            blockNumber: context.ctxBlockNumber.toNumber(),
                            txData: hex_utils_1.toHexString(txData),
                        },
                    });
                    nextTxPointer += 3 + txDataLength.toNumber();
                }
            }
            return transactions;
        }
        else {
            return [];
        }
    }
    async getTransactionBatchProof(index) {
        const batchHeader = await this.getTransactionBatchHeader(index);
        const transactions = await this.getBatchTransactions(index);
        const elements = [];
        for (let i = 0; i < Math.pow(2, Math.ceil(Math.log2(transactions.length))); i++) {
            if (i < transactions.length) {
                const tx = transactions[i];
                elements.push(`0x01${ethers_1.BigNumber.from(tx.transaction.timestamp)
                    .toHexString()
                    .slice(2)
                    .padStart(64, '0')}${ethers_1.BigNumber.from(tx.transaction.blockNumber)
                    .toHexString()
                    .slice(2)
                    .padStart(64, '0')}${tx.transaction.data.slice(2)}`);
            }
            else {
                elements.push('0x' + '00'.repeat(32));
            }
        }
        const hash = (el) => {
            return Buffer.from(ethers_1.ethers.utils.keccak256(el).slice(2), 'hex');
        };
        const leaves = elements.map((element) => {
            return hash(element);
        });
        const tree = new merkletreejs_1.MerkleTree(leaves, hash);
        const batchIndex = index - batchHeader.prevTotalElements.toNumber();
        const treeProof = tree
            .getProof(leaves[batchIndex], batchIndex)
            .map((element) => {
            return element.data;
        });
        return {
            transaction: transactions[batchIndex].transaction,
            transactionChainElement: transactions[batchIndex].transactionChainElement,
            transactionBatchHeader: batchHeader,
            transactionProof: {
                index: batchIndex,
                siblings: treeProof,
            },
        };
    }
    async _getStateRootBatchEvent(index) {
        const events = await this.findAllEvents(this.OVM_StateCommitmentChain, this.OVM_StateCommitmentChain.filters.StateBatchAppended());
        if (events.length === 0) {
            return;
        }
        const matching = events.filter((event) => {
            return (event.args._prevTotalElements.toNumber() <= index &&
                event.args._prevTotalElements.toNumber() +
                    event.args._batchSize.toNumber() >
                    index);
        });
        const deletions = await this.findAllEvents(this.OVM_StateCommitmentChain, this.OVM_StateCommitmentChain.filters.StateBatchDeleted());
        const results = [];
        for (const event of matching) {
            const wasDeleted = deletions.some((deletion) => {
                return (deletion.blockNumber > event.blockNumber &&
                    deletion.args._batchIndex.toNumber() ===
                        event.args._batchIndex.toNumber());
            });
            if (!wasDeleted) {
                results.push(event);
            }
        }
        if (results.length === 0) {
            return;
        }
        if (results.length > 2) {
            throw new Error(`Found more than one batch header for the same state root, this shouldn't happen.`);
        }
        return results[results.length - 1];
    }
    async _getTransactionBatchEvent(index) {
        const events = await this.findAllEvents(this.OVM_CanonicalTransactionChain, this.OVM_CanonicalTransactionChain.filters.TransactionBatchAppended());
        if (events.length === 0) {
            return;
        }
        const event = events.find((event) => {
            return (event.args._prevTotalElements.toNumber() <= index &&
                event.args._prevTotalElements.toNumber() +
                    event.args._batchSize.toNumber() >
                    index);
        });
        if (!event) {
            return;
        }
        const batchSubmissionEvents = await this.findAllEvents(this.OVM_CanonicalTransactionChain, this.OVM_CanonicalTransactionChain.filters.SequencerBatchAppended());
        if (batchSubmissionEvents.length === 0) {
            ;
            event.isSequencerBatch = false;
        }
        else {
            const batchSubmissionEvent = batchSubmissionEvents.find((event) => {
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
        return event;
    }
}
exports.L1ProviderWrapper = L1ProviderWrapper;
//# sourceMappingURL=l1-provider-wrapper.js.map