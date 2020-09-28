"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.remove0x = exports.makeAddress = exports.makeHexString = void 0;
exports.makeHexString = (byte, len) => {
    return '0x' + byte.repeat(len);
};
exports.makeAddress = (byte) => {
    return exports.makeHexString(byte, 20);
};
exports.remove0x = (str) => {
    if (str.startsWith('0x')) {
        return str.slice(2);
    }
    else {
        return str;
    }
};
//# sourceMappingURL=byte-utils.js.map