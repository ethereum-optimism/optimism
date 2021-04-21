"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.makeTrieFromProofs = void 0;
const merkle_patricia_tree_1 = require("merkle-patricia-tree");
const hex_utils_1 = require("./hex-utils");
const makeTrieFromProofs = (proofs) => {
    if (proofs.length === 0 ||
        proofs.every((proof) => {
            return proof.length === 0;
        })) {
        return merkle_patricia_tree_1.BaseTrie.fromProof([]);
    }
    const nodes = proofs.reduce((nodes, proof) => {
        if (proof.length > 1) {
            return nodes.concat(proof.slice(1));
        }
        else {
            return nodes;
        }
    }, [proofs[0][0]]);
    return merkle_patricia_tree_1.BaseTrie.fromProof(nodes.map((node) => {
        return hex_utils_1.fromHexString(node);
    }));
};
exports.makeTrieFromProofs = makeTrieFromProofs;
//# sourceMappingURL=trie-utils.js.map