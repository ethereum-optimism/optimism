import json
import time
import hmac
import hashlib
import base64
import os
from urllib.parse import urlencode, quote

def get_string_to_sign(params):
    """Sort and join parameters for signing"""
    sorted_params = sorted(params.items())
    return '&'.join([f"{k}={v}" for k, v in sorted_params if v])

def generate_signature(timestamp, http_method, request_path, secret_key):
    """Generate HMAC SHA256 signature"""
    signature_string = f"{timestamp}{http_method}{request_path}"

    signature = hmac.new(
        secret_key.encode(),
        signature_string.encode(),
        hashlib.sha256
    ).digest()

    return quote(base64.b64encode(signature).decode())

def generate_signed_url(event, context):
    """Lambda handler to generate signed Alchemy Pay URL"""
    try:
        # Parse request body
        body = json.loads(event.get('body', '{}'))

        # Get environment variables
        app_id = os.environ['APP_ID']
        secret_key = os.environ['SECRET_KEY']
        base_url = os.environ['BASE_URL']

        # Required parameters
        http_method = "GET"
        request_path = "/index/rampPageBuy"
        timestamp = str(int(time.time() * 1000))

        # Default parameters
        default_params = {
            'crypto': 'USDT',
            'fiat': 'USD',
            'network': 'ETH',
            'merchantOrderNo': str(int(time.time() * 1000)) + str(hash(time.time()))[:8],
            'appId': app_id,
            'timestamp': timestamp
        }

        # Merge default params with body, letting body override defaults
        params = {**default_params, **body}

        # Generate signature
        raw_data = get_string_to_sign(params)
        request_path_with_params = f"{request_path}?{raw_data}"
        signature = generate_signature(
            timestamp,
            http_method,
            request_path_with_params,
            secret_key
        )

        # Generate final URL
        final_url = f"{base_url}?{raw_data}&sign={signature}"

        return {
            'statusCode': 200,
            'headers': {
                'Content-Type': 'application/json',
                'Access-Control-Allow-Origin': '*'
            },
            'body': json.dumps({
                'url': final_url,
                'timestamp': timestamp
            })
        }

    except Exception as e:
        return {
            'statusCode': 500,
            'headers': {
                'Content-Type': 'application/json',
                'Access-Control-Allow-Origin': '*'
            },
            'body': json.dumps({
                'error': str(e)
            })
        }
