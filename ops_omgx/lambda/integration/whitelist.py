import json

def get_whitelist(event, context):
  whitelist = [
    "0x1E7C2Ed00FaaFeD62afC9DD630ACB8C8c6C16D52",
    "0x2C12649A5A4FC61F146E0a3409f3e4c7FbeD15Dc"
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
