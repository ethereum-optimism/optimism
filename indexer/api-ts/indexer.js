// indexer.ts
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
export {
  depositEndpoint,
  withdrawalEndoint
};
//# sourceMappingURL=indexer.js.map