-- Source: https://github.com/adetante/ethereum-nginx-proxy
local cjson = require('cjson')

local function empty(s)
  return s == nil or s == ''
end

local function split(s)
  local res = {}
  local i = 1
  for v in string.gmatch(s, "([^,]+)") do
    res[i] = v
    i = i + 1
  end
  return res
end

local function contains(arr, val)
  for i, v in ipairs (arr) do
    if v == val then
      return true
    end
  end
  return false
end

-- parse conf
local blacklist, whitelist = nil
if not empty(ngx.var.jsonrpc_blacklist) then
  blacklist = split(ngx.var.jsonrpc_blacklist)
end
if not empty(ngx.var.jsonrpc_whitelist) then
  whitelist = split(ngx.var.jsonrpc_whitelist)
end

-- check conf
if blacklist ~= nil and whitelist ~= nil then
  ngx.log(ngx.ERR, 'invalid conf: jsonrpc_blacklist and jsonrpc_whitelist are both set')
  ngx.exit(ngx.HTTP_INTERNAL_SERVER_ERROR)
  return
end

-- get request content
ngx.req.read_body()

-- try to parse the body as JSON
local success, body = pcall(cjson.decode, ngx.var.request_body);
if not success then
  ngx.log(ngx.ERR, 'invalid JSON request')
  ngx.exit(ngx.HTTP_BAD_REQUEST)
  return
end

local method = body['method']
local version = body['jsonrpc']

-- check we have a method and a version
if empty(method) or empty(version) then
  ngx.log(ngx.ERR, 'no method and/or jsonrpc attribute')
  ngx.exit(ngx.HTTP_BAD_REQUEST)
  return
end

metric_sequencer_requests:inc(1, {method, ngx.var.server_name, ngx.var.status})

-- check the version is supported
if version ~= "2.0" then
  ngx.log(ngx.ERR, 'jsonrpc version not supported: ' .. version)
  ngx.exit(ngx.HTTP_INTERNAL_SERVER_ERROR)
  return
end

-- if whitelist is configured, check that the method is whitelisted
if whitelist ~= nil then
  if not contains(whitelist, method) then
    ngx.log(ngx.ERR, 'jsonrpc method is not whitelisted: ' .. method)
    ngx.exit(ngx.HTTP_FORBIDDEN)
    return
  end
end

-- if blacklist is configured, check that the method is not blacklisted
if blacklist ~= nil then
  if contains(blacklist, method) then
    ngx.log(ngx.ERR, 'jsonrpc method is blacklisted: ' .. method)
    ngx.exit(ngx.HTTP_FORBIDDEN)
    return
  end
end

return
