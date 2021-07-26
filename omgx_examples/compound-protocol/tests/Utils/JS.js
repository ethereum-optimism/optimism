"use strict";

function dfn(val, def) {
  return isFinite(val) ? val : def;
}

function last(elems) {
  return Array.isArray(elems) ? elems[elems.length - 1] : elems;
}

function lookup(obj, path = []) {
  return Array.isArray(path) ? path.reduce((a, k) => a[k], obj) : obj[path];
}

function select(obj, keys = []) {
  return keys.reduce((a, k) => (a[k] = obj[k], a), {})
}

module.exports = {
  dfn,
  last,
  lookup,
  select
};
