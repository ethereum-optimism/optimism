"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.shuffle = exports.sleep = void 0;
const sleep = async (ms) => {
    return new Promise((resolve) => {
        setTimeout(resolve, ms);
    });
};
exports.sleep = sleep;
const shuffle = (array) => {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]];
    }
    return array;
};
exports.shuffle = shuffle;
//# sourceMappingURL=common.js.map