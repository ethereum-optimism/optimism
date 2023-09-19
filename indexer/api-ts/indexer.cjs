"use strict";
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// indexer.ts
var indexer_exports = {};
__export(indexer_exports, {
  depositEndpoint: () => depositEndpoint,
  withdrawalEndoint: () => withdrawalEndoint
});
module.exports = __toCommonJS(indexer_exports);
var createQueryString = ({ cursor, limit }) => {
  if (cursor === void 0 && limit === void 0) {
    return "";
  }
  const queries = [];
  if (cursor) {
    queries.push(`cursor=${cursor}`);
  }
  if (limit) {
    queries.push(`limit=${limit}`);
  }
  return `?${queries.join("&")}`;
};
var depositEndpoint = ({ baseUrl = "", address, cursor, limit }) => {
  return [baseUrl, "deposits", `${address}${createQueryString({ cursor, limit })}`].join("/");
};
var withdrawalEndoint = ({ baseUrl = "", address, cursor, limit }) => {
  return [baseUrl, "withdrawals", `${address}${createQueryString({ cursor, limit })}`].join("/");
};
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  depositEndpoint,
  withdrawalEndoint
});
//# sourceMappingURL=indexer.cjs.map