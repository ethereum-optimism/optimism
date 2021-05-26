"use strict";
exports.__esModule = true;
exports.makeTrieFromProofs = void 0;
var merkle_patricia_tree_1 = require("merkle-patricia-tree");
var core_utils_1 = require("@eth-optimism/core-utils");
var makeTrieFromProofs = function (proofs) {
    if (proofs.length === 0 ||
        proofs.every(function (proof) {
            return proof.length === 0;
        })) {
        return merkle_patricia_tree_1.BaseTrie.fromProof([]);
    }
    var nodes = proofs.reduce(
    // tslint:disable-next-line
    function (nodes, proof) {
        if (proof.length > 1) {
            return nodes.concat(proof.slice(1));
        }
        else {
            return nodes;
        }
    }, [proofs[0][0]]);
    return merkle_patricia_tree_1.BaseTrie.fromProof(nodes.map(function (node) {
        return core_utils_1.fromHexString(node);
    }));
};
exports.makeTrieFromProofs = makeTrieFromProofs;
