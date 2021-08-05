import json
import yaml
import pymysql
import boto3
import string
import random
import time
import requests
import redis

def watcher_getCrossDomainMessage(event, context):

  # Parse incoming event
  body = json.loads(event["body"])
  receiptHash = body.get("hash")

  # Read YML
  with open("env.yml", 'r') as ymlfile:
    config = yaml.load(ymlfile)

  # Get MySQL host and port
  endpoint = config.get('RDS_ENDPOINT')
  user = config.get('RDS_MYSQL_NAME')
  dbpassword = config.get('RDS_MYSQL_PASSWORD')
  dbname = config.get('RDS_DBNAME')

  con = pymysql.connect(endpoint, user=user, db=dbname,
                        passwd=dbpassword, connect_timeout=5)

  with con:
    try:
      cur = con.cursor()
      cur.execute("""SELECT hash, blockNumber, `from`, `to`, timestamp, crossDomainMessage, crossDomainMessageFinalize, fastRelay, crossDomainMessageEstimateFinalizedTime,
        l1Hash, l1BlockNumber, l1BlockHash, l1From, l1To
        FROM receipt WHERE hash=%s""", (receiptHash)
      )
      transactionDataRaw = cur.fetchall()[0]
      transactionData = {
        "hash": transactionDataRaw[0],
        "blockNumber": int(transactionDataRaw[1]),
        "from": transactionDataRaw[2],
        "to": transactionDataRaw[3],
        "timestamp": transactionDataRaw[4],
        "crossDomainMessage": transactionDataRaw[5],
        "crossDomainMessageFinalize": transactionDataRaw[6],
        "fastRelay": transactionDataRaw[7],
        "crossDomainMessageEstimateFinalizedTime": transactionDataRaw[8],
        "l1Hash": transactionDataRaw[9],
        "l1BlockNumber": transactionDataRaw[10],
        "l1BlockHash": transactionDataRaw[11],
        "l1From": transactionDataRaw[12],
        "l1To": transactionDataRaw[13]
      }
    except Exception as e:
      transactionData = {}

  con.close()

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
    "body": json.dumps(transactionData),
  }
  return response