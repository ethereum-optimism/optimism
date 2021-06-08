import json

def get_whitelist(event, context):
  whitelist = [
    "0xD0Fb87bf4017e0A5A3bcE2eF33Bf0B95348a479E",
    "0x1383fF5A0Ef67f4BE949408838478917d87FeAc7",
  ]
  response = {
    "statusCode": 201,
    "headers": {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Credentials": True,
      "Strict-Transport-Security": "max-age=63072000; includeSubdomains; preload",
      "X-Content-Type-Options": "nosniff",
      "X-Frame-Options": "DENY",
      "X-XSS-Protection": "1; mode=block",
      "Referrer-Policy": "same-origin",
      "Permissions-Policy": "*",
    },
    "body": json.dumps(whitelist),
  }
  return response